package monitor

import (
	"crypto/md5"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/yinyajiang/yt-mnt/pkg/ies"
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
}

const (
	IndexCurrentPage = -iota - 1
	IndexExplored
	IndexAllPage
	IndexRoot
	IndexUser
)

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

func (e *Explorer) Selected() ([]*ies.MediaEntry, error) {
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
		case IndexRoot:
			return []*ies.MediaEntry{e.Tree()}, nil
		case IndexUser:
			if e.rootToken.MediaType == ies.MediaTypeUser && e.rootInfo.MediaType == ies.MediaTypeUser {
				return []*ies.MediaEntry{e.Tree()}, nil
			}
			*e = Explorer{
				ie:        e.ie,
				rootToken: e.rootToken,
				rootInfo:  e.rootInfo,
				selecteds: e.selecteds,
			}
			err := e.ie.ConvertToUserRoot(&e.rootToken, &e.rootInfo)
			if err != nil {
				return nil, err
			}
			return []*ies.MediaEntry{e.Tree()}, nil
		default:
			selected = append(selected, e.allPage[index])
		}
	}
	return selected, nil
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

func (e *Explorer) firstSelectedIndex() int {
	if len(e.selecteds) == 0 {
		return math.MaxInt
	}
	return e.selecteds[0]
}

type ExplorerCacher struct {
	lock       sync.RWMutex
	explorers  []*Explorer
	cacheCount int
}

func NewExplorerCacher(cacheCount_ ...int) *ExplorerCacher {
	cacheCount := 10
	if len(cacheCount_) > 0 {
		cacheCount = cacheCount_[0]
	}
	if cacheCount < 1 {
		cacheCount = 10
	}
	return &ExplorerCacher{
		lock:       sync.RWMutex{},
		explorers:  make([]*Explorer, 0),
		cacheCount: cacheCount,
	}
}

func (c *ExplorerCacher) Cache(explorer *Explorer) (handle string) {
	if explorer == nil {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	for i, explorer := range c.explorers {
		if !strings.EqualFold(urlToHandle(explorer.URL()), handle) {
			continue
		}
		explorer.Close()
		if len(c.explorers) == 1 {
			c.explorers = make([]*Explorer, 0)
		} else {
			c.explorers = append(c.explorers[:i], c.explorers[i+1:]...)
		}
		break
	}
	if len(c.explorers) > c.cacheCount {
		c.explorers[0].Close()
		c.explorers = c.explorers[1:]
	}

	c.explorers = append(c.explorers, explorer)
	return urlToHandle(explorer.URL())
}

func (c *ExplorerCacher) Get(handle string) *Explorer {
	if handle == "" {
		return nil
	}
	c.lock.RLock()
	defer c.lock.RUnlock()
	for _, explorer := range c.explorers {
		if strings.EqualFold(urlToHandle(explorer.URL()), handle) {
			return explorer
		}
	}
	return nil
}

func (c *ExplorerCacher) Delete(handle string) {
	if handle == "" {
		return
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	for i, explorer := range c.explorers {
		if !strings.EqualFold(urlToHandle(explorer.URL()), handle) {
			continue
		}
		explorer.Close()
		if len(c.explorers) == 1 {
			c.explorers = make([]*Explorer, 0)
		} else {
			c.explorers = append(c.explorers[:i], c.explorers[i+1:]...)
		}
		return
	}
}
func urlToHandle(url string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(url)))
}
