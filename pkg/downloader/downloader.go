package downloader

import (
	"context"
	"github.com/yinyajiang/yt-mnt/model"
	"strings"

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

func GetByName(downloader string) Downloader {
	if downloader == "" {
		log.Panic("downloader name is empty")
		return nil
	}
	if d, ok := _downloaders[downloader]; ok {
		return d
	}
	log.Panicf("downloader %s not found", downloader)
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
