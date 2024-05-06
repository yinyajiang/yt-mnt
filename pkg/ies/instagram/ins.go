package instagram

import (
	"github.com/yinyajiang/yt-mnt/model"
	"github.com/yinyajiang/yt-mnt/pkg/ies"
	"github.com/yinyajiang/yt-mnt/pkg/ies/instagram/insapi"
	"log"
	"strings"
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

func (i *InstagramIE) Extract(link string) (*model.MediaEntry, error) {
	kind, usr, err := parseInstagramURL(link)
	if err != nil {
		return nil, err
	}
	var entry *model.MediaEntry
	switch kind {
	case kindUser:
		entry, err = i.client.User(usr)
		if err == nil {
			entry.Entries, err = i.client.UserPosts(entry.MediaID)
		}
	case kindStory:
		entry, err = i.client.UsersStory(usr)
	}
	if err != nil {
		return nil, err
	}
	entry.MediaType = model.MediaTypeUser
	entry.SetNew(true)
	return entry, nil
}

func (i *InstagramIE) Update(entry *model.MediaEntry) error {
	return ies.HelperUpdateSubItems(entry,
		func(userID string) (int64, error) {
			return i.client.GetUserPostsCount(userID)
		},
		func(userID string, latestCount ...int64) ([]*model.MediaEntry, error) {
			return i.client.UserPosts(userID, latestCount...)
		})
}
