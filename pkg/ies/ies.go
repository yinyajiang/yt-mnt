package ies

import (
	"errors"
	"github.com/yinyajiang/yt-mnt/model"
)

type InfoExtractor interface {
	Extract(url string) (*model.MediaEntry, error)
	Update(update *model.MediaEntry) error
	IsMatched(url string) bool
	Name() string
}

var (
	_ies = make(map[string]InfoExtractor)
)

func Regist(ie InfoExtractor) {
	_ies[ie.Name()] = ie
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
