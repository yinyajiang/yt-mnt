package ies

import (
	"errors"

	"github.com/yinyajiang/yt-mnt/model"
)

type LinkInfo struct {
	LinkID    string
	MediaID   string
	MediaType int
}

type NextPage struct {
	NextPageID    string
	HintPageCount int64
	IsEnd         bool
}

type InfoExtractor interface {
	Parse(link string) (*model.MediaEntry, error)
	ExtractPage(linkInfo LinkInfo, nextPage *NextPage) ([]*model.MediaEntry, error)
	UpdateMedia(update *model.MediaEntry) error
	IsMatched(url string) bool
	Name() string
}

var (
	_ies = make(map[string]InfoExtractor)
)

func Regist(ie InfoExtractor) {
	_ies[ie.Name()] = &cacheInfoExtractor{
		ie:    ie,
		cache: make([]cacheInfo, 0),
	}
}

func GetIE(hints ...string) (InfoExtractor, error) {
	for _, name := range hints {
		if name == "" {
			continue
		}
		pro, ok := _ies[name]
		if ok {
			return pro, nil
		}
	}
	for _, url := range hints {
		if url == "" {
			continue
		}
		for _, ie := range _ies {
			if ie.IsMatched(url) {
				return ie, nil
			}
		}
	}
	return nil, errors.New("no matched IE")
}
