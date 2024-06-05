package ies

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

const (
	FormatTypeComplete = iota
	FormatTypeVideo
	FormatTypeAudio
)

type Format struct {
	Width      int64
	Height     int64
	URL        string
	FormatType int
}

const (
	MediaTypeVideo = iota + 1
	MediaTypeAudio
	MediaTypeImage
	MediaTypeCarousel

	/**************************/
	MediaTypePlaylist
	MediaTypePlaylistGroup
	MediaTypeUser
)

type MediaEntry struct {
	MediaType int
	MediaID   string
	IsPrivate bool

	Title       string
	Description string
	Thumbnail   string
	URL         string
	Duration    int64
	UploadDate  time.Time
	Uploader    string
	Email       string
	Formats     []*Format
	EntryCount  int64
	Entries     []*MediaEntry

	Reserve any
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
