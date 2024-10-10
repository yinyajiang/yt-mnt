package downloader

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"

	"log"

	"github.com/yinyajiang/yt-mnt/pkg/common"
	"github.com/yinyajiang/yt-mnt/pkg/ies"
)

type ProgressSink func(total, downloaded, speed, eta int64, percent float64, videoDuration int64)

type RefillInfo struct {
	Title     *string
	Thumbnail *string
	Duration  *int64
}

type Downloader interface {
	Name() string
	SupportedIE() []string
	IsNeedFormat() bool
	/*
		err==nil 表示成功，否则表示失败
		ok==true 表示是可恢复的失败
	*/
	Download(ctx context.Context, opt DownloadOptions, sink ProgressSink) (ok bool, err error)
	Delete(opts DeleteOptions, deleteFile bool)
	ChangeFileTitle(opt DownloadOptions, title string) error
}

type DownloadOptions struct {
	URL string

	MainDownloadFormat  ies.Format
	AudioDownloadFormat ies.Format
	DownloadedSize      int64
	DownloadPercent     float64
	DownloadFileDir     string

	Subtitle            string
	IsDownloadThumbnail bool
	IsOriginalSubtitle  bool

	//in out params
	DownloadFileStem *string
	DownloadFileExt  *string
	DownloaderData   *string
	Quality          *string

	RefillInfo
}

type DeleteOptions struct {
	DownloadFileDir  string
	DownloadFileStem string
	DownloadFileExt  string
	DownloaderData   string
	HasAudioFormat   bool
}

func (opt *DeleteOptions) FilePath() string {
	if opt.DownloadFileExt != "" && !strings.HasPrefix(opt.DownloadFileExt, ".") {
		opt.DownloadFileExt = "." + opt.DownloadFileExt
	}
	return filepath.Join(opt.DownloadFileDir, opt.DownloadFileStem+opt.DownloadFileExt)
}

func (opt *DownloadOptions) FilePath() string {
	if *opt.DownloadFileExt != "" && !strings.HasPrefix(*opt.DownloadFileExt, ".") {
		*opt.DownloadFileExt = "." + *opt.DownloadFileExt
	}
	return filepath.Join(opt.DownloadFileDir, *opt.DownloadFileStem+*opt.DownloadFileExt)
}

func (opt *DownloadOptions) SetExistFileName(name string) {
	if name == "" {
		return
	}
	name = filepath.Base(name)

	i := strings.Index(name, ".")
	if i == -1 || i == 0 || i == len(name)-1 {
		return
	}
	opt.SetExt(name[i:])
	opt.SetStem(name[:i])
}

func (opt *DownloadOptions) SetExt(ext string) {
	if ext == "" {
		return
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	*opt.DownloadFileExt = ext
}

func (opt *DownloadOptions) SetStem(stem string) {
	if stem == "" {
		return
	}
	*opt.DownloadFileStem = common.ReplaceWrongFileChars(stem)

	path := opt.FilePath()
	maxLength := 500
	if strings.EqualFold(runtime.GOOS, "windows") {
		maxLength = 255
	}
	if len(path) > maxLength {
		if len := maxLength - len(opt.DownloadFileDir); len > 0 {
			*opt.DownloadFileStem = (*opt.DownloadFileStem)[:len]
		}
	}
}

var _downloaders = make(map[string]Downloader)

func Regist(d Downloader) {
	_downloaders[d.Name()] = &MiddleDownloader{
		d: d,
	}
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
			if suportedIE == "*" || strings.EqualFold(suportedIE, ie) {
				return d
			}
			if strings.HasPrefix(suportedIE, "!") && !strings.EqualFold(ie, suportedIE[1:]) {
				return d
			}
		}
	}
	return nil
}
