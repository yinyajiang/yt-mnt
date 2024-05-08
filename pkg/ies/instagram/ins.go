package instagram

import (
	"errors"
	"log"
	"strings"

	"github.com/yinyajiang/yt-mnt/model"
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

func (i *InstagramIE) Parse(link string) (*model.MediaEntry, error) {
	kind, usr, err := parseInstagramURL(link)
	if err != nil {
		return nil, err
	}
	var entry *model.MediaEntry
	switch kind {
	case kindUser:
		entry, err = i.client.User(usr)
	case kindStory:
		err = errors.New("instagram story is not supported")
	}
	if err != nil {
		return nil, err
	}
	entry.MediaType = model.MediaTypeUser
	entry.SetNew(true)
	return entry, nil
}

func (i *InstagramIE) ExtractPage(linkInfo ies.LinkInfo, nextPage *ies.NextPage) ([]*model.MediaEntry, error) {
	if linkInfo.MediaType != model.MediaTypeUser {
		return nil, errors.New("only user media type is supported")
	}
	return i.client.UserPostWithPageID(linkInfo.MediaID, nextPage)
}

func (i *InstagramIE) UpdateMedia(entry *model.MediaEntry) error {
	return ies.HelperUpdateSubItems(entry,
		func(userID string) (int64, error) {
			return i.client.GetUserPostsCount(userID)
		},
		func(userID string, latestCount ...int64) ([]*model.MediaEntry, error) {
			return i.client.UserPosts(userID, latestCount...)
		})
}
