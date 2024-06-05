package direct

import (
	"context"
	"errors"
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

func (d *DirectDownloader) Download(ctx context.Context, opt downloader.DownloadOptions, sink downloader.ProgressSink) (ok bool, err error) {
	if opt.MainDownloadFormat.URL == "" {
		return false, errors.New("no formats available for direct download")
	}
	ext := common.URLDotExt(opt.MainDownloadFormat.URL)
	opt.SetExt(ext)
	if opt.DownloadFileStem != nil && *opt.DownloadFileStem == "" {
		opt.SetStem(time.Now().Format("20060102"))
	}
	ok = true
	//只下载一个
	if opt.AudioDownloadFormat.URL == "" {
		err = downloadFile(ctx, opt.MainDownloadFormat.URL, opt.FilePath(), 0, sink)
		return ok, err
	}

	//需要音视频合并
	avTotal := urlSize(ctx, opt.AudioDownloadFormat.URL)
	avTotal += urlSize(ctx, opt.MainDownloadFormat.URL)
	if common.IsCtxDone(ctx) {
		return ok, ctx.Err()
	}
	vPath := opt.FilePath() + ".video"
	aPath := opt.FilePath() + ".audio"
	err = downloadFile(ctx, opt.MainDownloadFormat.URL, vPath, avTotal, sink)
	if err != nil {
		return ok, err
	}
	err = downloadFile(ctx, opt.AudioDownloadFormat.URL, aPath, avTotal, sink)
	if err != nil {
		return ok, err
	}
	return ok, common.MergeAV(ctx, vPath, aPath, opt.FilePath())
}

func (d *DirectDownloader) SupportedIE() []string {
	return []string{
		instagram.Name(),
	}
}

func (d *DirectDownloader) Name() string {
	return Name()
}

func (m *DirectDownloader) IsNeedFormat() bool {
	return true
}
