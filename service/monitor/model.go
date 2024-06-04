package monitor

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/yinyajiang/yt-mnt/pkg/ies"
	"gorm.io/gorm"
)

const (
	AssetStatusNew = iota + 1
	AssetStatusDownloading
	AssetStatusFinished
	AssetStatusCanceled
	AssetStatusFail
)

const (
	AssetTypeVideo = iota + 1
	AssetTypeAudio
	AssetTypeImage
)

// 每个是一个可下载项
type Asset struct {
	gorm.Model
	BundleID    uint `gorm:"index"`
	BundleTitle string
	Type        int
	Status      int

	Title     string
	Thumbnail string

	URL           string
	Quality       string
	QualityFormat *ies.Format `gorm:"type:json"`
	AudioFormat   *ies.Format `gorm:"type:json"`

	Downloader       string
	DownloaderData   string
	DownloadFileDir  string
	DownloadFileStem string
	DownloadFileExt  string

	DownloadTotalSize int64
	DownloadedSize    int64
	DownloadPercent   float64

	UserData string

	_tabname string
}

func (a *Asset) TableName() string {
	if a._tabname != "" {
		return a._tabname
	}
	return "assets"
}

func (a *Asset) FilePath() string {
	return filepath.Join(a.DownloadFileDir, a.DownloadFileStem+a.DownloadFileExt)
}

func (a *Asset) NotDotExt() string {
	if strings.HasPrefix(a.DownloadFileExt, ".") && len(a.DownloadFileExt) > 1 {
		return a.DownloadFileExt[1:]
	}
	return a.DownloadFileExt
}

const (
	FeedTypeUser = iota + 1
	FeedTypePlaylist
)

const (
	BundleTypeFeed = iota + 1
	BundleTypeGeneric
)

const (
	BundleFlagExternal = 1 << iota
)

type Bundle struct {
	gorm.Model
	IE         string
	BundleType int `gorm:"index"`
	FeedType   int `gorm:"index"`

	URL       string `gorm:"index"`
	MediaID   string
	Title     string
	Thumbnail string
	Uploader  string

	Flags int64

	LastUpdate time.Time
	AssetCount int64    `gorm:"-"`
	Assets     []*Asset `gorm:"foreignKey:BundleID"`

	UserData string

	_tabname string
}

func (f *Bundle) TableName() string {
	if f._tabname != "" {
		return f._tabname
	}
	return "bundles"
}

func (f *Bundle) SetFlags(flag int64, set bool) {
	if set {
		f.Flags = f.Flags | flag
	} else {
		f.Flags = f.Flags &^ flag
	}
}

func (f *Bundle) Flag(flag int64) bool {
	return (f.Flags & flag) != 0
}

type ExternalDownloadingStatManagerFunc struct {
	GetMaxConcurrentCount       func() int
	GetExternalDownloadingCount func() int
	OverMaxConcurrentErr        error
}
