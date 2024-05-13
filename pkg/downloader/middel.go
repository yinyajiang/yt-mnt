package downloader

import (
	"context"
	"fmt"
)

type MiddleDownloader struct {
	d Downloader
}

func (m *MiddleDownloader) Download(ctx context.Context, opt DownloadOptions, sink ProgressSink, stageSaver_ ...DownloaderStageSaver) (ok bool, err error) {
	var stageSaver DownloaderStageSaver
	if len(stageSaver_) > 0 {
		stageSaver = stageSaver_[0]
	} else {
		stageSaver = &DefaultDownloaderStageSaver{}
	}
	if opt.DownloadFileExt == nil || opt.DownloadFileStem == nil {
		return false, fmt.Errorf("DownloadFileExt or DownloadFileStem must be set")
	}
	return m.d.Download(ctx, opt, sink, stageSaver)
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
