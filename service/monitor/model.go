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

	Downloader       string
	DownloaderData   string
	DownloadFileDir  string
	DownloadFileStem string
	DownloadFileExt  string

	DownloadTotalSize int64
	DownloadedSize    int64
	DownloadPercent   float64

	__tabname string
}

func (a *Asset) TableName() string {
	if a.__tabname != "" {
		return a.__tabname
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

type Bundle struct {
	gorm.Model
	IE         string
	BundleType int

	FeedType int `gorm:"index"`

	URL       string `gorm:"index"`
	MediaID   string
	Title     string
	Thumbnail string
	Uploader  string

	LastUpdate time.Time
	AssetCount int64    `gorm:"-"`
	Assets     []*Asset `gorm:"foreignKey:BundleID"`

	__tabname string
}

func (f *Bundle) TableName() string {
	if f.__tabname != "" {
		return f.__tabname
	}
	return "bundles"
}

type Preferences struct {
	gorm.Model
	Name                string `gorm:"unique"`
	DefaultAssetQuality string
}
