package model

import (
	"time"
)

type Format struct {
	Width  int64
	Height int64
	URL    string
}

type FormatList []*Format

type MediaEntryList []*MediaEntry

const (
	MediaTypeVideo = iota + 1
	MediaTypeAudio
	MediaTypeImage
	MediaTypeCarousel

	MediaTypeUser
	MediaTypePlaylistGroup
	MediaTypePlaylist
)

/* 可以是单个视频，也可以是playlist */
type MediaEntry struct {
	MediaType int
	LinkID    string
	MediaID   string

	Title       string
	Description string
	Note        string
	Thumbnail   string
	URL         string
	Duration    int64
	UploadDate  time.Time
	IsPrivate   bool

	Formats         FormatList `gorm:"type:json"` // 如果playlist,则无
	QueryEntryCount int64
	Entries         MediaEntryList `gorm:"type:json"`

	IsNew bool `json:"-" gorm:"-"`
}
