package model

import (
	"gorm.io/gorm"
)

const (
	FeedEntryStatusNew = iota + 1
	FeedEntryStatusDownloading
	FeedEntryStatusFinished
	FeedEntryStatusFail
)

// 每个Entry是一个可下载项
type FeedEntry struct {
	gorm.Model
	OwnerID uint
	IE      string
	Status  int

	DownloadQuality   string
	DownloaderStaging string
	DownloadDir       string
	DownloadFile      string
	Downloader        string
	MediaEntry
}

func (f *FeedEntry) TableName() string {
	return "feed_entries"
}

const (
	FeedTypeUser     = "user"
	FeedTypePlaylist = "playlist"
)

type Feed struct {
	gorm.Model
	IE          string
	OriginalURL string `gorm:"index"`
	Type        string

	URL       string
	QueryID   string
	Note      string
	Title     string
	Thumbnail string

	EntryCount int64
	Entries    []*FeedEntry `gorm:"foreignKey:OwnerID"`
}

func (f *Feed) TableName() string {
	return "feeds"
}
