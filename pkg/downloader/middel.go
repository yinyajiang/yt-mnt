package downloader

import (
	"context"
	"fmt"
)

type MiddleDownloader struct {
	d Downloader
}

func (m *MiddleDownloader) Download(ctx context.Context, opt DownloadOptions, sink ProgressSink) (ok bool, err error) {
	if opt.DownloadFileExt == nil || opt.DownloadFileStem == nil {
		return false, fmt.Errorf("DownloadFileExt or DownloadFileStem must be set")
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
