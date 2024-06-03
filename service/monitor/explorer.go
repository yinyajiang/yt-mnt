package monitor

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/mathutil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/google/uuid"
	"github.com/yinyajiang/yt-mnt/pkg/ies"
	"golang.org/x/exp/maps"
)

type selectedType struct {
	mediaType int
	allPage   bool
}

type Explorer struct {
	ie        ies.InfoExtractor
	rootToken ies.RootToken
	rootInfo  ies.MediaEntry

	nextToken ies.NextPageToken

	allPage   []*ies.MediaEntry
	pageIndex int

	selecteds     []int
	selectedTypes []selectedType
	isPlain       bool

	url  string
	time time.Time

	_uuid string
}

const (
	IndexCurrentPage = -iota - 1
	IndexExplored
	IndexAllPage
	IndexRoot
	IndexUser
)

func newExplorer() *Explorer {
	var explorer Explorer
	explorer.time = time.Now()
	explorer._uuid = uuid.New().String()
	return &explorer
}

func (e *Explorer) IsValid() bool {
	return (e.ie != nil) &&
		(e.rootToken.MediaID != "" || e.rootToken.LinkID != "") &&
		(e.rootToken.MediaType == ies.MediaTypePlaylist ||
			e.rootToken.MediaType == ies.MediaTypePlaylistGroup ||
			e.rootToken.MediaType == ies.MediaTypeUser)
}

func (e *Explorer) IsEnd() bool {
	return e.nextToken.IsEnd
}

func (e *Explorer) Root() *ies.MediaEntry {
	root := e.rootInfo
	return &root
}

func (e *Explorer) CreateTime() time.Time {
	return e.time
}

func (e *Explorer) URL() string {
	return e.url
}

func (e *Explorer) RootMediaType() int {
	return e.rootInfo.MediaType
}

func (e *Explorer) Tree() *ies.MediaEntry {
	root := e.rootInfo
	root.Entries = e.allPage
	return &root
}

func (e *Explorer) Page() []*ies.MediaEntry {
	if e.pageIndex < 0 || len(e.allPage) == 0 || e.pageIndex >= len(e.allPage) {
		return nil
	}
	return e.allPage[e.pageIndex:]
}

func (e *Explorer) AllPage(loadLeft ...bool) []*ies.MediaEntry {
	if len(loadLeft) > 0 && loadLeft[0] {
		e.ExploreNextAll()
	}
	return e.allPage
}

func (e *Explorer) ScrollToTop() {
	e.pageIndex = 0
}

func (e *Explorer) PageCount() int {
	if e.pageIndex < 0 || len(e.allPage) == 0 || e.pageIndex >= len(e.allPage) {
		return 0
	}
	return len(e.allPage) - e.pageIndex
}

func (e *Explorer) ResetSelected() {
	e.selecteds = []int{}
}

func (e *Explorer) Select(indexs ...int) {
	for _, index := range indexs {
		if index >= len(e.allPage) {
			continue
		}
		if index < 0 {
			if (len(e.selecteds) > 0 && e.selecteds[0] > index) || len(e.selecteds) == 0 {
				e.selecteds = []int{index}
			}
		} else {
			e.selecteds = append(e.selecteds, index)
		}
	}
}

func (e *Explorer) SelectMediaType(mediaType int, allPage bool) {
	e.selectedTypes = append(e.selectedTypes, selectedType{
		mediaType: mediaType,
		allPage:   allPage,
	})
}

func (e Explorer) User() (*ies.MediaEntry, error) {
	userExplorer, err := e.GetUserExplorer()
	if err != nil {
		return nil, err
	}
	return userExplorer.Root(), nil
}

func (e Explorer) Item(index int, enableConvertUser bool) (*ies.MediaEntry, error) {
	if index < 0 {
		switch index {
		case IndexRoot:
			return e.Tree(), nil
		case IndexUser:
			if e.rootInfo.MediaType == ies.MediaTypeUser {
				return e.Tree(), nil
			}
			if enableConvertUser {
				err := e.ConvertToUserExplorer()
				if err != nil {
					return nil, err
				}
				return e.Tree(), nil
			} else {
				userExplorer, err := e.GetUserExplorer()
				if err != nil {
					return nil, err
				}
				return userExplorer.Item(IndexUser, false)
			}
		default:
			return nil, errors.New("invalid item index")
		}
	} else {
		if index >= len(e.allPage) {
			return nil, errors.New("index out of range")
		}
		return e.allPage[index], nil
	}
}

func (e *Explorer) GetUserExplorer() (*Explorer, error) {
	userExplorer := Explorer{
		ie:        e.ie,
		rootToken: e.rootToken,
		rootInfo:  e.rootInfo,
	}
	err := userExplorer.ie.ConvertToUserRoot(&userExplorer.rootToken, &userExplorer.rootInfo)
	if err != nil {
		return nil, err
	}
	return &userExplorer, nil
}

func (e *Explorer) ConvertToUserExplorer() error {
	userExplorer, err := e.GetUserExplorer()
	if err != nil {
		return err
	}
	*e = *userExplorer
	return nil
}

func (e *Explorer) Selected(enableConvertUser bool) ([]*ies.MediaEntry, error) {
	// 筛选类型
	for _, selectedType := range e.selectedTypes {
		if selectedType.allPage {
			e.ExploreNextAll()
		}
		for i, entry := range e.allPage {
			if entry.MediaType == selectedType.mediaType {
				e.Select(i)
			}
		}
	}

	if len(e.selecteds) != 0 {
		e.selecteds = slice.Unique(e.selecteds)
	}

	selected := make([]*ies.MediaEntry, 0)
	for _, index := range e.selecteds {
		switch index {
		case IndexCurrentPage:
			return e.Page(), nil
		case IndexExplored:
			return e.allPage, nil
		case IndexAllPage:
			e.ExploreNextAll()
			return e.allPage, nil
		case IndexRoot, IndexUser:
			item, err := e.Item(index, enableConvertUser)
			if err != nil {
				return nil, err
			}
			return []*ies.MediaEntry{item}, nil
		default:
			selected = append(selected, e.allPage[index])
		}
	}
	return selected, nil
}

func (e *Explorer) Size() int {
	return len(e.allPage)
}

func (e *Explorer) ExploreNextAll() ([]*ies.MediaEntry, error) {
	for !e.IsEnd() {
		if _, err := e.ExporeNextPage(); err != nil {
			return nil, err
		}
	}
	return e.allPage, nil
}

func (e *Explorer) SetPlain() {
	e.isPlain = true
	if len(e.allPage) > 0 {
		old := e.allPage
		e.allPage = make([]*ies.MediaEntry, 0)
		plainAppend(&e.allPage, old)
	}
}

func (e *Explorer) IsPlain() bool {
	return e.isPlain
}

func (e *Explorer) ExporeNextPage() ([]*ies.MediaEntry, error) {
	if !e.IsValid() {
		return nil, errors.New("invalid explore handle")
	}
	if e.IsEnd() {
		return nil, errors.New("no more page")
	}
	page, err := e.ie.ExtractPage(&e.rootToken, &e.nextToken)
	if err != nil {
		return nil, err
	}

	before := len(e.allPage)
	if e.isPlain {
		plainAppend(&e.allPage, page)
	} else {
		e.allPage = append(e.allPage, page...)
	}
	e.pageIndex = before

	return e.Page(), nil
}

func (e *Explorer) Close() {
	e.nextToken.IsEnd = true
}

func (e *Explorer) uuid() string {
	return e._uuid
}

func (e *Explorer) firstSelectedIndex() int {
	if len(e.selecteds) == 0 {
		return math.MaxInt
	}
	return e.selecteds[0]
}

type ExplorerCacher struct {
	lock         sync.RWMutex
	explorersMap map[string]*Explorer
	cacheCount   int
}

func NewExplorerCacher(cacheCount_ ...int) *ExplorerCacher {
	cacheCount := 9999
	if len(cacheCount_) > 0 && cacheCount_[0] >= 1 {
		cacheCount = cacheCount_[0]
	}
	return &ExplorerCacher{
		lock:         sync.RWMutex{},
		cacheCount:   cacheCount,
		explorersMap: map[string]*Explorer{},
	}
}

func (c *ExplorerCacher) Put(explorer *Explorer) (handle string) {
	if explorer == nil {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.explorersMap[explorer.uuid()] = explorer
	if len(c.explorersMap) > c.cacheCount {
		delete(c.explorersMap, c.first().uuid())
	}
	return explorer.uuid()
}

func (c *ExplorerCacher) Get(handle string) *Explorer {
	if handle == "" {
		return nil
	}
	c.lock.RLock()
	defer c.lock.RUnlock()
	if explorer, ok := c.explorersMap[handle]; ok {
		return explorer
	}
	return nil
}

func (c *ExplorerCacher) IsContain(handle string) bool {
	if handle == "" {
		return false
	}
	c.lock.RLock()
	defer c.lock.RUnlock()
	_, ok := c.explorersMap[handle]
	return ok
}

func (c *ExplorerCacher) Delete(handle string) {
	if handle == "" {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.explorersMap, handle)
}

func (c *ExplorerCacher) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.explorersMap = map[string]*Explorer{}
}

func (c *ExplorerCacher) first() *Explorer {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if len(c.explorersMap) == 0 {
		return nil
	}
	return mathutil.MinBy(maps.Values(c.explorersMap), func(v, min *Explorer) bool {
		return v.CreateTime().Before(min.CreateTime())
	})
}
