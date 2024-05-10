package ies

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type Format struct {
	Width  int64
	Height int64
	URL    string
}

const (
	MediaTypeVideo = iota + 1
	MediaTypeAudio
	MediaTypeImage
	MediaTypeCarousel

	MediaTypeUser
	MediaTypePlaylist
	MediaTypePlaylistGroup
)

type MediaEntry struct {
	MediaType int
	LinkID    string
	MediaID   string

	Title       string
	Description string
	Thumbnail   string
	URL         string
	Duration    int64
	UploadDate  time.Time
	IsPrivate   bool

	Formats    []*Format
	EntryCount int64
	Entries    []*MediaEntry
}

func (m *MediaEntry) LinkInfo() LinkInfo {
	return LinkInfo{
		LinkID:    m.LinkID,
		MediaID:   m.MediaID,
		MediaType: m.MediaType,
	}
}

func (f *Format) Value() (driver.Value, error) {
	if f == nil {
		return nil, nil
	}
	return json.Marshal(f)
}

func (f *Format) Scan(value interface{}) error {
	if f == nil {
		return nil
	}
	data, ok := value.([]byte)
	if !ok || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, f)
}

type FormatList []*Format

func (f FormatList) Value() (driver.Value, error) {
	if len(f) == 0 {
		return nil, nil
	}
	return json.Marshal(f)
}

func (f *FormatList) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	data, ok := value.([]byte)
	if !ok || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, f)
}
