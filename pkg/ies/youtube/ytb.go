package youtube

import (
	"errors"
	"github.com/yinyajiang/yt-mnt/model"
	"github.com/yinyajiang/yt-mnt/pkg/ies"
	"github.com/yinyajiang/yt-mnt/pkg/ies/youtube/ytbapi"
	"log"
	"strings"
)

type YoutubeIE struct {
	client *ytbapi.Client
}

func Name() string {
	return "youtube"
}

func init() {
	key := ies.Cfg.Tokens[Name()]
	if key == "" {
		log.Fatal(Name() + " token is empty")
	}
	client, err := ytbapi.New(key)
	if err != nil {
		log.Println(err)
		return
	}
	ies.Regist(&YoutubeIE{
		client: client,
	})
}

func (y *YoutubeIE) Name() string {
	return Name()
}

func (y *YoutubeIE) IsMatched(link string) bool {
	return strings.Contains(link, "youtube.com")
}

func (y *YoutubeIE) Extract(link string) (*model.MediaEntry, error) {
	linkkind, linkid, err := parseYoutubeURL(link)
	if err != nil {
		return nil, err
	}

	var entry *model.MediaEntry
	switch linkkind {
	case kindChannel:
		entry, err = y.client.Channel(linkid)
		entry.MediaType = model.MediaTypeUser
	case kindPlaylist:
		entry, err = y.client.Playlist(linkid)
		entry.MediaType = model.MediaTypePlaylist
	default:
		return nil, errors.New("unsupported url type")
	}
	if err != nil {
		return nil, err
	}
	if len(entry.Entries) == 0 {
		videos, err := y.client.PlaylistsVideo(entry.MediaID)
		if err != nil {
			return nil, err
		}
		for _, video := range videos {
			video.SetNew(true)
		}
		entry.Entries = videos
	}
	if entry.QueryEntryCount == 0 && len(entry.Entries) > 0 {
		entry.QueryEntryCount = int64(len(entry.Entries))
	}
	entry.SetNew(true)
	return entry, nil
}

func (y *YoutubeIE) Update(entry *model.MediaEntry) error {
	return ies.HelperUpdateSubItems(entry,
		func(playlistQueryID string) (int64, error) {
			return y.client.PlaylistsVideoCount(playlistQueryID)
		},
		func(playlistQueryID string, latestCount ...int64) ([]*model.MediaEntry, error) {
			return y.client.PlaylistsVideo(playlistQueryID, latestCount...)
		})
}
