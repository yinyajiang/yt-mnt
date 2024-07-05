package instagram

import (
	"errors"
	"strings"
	"time"

	"github.com/yinyajiang/yt-mnt/pkg/ies"
	"github.com/yinyajiang/yt-mnt/pkg/ies/instagram/insapi"
)

type InstagramIE struct {
	client *insapi.InstagramApi
}

type InstagramReserve struct {
	PostsCount int64
}

func Name() string {
	return "instagram"
}

func init() {
	ies.Regist(&InstagramIE{})
}

func (i *InstagramIE) Name() string {
	return Name()
}

func (i *InstagramIE) Init() error {
	key := ies.Cfg.Tokens[Name()]
	if key == "" {
		return errors.New(Name() + " token is empty")
	}
	i.client = insapi.New(key)
	return nil
}

func (i *InstagramIE) IsMatched(link string) bool {
	return strings.Contains(link, "instagram.com")
}

func (i *InstagramIE) ParseRoot(link string, _ ...ies.ParseOptions) (*ies.MediaEntry, *ies.RootToken, error) {
	kind, usr, err := ParseInstagramURL(link)
	if err != nil {
		return nil, nil, err
	}
	var entry *ies.MediaEntry
	switch kind {
	case KindUser:
		entry, err = i.client.User(usr)
	case KindStory:
		err = errors.New("instagram story is not supported")
	}
	if err != nil {
		return nil, nil, err
	}
	entry.Uploader = usr
	entry.MediaType = ies.MediaTypeUser
	entry.Reserve = InstagramReserve{
		PostsCount: entry.EntryCount,
	}
	return entry, &ies.RootToken{
		MediaID:   entry.MediaID,
		MediaType: entry.MediaType,
	}, nil
}

func (i *InstagramIE) ConvertToUserRoot(_ *ies.RootToken, _ *ies.MediaEntry) error {
	return errors.New("instagram generate user is not supported")
}

func (i *InstagramIE) ExtractPage(rootToken *ies.RootToken, nextPage *ies.NextPageToken) ([]*ies.MediaEntry, error) {
	if rootToken.MediaType != ies.MediaTypeUser {
		return nil, errors.New("only user media type is supported")
	}
	return i.client.UserPostWithPageID(rootToken.MediaID, nextPage)
}

// mustHasItem 调试接口，无论是否时间满足都会返回数据
func (i *InstagramIE) ExtractAllAfterTime(paretnMediaID string, afterTime time.Time, mustHasItem ...bool) ([]*ies.MediaEntry, error) {
	return ies.HelperGetSubItemsByTime(paretnMediaID,
		func(mediaID string, nextPage *ies.NextPageToken) ([]*ies.MediaEntry, error) {
			return i.client.UserPostWithPageID(mediaID, nextPage)
		}, afterTime, mustHasItem...)
}
