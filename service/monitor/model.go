package monitor

import (
	"encoding/json"
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

	URL                 string
	Quality             string
	Subtitle            string
	IsDownloadThumbnail bool
	IsOriginalSubtitle  bool
	HopeMediaType       string

	Duration      int64
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
	BundleFlagUnparse
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

	LastUpdate         time.Time
	AssetCount         int64    `gorm:"-"`
	AssetFinishedCount int64    `gorm:"-"`
	Assets             []*Asset `gorm:"foreignKey:BundleID"`

	UserData   string
	UserKVData string

	_tabname string
	_kvdata  map[string]any
}

func (f *Bundle) TableName() string {
	if f._tabname != "" {
		return f._tabname
	}
	return "bundles"
}

func (f *Bundle) SetFlag(flag int64) {
	f.setFlags(flag, true)
}

func (f *Bundle) UnSetFlag(flag int64) {
	f.setFlags(flag, false)
}

func (f *Bundle) SetKVData(kv map[string]any) {
	f._kvdata = kv
	if len(f._kvdata) != 0 {
		by, _ := json.Marshal(f._kvdata)
		if len(by) != 0 {
			f.UserKVData = string(by)
		}
	} else {
		f.UserKVData = ""
	}
}

func (f *Bundle) AddKVData(key string, value any) {
	if f._kvdata == nil {
		f.GetKVData("")
	}
	if key == "" {
		return
	}
	f._kvdata[key] = value
	f.SetKVData(f._kvdata)
}

func (f *Bundle) GetKVData(key string) (any, bool) {
	if f._kvdata == nil {
		if f.UserKVData != "" {
			json.Unmarshal([]byte(f.UserKVData), &f._kvdata)
		}
		if f._kvdata == nil {
			f._kvdata = make(map[string]any, 0)
		}
	}
	if key == "" {
		return nil, false
	}
	v, ok := f._kvdata[key]
	return v, ok
}

func (f *Bundle) GetKVDataInt(key string, def int) int {
	v, ok := f.GetKVData(key)
	if !ok || v == nil {
		return def
	}
	switch v := v.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case int64:
		return int(v)
	}
	return def
}

func (f *Bundle) GetKVDataString(key string) string {
	v, ok := f.GetKVData(key)
	if !ok || v == nil {
		return ""
	}
	switch v := v.(type) {
	case string:
		return v
	}
	return ""
}

func (f *Bundle) setFlags(flag int64, set bool) {
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

type LastDownloading struct {
	gorm.Model
	AssetID  uint
	_tabname string
}

func (l *LastDownloading) TableName() string {
	if l._tabname != "" {
		return l._tabname
	}
	return "last_downloading"
}
