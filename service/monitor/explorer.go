package monitor

import (
	"errors"

	"github.com/yinyajiang/yt-mnt/pkg/ies"
)

type Explorer struct {
	ie       string
	root     ies.MediaEntry
	allNext  []*ies.MediaEntry
	nextPage ies.NextPage

	pageIndex int

	selected  []int
	_is_plain bool
}

func (e *Explorer) IsValid() bool {
	linkInfo := e.root.LinkInfo()
	return (e.ie != "") &&
		(linkInfo.MediaID != "" || linkInfo.LinkID != "") &&
		(linkInfo.MediaType == ies.MediaTypePlaylist ||
			linkInfo.MediaType == ies.MediaTypePlaylistGroup ||
			linkInfo.MediaType == ies.MediaTypeUser)
}

func (e *Explorer) IsEnd() bool {
	return e.nextPage.IsEnd
}

func (e *Explorer) Root() *ies.MediaEntry {
	root := e.root
	return &root
}

func (e *Explorer) Tree() *ies.MediaEntry {
	root := e.root
	root.Entries = e.allNext
	return &root
}

func (e *Explorer) Page() []*ies.MediaEntry {
	if e.pageIndex < 0 || len(e.allNext) == 0 || e.pageIndex >= len(e.allNext) {
		return nil
	}
	return e.allNext[e.pageIndex:]
}

func (e *Explorer) SetPageIndexToHead() {
	e.pageIndex = 0
}

func (e *Explorer) PageCount() int {
	if e.pageIndex < 0 || len(e.allNext) == 0 || e.pageIndex >= len(e.allNext) {
		return 0
	}
	return len(e.allNext) - e.pageIndex
}

func (e *Explorer) ResetSelected() {
	e.selected = []int{}
}

func (e *Explorer) Select(indexs ...int) {
	for _, index := range indexs {
		if index >= len(e.allNext) {
			continue
		}
		if index == -1 {
			e.selected = []int{-1}
			return
		}
		e.selected = append(e.selected, index)
	}
}

func (e *Explorer) SelectRoot() {
	e.selected = []int{-1}
}

func (e *Explorer) Selected(loadAllIfSelectAll ...bool) []*ies.MediaEntry {
	selected := make([]*ies.MediaEntry, 0)

	if len(loadAllIfSelectAll) != 0 && loadAllIfSelectAll[0] && e.isSelectedAll() {
		e.ExplorerAll()
	}

	for _, index := range e.selected {
		if index == -1 {
			return []*ies.MediaEntry{e.Tree()}
		}
		if index >= len(e.allNext) {
			continue
		}
		selected = append(selected, e.allNext[index])
	}
	return selected
}

func (e *Explorer) ExplorerAll() ([]*ies.MediaEntry, error) {
	for !e.IsEnd() {
		if _, err := e.ExporeNextPage(); err != nil {
			return nil, err
		}
	}
	return e.allNext, nil
}

func (e *Explorer) SetPlain() {
	e._is_plain = true
	if len(e.allNext) > 0 {
		old := e.allNext
		e.allNext = make([]*ies.MediaEntry, 0)
		plainAppend(&e.allNext, old)
	}
}

func (e *Explorer) IsPlain() bool {
	return e._is_plain
}

func (e *Explorer) ExporeNextPage() ([]*ies.MediaEntry, error) {
	if !e.IsValid() {
		return nil, errors.New("invalid explore handle")
	}
	if e.IsEnd() {
		return nil, errors.New("no more page")
	}
	ie, err := ies.GetIE(e.ie)
	if err != nil {
		return nil, err
	}
	page, err := ie.ExtractPage(e.root.LinkInfo(), &e.nextPage)
	if err != nil {
		return nil, err
	}

	before := len(e.allNext)
	if e._is_plain {
		plainAppend(&e.allNext, page)
	} else {
		e.allNext = append(e.allNext, page...)
	}
	e.pageIndex = before

	return e.Page(), nil
}

func (e *Explorer) isSelectedAll() bool {
	return len(e.selected) != 0 && e.selected[0] == -1
}
