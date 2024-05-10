package downloader

import (
	"context"
	"strings"

	"github.com/yinyajiang/yt-mnt/pkg/ies"

	"log"
)

type ProgressSink func(total, downloaded, speed int64, percent float64)

type DownloadOptions struct {
	URL            string
	Quality        string
	DownloadDir    string
	DownloadFile   string
	DownloadFormat ies.Format
	DownloaderData *string
}

type Downloader interface {
	Name() string
	SupportedIE() []string
	IsNeedFormat() bool
	/*
		err==nil 表示成功，否则表示失败
		ok==true 表示是可恢复的失败
	*/
	Download(ctx context.Context, opt DownloadOptions, sink ProgressSink, stageSaver ...DownloaderStageSaver) (ok bool, err error)
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
