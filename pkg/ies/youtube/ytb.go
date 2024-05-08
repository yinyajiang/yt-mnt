package youtube

import (
	"errors"
	"log"
	"strings"

	"github.com/yinyajiang/yt-mnt/model"
	"github.com/yinyajiang/yt-mnt/pkg/ies"
	"github.com/yinyajiang/yt-mnt/pkg/ies/youtube/ytbapi"
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

func (y *YoutubeIE) Parse(link string) (*model.MediaEntry, error) {
	linkkind, linkid, err := parseYoutubeURL(link)
	if err != nil {
		return nil, err
	}

	var entry *model.MediaEntry
	switch linkkind {
	case kindChannel:
		entry, err = y.client.Channel(linkid)
		if err == nil {
			entry.MediaType = model.MediaTypeUser
		}
	case kindPlaylist:
		entry, err = y.client.Playlist(linkid)
		if err == nil {
			entry.MediaType = model.MediaTypePlaylist
		}
		if err == nil {
			entry.MediaType = model.MediaTypePlaylistGroup
		}
	case kindPlaylistGroup:
		entry, err = y.client.Channel(linkid)
		if err == nil {
			entry.MediaType = model.MediaTypePlaylistGroup
		}
	default:
		return nil, errors.New("unsupported url type")
	}
	if err != nil {
		return nil, err
	}
	entry.LinkID = linkid
	return entry, nil
}

func (y *YoutubeIE) ExtractPage(linkInfo ies.LinkInfo, nextPage *ies.NextPage) ([]*model.MediaEntry, error) {
	switch linkInfo.MediaType {
	case model.MediaTypePlaylistGroup:
		return y.client.ChannelsPlaylistWithPage(linkInfo.LinkID, nextPage)
	case model.MediaTypePlaylist, model.MediaTypeUser:
		return y.client.PlaylistsVideoWithPage(linkInfo.MediaID, nextPage)
	}
	return nil, errors.New("unsupported media type")
}

func (y *YoutubeIE) UpdateMedia(entry *model.MediaEntry) error {
	if entry.MediaType == model.MediaTypePlaylistGroup {
		return errors.New("playlist group is not supported for update")
	}
	return ies.HelperUpdateSubItems(entry,
		func(mediaID string) (int64, error) {
			return y.client.PlaylistsVideoCount(mediaID)
		},
		func(mediaID string, latestCount ...int64) ([]*model.MediaEntry, error) {
			return y.client.PlaylistsVideo(mediaID, latestCount...)
		})
}
