package ies

import (
	"errors"
	"time"
)

type RootToken struct {
	LinkID    string
	MediaID   string
	MediaType int
}

type NextPageToken struct {
	NextPageID    string
	HintPageCount int64
	IsEnd         bool
}

type ParseOptions struct {
}

type InfoExtractor interface {
	ParseRoot(link string, options ...ParseOptions) (*MediaEntry, *RootToken, error)
	ConvertToUserRoot(rootToken *RootToken, rootInfo *MediaEntry) error
	ExtractPage(rootToken *RootToken, nextPage *NextPageToken) ([]*MediaEntry, error)
	ExtractAllAfterTime(parentMediaID string, afterTime time.Time) ([]*MediaEntry, error)
	IsMatched(link string) bool
	Name() string
	Init() error
}

var (
	_ies = make(map[string]InfoExtractor)
)

func Regist(ie InfoExtractor) {
	_ies[ie.Name()] = &middleInfoExtractor{
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

func InitIE() error {
	for _, ie := range _ies {
		if err := ie.Init(); err != nil {
			return err
		}
	}
	return nil
}
