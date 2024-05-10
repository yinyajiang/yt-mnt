package youtube

import (
	"errors"
	"log"
	"strings"
	"time"

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

func (y *YoutubeIE) Parse(link string, _ ...ies.ParseOptions) (*ies.MediaEntry, error) {
	linkkind, linkid, err := parseYoutubeURL(link)
	if err != nil {
		return nil, err
	}

	var entry *ies.MediaEntry
	switch linkkind {
	case kindChannel:
		entry, err = y.client.Channel(linkid)
		if err == nil {
			entry.MediaType = ies.MediaTypeUser
		}
	case kindPlaylist:
		entry, err = y.client.Playlist(linkid)
		if err == nil {
			entry.MediaType = ies.MediaTypePlaylist
		}
	case kindPlaylistGroup:
		entry, err = y.client.Channel(linkid)
		if err == nil {
			entry.EntryCount = 0
			entry.MediaType = ies.MediaTypePlaylistGroup
			entry.EntryCount, _ = y.client.ChannelsPlaylistCount(linkid)
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

func (y *YoutubeIE) ExtractPage(linkInfo ies.LinkInfo, nextPage *ies.NextPage) ([]*ies.MediaEntry, error) {
	switch linkInfo.MediaType {
	case ies.MediaTypePlaylistGroup:
		return y.client.ChannelsPlaylistWithPage(linkInfo.LinkID, nextPage)
	case ies.MediaTypePlaylist, ies.MediaTypeUser:
		return y.client.PlaylistsVideoWithPage(linkInfo.MediaID, nextPage)
	}
	return nil, errors.New("unsupported media type")
}

func (y *YoutubeIE) ExtractAllAfterTime(paretnMediaID string, afterTime time.Time) ([]*ies.MediaEntry, error) {
	return ies.HelperGetSubItemsByTime(paretnMediaID,
		func(mediaID string, nextPage *ies.NextPage) ([]*ies.MediaEntry, error) {
			return y.client.PlaylistsVideoWithPage(mediaID, nextPage)
		}, afterTime)
}
