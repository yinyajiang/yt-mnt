package downloader

import (
	"context"
	"errors"
	"fmt"
)

type MiddleDownloader struct {
	d Downloader
}

func (m *MiddleDownloader) Delete(downloaderData DeleteOptions, deleteFile bool) {
	m.d.Delete(downloaderData, deleteFile)
}

func (m *MiddleDownloader) ChangeFileTitle(opt DownloadOptions, title string) error {
	if opt.DownloadFileStem == nil {
		return errors.New("downloadFileStem is nil")
	}
	if title == "" {
		return errors.New("title is empty")
	}
	return m.d.ChangeFileTitle(opt, title)
}
func (m *MiddleDownloader) Download(ctx context.Context, opt DownloadOptions, sink_ ProgressSink) (ok bool, err error) {
	if opt.DownloadFileExt == nil || opt.DownloadFileStem == nil {
		return false, fmt.Errorf("DownloadFileExt or DownloadFileStem must be set")
	}
	sink := func(total, downloaded, speed, eta int64, percent float64, videoDuration int64) {
		if downloaded < 0 {
			downloaded = opt.DownloadedSize
		}
		if percent < 0 {
			percent = opt.DownloadPercent
		}
		if percent == 0 && total != 0 {
			percent = float64(downloaded) / float64(total) * 100
		}
		if videoDuration <= 0 && opt.Duration != nil {
			videoDuration = *opt.Duration
		}
		if sink_ != nil {
			sink_(total, downloaded, speed, eta, percent, videoDuration)
		}
	}
	return m.d.Download(ctx, opt, sink)
}

func (m *MiddleDownloader) Name() string {
	return m.d.Name()
}

func (m *MiddleDownloader) IsNeedFormat() bool {
	return m.d.IsNeedFormat()
}

func (m *MiddleDownloader) SupportedIE() []string {
	return m.d.SupportedIE()
}
