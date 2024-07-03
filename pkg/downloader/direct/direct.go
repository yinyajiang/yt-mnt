package direct

import (
	"context"
	"errors"
	"os"
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

func (m *DirectDownloader) Delete(opt downloader.DeleteOptions, deleteFile bool) {
	if !deleteFile {
		return
	}
	if !opt.HasAudioFormat {
		if common.IsExistsFile(opt.FilePath()) {
			os.Remove(opt.FilePath())
		}
		return
	}
	vPath := opt.FilePath() + ".video"
	if common.IsExistsFile(vPath) {
		os.Remove(opt.FilePath())
	}
	aPath := opt.FilePath() + ".audio"
	if common.IsExistsFile(aPath) {
		os.Remove(opt.FilePath())
	}
}

func (m *DirectDownloader) ChangeFileTitle(opt downloader.DownloadOptions, title string) error {
	if opt.DownloadFileStem == nil {
		return errors.New("downloadFileStem is nil")
	}
	if title == "" {
		return errors.New("title is empty")
	}
	vPath := opt.FilePath() + ".video"
	aPath := opt.FilePath() + ".audio"
	if _, err := os.Stat(vPath); err == nil {
		os.Remove(vPath)
	}
	if _, err := os.Stat(aPath); err == nil {
		os.Remove(aPath)
	}
	opt.SetStem(title)
	return nil
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
