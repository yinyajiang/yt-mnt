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

func (i *InstagramIE) IsMatched(link string) bool {
	return strings.Contains(link, "instagram.com")
}

func (i *InstagramIE) Parse(link string, _ ...ies.ParseOptions) (*ies.MediaEntry, error) {
	kind, usr, err := parseInstagramURL(link)
	if err != nil {
		return nil, err
	}
	var entry *ies.MediaEntry
	switch kind {
	case kindUser:
		entry, err = i.client.User(usr)
	case kindStory:
		err = errors.New("instagram story is not supported")
	}
	if err != nil {
		return nil, err
	}
	entry.MediaType = ies.MediaTypeUser
	return entry, nil
}

func (i *InstagramIE) ExtractPage(linkInfo ies.LinkInfo, nextPage *ies.NextPage) ([]*ies.MediaEntry, error) {
	if linkInfo.MediaType != ies.MediaTypeUser {
		return nil, errors.New("only user media type is supported")
	}
	return i.client.UserPostWithPageID(linkInfo.MediaID, nextPage)
}

func (i *InstagramIE) ExtractAllAfterTime(paretnMediaID string, afterTime time.Time) ([]*ies.MediaEntry, error) {
	return ies.HelperGetSubItemsByTime(paretnMediaID,
		func(mediaID string, nextPage *ies.NextPage) ([]*ies.MediaEntry, error) {
			return i.client.UserPostWithPageID(mediaID, nextPage)
		}, afterTime)
}
