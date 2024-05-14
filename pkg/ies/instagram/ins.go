package instagram

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/yinyajiang/yt-mnt/pkg/ies"
	"github.com/yinyajiang/yt-mnt/pkg/ies/instagram/insapi"
)

type InstagramIE struct {
	client *insapi.InstagramApi
}

func Name() string {
	return "instagram"
}

func init() {
	key := ies.Cfg.Tokens[Name()]
	if key == "" {
		log.Fatal(Name() + " token is empty")
	}
	client := insapi.New(key)
	ies.Regist(&InstagramIE{
		client: client,
	})
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
	kind, usr, err := parseInstagramURL(link)
	if err != nil {
		return nil, nil, err
	}
	var entry *ies.MediaEntry
	switch kind {
	case kindUser:
		entry, err = i.client.User(usr)
	case kindStory:
		err = errors.New("instagram story is not supported")
	}
	if err != nil {
		return nil, nil, err
	}
	entry.MediaType = ies.MediaTypeUser
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

func (i *InstagramIE) ExtractAllAfterTime(paretnMediaID string, afterTime time.Time) ([]*ies.MediaEntry, error) {
	return ies.HelperGetSubItemsByTime(paretnMediaID,
		func(mediaID string, nextPage *ies.NextPageToken) ([]*ies.MediaEntry, error) {
			return i.client.UserPostWithPageID(mediaID, nextPage)
		}, afterTime)
}
