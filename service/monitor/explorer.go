package monitor

import (
	"errors"
	"math"

	"github.com/yinyajiang/yt-mnt/pkg/ies"
)

type Explorer struct {
	ie        ies.InfoExtractor
	rootToken ies.RootToken
	rootInfo  ies.MediaEntry

	nextToken ies.NextPageToken

	allPage   []*ies.MediaEntry
	pageIndex int

	selected []int
	is_plain bool
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

func (e *Explorer) AllPage() []*ies.MediaEntry {
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
	e.selected = []int{}
}

func (e *Explorer) Select(indexs ...int) {
	for _, index := range indexs {
		if index >= len(e.allPage) {
			continue
		}
		if index < 0 {
			if (len(e.selected) > 0 && e.selected[0] > index) || len(e.selected) == 0 {
				e.selected = []int{index}
			}
		} else {
			e.selected = append(e.selected, index)
		}

	}
}

func (e *Explorer) Selected() ([]*ies.MediaEntry, error) {
	selected := make([]*ies.MediaEntry, 0)
	for _, index := range e.selected {
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
				selected:  e.selected,
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
	e.is_plain = true
	if len(e.allPage) > 0 {
		old := e.allPage
		e.allPage = make([]*ies.MediaEntry, 0)
		plainAppend(&e.allPage, old)
	}
}

func (e *Explorer) IsPlain() bool {
	return e.is_plain
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
	if e.is_plain {
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
	if len(e.selected) == 0 {
		return math.MaxInt
	}
	return e.selected[0]
}
