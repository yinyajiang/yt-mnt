package youtube

import (
	"errors"
	"strings"
	"time"

	"github.com/yinyajiang/yt-mnt/pkg/ies"
	"github.com/yinyajiang/yt-mnt/pkg/ies/youtube/ytbapi"
)

type YoutubeIE struct {
	client *ytbapi.Client
}

type YoutubeReserve struct {
	VideosCount    int64
	PlaylistsCount int64
}

func Name() string {
	return "youtube"
}

func init() {
	ies.Regist(&YoutubeIE{})
}

func (y *YoutubeIE) Name() string {
	return Name()
}

func (y *YoutubeIE) Init() error {
	key := ies.Cfg.Tokens[Name()]
	if key == "" {
		return errors.New(Name() + " token is empty")
	}
	var err error
	y.client, err = ytbapi.New(key)
	return err
}

func (y *YoutubeIE) IsMatched(link string) bool {
	return strings.Contains(link, "youtube.com")
}

func (y *YoutubeIE) ParseRoot(link string, _ ...ies.ParseOptions) (*ies.MediaEntry, *ies.RootToken, error) {
	linkkind, linkid, err := parseYoutubeURL(link)
	if err != nil {
		return nil, nil, err
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
			reserve := YoutubeReserve{
				VideosCount: entry.EntryCount,
			}
			entry.EntryCount = 0
			entry.MediaType = ies.MediaTypePlaylistGroup
			entry.EntryCount, _ = y.client.ChannelsPlaylistCount(linkid)
			reserve.PlaylistsCount = entry.EntryCount
			entry.Reserve = reserve
		}
	default:
		return nil, nil, errors.New("unsupported url type")
	}
	if err != nil {
		return nil, nil, err
	}
	return entry, &ies.RootToken{
		LinkID:    linkid,
		MediaID:   entry.MediaID,
		MediaType: entry.MediaType,
	}, nil
}

func (y *YoutubeIE) ConvertToUserRoot(rootToken *ies.RootToken, rootInfo *ies.MediaEntry) error {
	if rootInfo == nil {
		return errors.New("invalid root or rootInfo")
	}
	if rootInfo.MediaType == ies.MediaTypeUser {
		return nil
	}
	if rootInfo.MediaType == ies.MediaTypePlaylistGroup {
		if rootInfo.Reserve == nil {
			return errors.New("invalid reserve count")
		}
		rootInfo.MediaType = ies.MediaTypeUser
		reserve, _ := rootInfo.Reserve.(YoutubeReserve)
		rootInfo.EntryCount = reserve.VideosCount
		return nil
	}
	return errors.New("unsupported media type for convert to user root")
}

func (y *YoutubeIE) ExtractPage(root *ies.RootToken, nextPage *ies.NextPageToken) ([]*ies.MediaEntry, error) {
	switch root.MediaType {
	case ies.MediaTypePlaylistGroup:
		return y.client.ChannelsPlaylistWithPage(root.LinkID, nextPage)
	case ies.MediaTypePlaylist, ies.MediaTypeUser:
		return y.client.PlaylistsVideoWithPage(root.MediaID, nextPage)
	}
	return nil, errors.New("unsupported media type")
}

func (y *YoutubeIE) ExtractAllAfterTime(paretnMediaID string, afterTime time.Time) ([]*ies.MediaEntry, error) {
	return ies.HelperGetSubItemsByTime(paretnMediaID,
		func(mediaID string, nextPage *ies.NextPageToken) ([]*ies.MediaEntry, error) {
			return y.client.PlaylistsVideoWithPage(mediaID, nextPage)
		}, afterTime)
}
