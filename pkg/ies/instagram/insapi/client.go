package insapi

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/yinyajiang/yt-mnt/pkg/ies"
)

type InstagramApi struct {
	h   http.Client
	key string
}

func New(key string) *InstagramApi {
	return &InstagramApi{
		h:   http.Client{},
		key: key,
	}
}

func (i *InstagramApi) User(user_name string) (*ies.MediaEntry, error) {
	return i.user(user_name, "")
}

func (i *InstagramApi) UsersStory(user_name string) (*ies.MediaEntry, error) {
	return i.usersStory(user_name, "")
}

func (i *InstagramApi) usersStory(user_name, or_user_id string) (*ies.MediaEntry, error) {
	var js gjson.Result
	var err error
	if user_name != "" {
		js, err = i.get("/v2/user/stories/by/username", map[string]any{
			"username": user_name,
		})
	} else {
		js, err = i.get("/v2/user/stories", map[string]any{
			"user_id": or_user_id,
		})
	}
	if err != nil {
		return nil, err
	}
	reel := js.Get("reel")
	entry := ies.MediaEntry{
		MediaID:     reel.Get("user.pk_id").String(),
		Title:       reel.Get("user.username").String() + " Story",
		URL:         "https://www.instagram.com/stories/" + reel.Get("user.username").String(),
		Description: reel.Get("user.full_name").String(),
		Thumbnail:   reel.Get("user.profile_pic_url").String(),
		EntryCount:  int64(len(reel.Get("items").Array())),
	}
	for _, item := range reel.Get("items").Array() {
		subentry := parseMediaInfo(item)
		entry.Entries = append(entry.Entries, &subentry)
	}
	return &entry, nil
}

func (i *InstagramApi) user(user_name, or_user_id string) (*ies.MediaEntry, error) {
	var js gjson.Result
	var err error
	if user_name != "" {
		js, err = i.get("/v2/user/by/username/", map[string]any{
			"username": user_name,
		})
	} else {
		js, err = i.get("/v2/user/by/id/", map[string]any{
			"id": or_user_id,
		})
	}
	if err != nil {
		return nil, err
	}

	user := ies.MediaEntry{
		MediaID:     js.Get("user.pk_id").String(),
		Title:       js.Get("user.username").String(),
		URL:         "https://www.instagram.com/" + js.Get("user.username").String(),
		Description: js.Get("user.biography").String(),
		Thumbnail:   js.Get("user.profile_pic_url").String(),
		EntryCount:  js.Get("user.media_count").Int(),
		IsPrivate:   js.Get("user.is_private").Bool(),
		Email:       js.Get("user.public_email").String(),
	}
	return &user, nil
}

func (i *InstagramApi) GetUserPostsCount(user_id string) (int64, error) {
	usr, err := i.user("", user_id)
	if err != nil {
		return 0, err
	}
	return usr.EntryCount, nil
}

func (i *InstagramApi) UserPosts(user_id string, latestCount ...int64) ([]*ies.MediaEntry, error) {
	leftCount := int64(0)
	if len(latestCount) > 0 {
		leftCount = latestCount[0]
	}
	if leftCount <= 0 {
		leftCount = math.MaxInt64
	}
	ret := make([]*ies.MediaEntry, 0)
	nextPage := ies.NextPageToken{}
	for {
		if leftCount <= 0 || nextPage.IsEnd {
			break
		}
		medias, err := i.UserPostWithPageID(user_id, &nextPage)
		if err != nil {
			if len(ret) != 0 {
				log.Println(err)
				break
			} else {
				return nil, err
			}
		}
		leftCount -= int64(len(medias))
		ret = append(ret, medias...)
	}
	return ret, nil
}

func (i *InstagramApi) UserPostWithPageID(user_id string, nextPage *ies.NextPageToken) ([]*ies.MediaEntry, error) {
	if nextPage == nil {
		return i.UserPosts(user_id)
	}
	js, err := i.get("/v2/user/medias/", map[string]any{
		"user_id": user_id,
		"page_id": nextPage.NextPageID,
	})
	if err != nil {
		return nil, err
	}
	num_results := js.Get("response.num_results").Int()
	if num_results <= 0 {
		nextPage.IsEnd = true
		return nil, nil
	}

	medias := make([]*ies.MediaEntry, 0)
	for _, item := range js.Get("response.items").Array() {
		media := parseMediaInfo(item)
		medias = append(medias, &media)
	}
	if !js.Get("response.more_available").Bool() {
		nextPage.IsEnd = true
		return medias, nil
	}
	nextPage.NextPageID = js.Get("next_page_id").String()
	if nextPage.NextPageID == "" {
		nextPage.IsEnd = true
	}
	return medias, nil
}

func (i *InstagramApi) get(api string, params map[string]any) (gjson.Result, error) {
	file := ""
	switch api {
	case "/v2/user/stories/by/username/", "/v2/user/stories/by/username":
		file = "/Users/new/Downloads/insta_story.txt"
	}
	if file != "" {
		if f, e := os.Open(file); e == nil {
			if by, e := io.ReadAll(f); e == nil {
				return gjson.ParseBytes(by), e
			}
		}
	}

	u := url.Values{}
	for k, v := range params {
		u.Set(k, fmt.Sprintf("%v", v))
	}
	if !strings.HasPrefix(api, "/") {
		api = "/" + api
	}
	req, err := i.newRequest("GET", "https://api.hikerapi.com"+api+"?"+u.Encode(), nil, nil)
	if err != nil {
		return gjson.Result{}, err
	}
	resp, err := i.h.Do(req)
	if err != nil {
		return gjson.Result{}, err
	}
	defer resp.Body.Close()
	by, err := io.ReadAll(resp.Body)
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(by), nil
}

func (i *InstagramApi) newRequest(method, url string, headers map[string]string, body io.Reader) (req *http.Request, err error) {
	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("x-access-key", i.key)
	return
}

func parseMediaInfo(item gjson.Result) ies.MediaEntry {

	user := item.Get("user.username").String()
	media := ies.MediaEntry{
		MediaID:     item.Get("pk").String(),
		Title:       item.Get("caption.text").String(),
		Description: item.Get("caption.text").String(),
		Thumbnail:   item.Get("thumbnail_url").String(),
		UploadDate:  time.Unix(item.Get("taken_at").Int(), 0),
	}
	if code := item.Get("code").String(); code != "" {
		media.URL = "https://www.instagram.com/p/" + code
	}
	if media.Title == "" {
		media.Title = item.Get("Post by ").String() + user
	}
	if media.Description == "" {
		media.Description = media.Title
	}
	if media.Thumbnail == "" {
		media.Thumbnail = item.Get("image_versions2.candidates.0.url").String()
	}

	switch item.Get("product_type").String() {
	case "story":
		media.URL = "https://www.instagram.com/stories/" + user + "/" + item.Get("pk").String()
	}

	switch item.Get("media_type").Int() {
	case 1: //photo
		media.MediaType = ies.MediaTypeImage
		for _, img := range item.Get("image_versions2.candidates").Array() {
			media.Formats = append(media.Formats, &ies.Format{
				URL:    img.Get("url").String(),
				Width:  img.Get("width").Int(),
				Height: img.Get("height").Int(),
			})
		}
	case 2: //video
		media.MediaType = ies.MediaTypeVideo
		media.Duration = int64(item.Get("video_duration").Float())
		for _, vid := range item.Get("video_versions").Array() {
			media.Formats = append(media.Formats, &ies.Format{
				URL:    vid.Get("url").String(),
				Width:  vid.Get("width").Int(),
				Height: vid.Get("height").Int(),
			})
		}
	case 8: //carousel
		media.MediaType = ies.MediaTypeCarousel
		for _, subitem := range item.Get("carousel_media").Array() {
			subentry := parseMediaInfo(subitem)
			if subentry.URL == "" {
				subentry.URL = media.URL
			}
			if subentry.Title == "" {
				subentry.Title = media.Title
			}
			media.Entries = append(media.Entries, &subentry)
		}
	}
	if media.Title == "" {
		media.Title = media.UploadDate.Format("2006-01-02 15-04-05")
	}
	if media.Title == "" {
		media.Title = media.MediaID
	}
	return media
}
