package downloader

import (
	"context"
	"strings"

	"github.com/yinyajiang/yt-mnt/model"

	"log"
)

type ProgressSink func(total, downloaded, speed int64, percent float64)

type Downloader interface {
	Name() string
	SupportedIE() []string
	Download(ctx context.Context, entry *model.FeedEntry, sink ProgressSink) (ok bool, err error)
}

var _downloaders = make(map[string]Downloader)

func Regist(d Downloader) {
	_downloaders[d.Name()] = d
}

func GetByName(name string) Downloader {
	if name == "" {
		log.Panic("downloader name is empty")
		return nil
	}
	if d, ok := _downloaders[name]; ok {
		return d
	}
	log.Panicf("downloader %s not found", name)
	return nil
}

func GetByIE(ie string) Downloader {
	for _, d := range _downloaders {
		for _, suportedIE := range d.SupportedIE() {
			if strings.EqualFold(suportedIE, ie) {
				return d
			}
		}
	}
	log.Panicf("downloader for %s not found", ie)
	return nil
}
