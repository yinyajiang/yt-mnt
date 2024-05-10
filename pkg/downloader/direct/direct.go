package direct

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/yinyajiang/yt-mnt/pkg/common"
	"github.com/yinyajiang/yt-mnt/pkg/downloader"
	instagram "github.com/yinyajiang/yt-mnt/pkg/ies/instagram"
)

func Name() string {
	return "direct"
}

func init() {
	downloader.Regist(&DirectDownloader{})
}

type DirectDownloader struct {
}

func (d *DirectDownloader) Download(ctx context.Context, opt downloader.DownloadOptions, sink downloader.ProgressSink, stageSaver ...downloader.DownloaderStageSaver) (ok bool, err error) {
	if opt.DownloadFormat.URL == "" {
		return false, errors.New("no formats available for direct download")
	}
	ext := common.URLDotExt(opt.DownloadFormat.URL)

	opt.DownloadFile = filepath.Join(opt.DownloadDir, time.Now().Format("20060102")+ext)

	ok = true
	err = downloadFile(ctx, opt.DownloadFormat.URL, opt.DownloadFile, sink)
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

func (d *DirectDownloader) IsAcceptNoPlain() bool {
	return true
}

func (m *DirectDownloader) IsNeedFormat() bool {
	return true
}
