package direct

import (
	"context"
	"errors"
	"github.com/yinyajiang/yt-mnt/model"
	"github.com/yinyajiang/yt-mnt/pkg/common"
	"github.com/yinyajiang/yt-mnt/pkg/downloader"
	instagram "github.com/yinyajiang/yt-mnt/pkg/ies/instagram"
	"path/filepath"
)

func Name() string {
	return "direct"
}

func init() {
	downloader.Regist(&DirectDownloader{})
}

type DirectDownloader struct {
}

func (d *DirectDownloader) Download(ctx context.Context, entry *model.FeedEntry, sink downloader.ProgressSink) (ok bool, err error) {
	if len(entry.Formats) == 0 {
		return false, errors.New("no formats available for direct download")
	}
	index := common.SelectFormat(entry.Formats, entry.DownloadQuality)
	f := entry.Formats[index]

	ext := common.URLDotExt(f.URL)
	entry.DownloadFile = filepath.Join(entry.DownloadDir, entry.UpdatedAt.Format("20060102")+ext)

	ok = true
	err = downloadFile(ctx, f.URL, entry.DownloadFile, sink)
	return
}

func (d *DirectDownloader) SupportedIE() []string {
	return []string{
		instagram.Name(),
	}
}

func (d *DirectDownloader) Name() string {
	return Name()
}
