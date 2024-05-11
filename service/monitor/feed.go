package monitor

import (
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
	OwnerFeedID   uint
	OwnerBundleID uint
	Type          int
	Status        int

	Title     string
	Thumbnail string

	URL           string
	Quality       string
	QualityFormat *ies.Format `gorm:"type:json"`

	Downloader     string
	DownloaderData string
	DownloadDir    string
	DownloadFile   string

	DownloadTotalSize int64
	DownloadedSize    int64
	DownloadPercent   float64
}

func (a *Asset) TableName() string {
	return "assets"
}

const (
	FeedUser     = "user"
	FeedPlaylist = "playlist"
)

type Feed struct {
	gorm.Model
	IE string

	FeedType string

	URL       string `gorm:"index"`
	MediaID   string
	Title     string
	Thumbnail string

	LastUpdate time.Time
	Assets     []*Asset `gorm:"foreignKey:OwnerFeedID"`
}

func (f *Feed) TableName() string {
	return "feeds"
}

type Bundle struct {
	gorm.Model

	URL        string
	Title      string
	Thumbnail  string
	UploadDate time.Time

	Assets []*Asset `gorm:"foreignKey:OwnerBundleID"`
}

func (b *Bundle) TableName() string {
	return "bundles"
}
