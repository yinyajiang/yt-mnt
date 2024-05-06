package monitor

import (
	"fmt"
	"github.com/yinyajiang/yt-mnt/model"
	"github.com/yinyajiang/yt-mnt/pkg/common"
	"github.com/yinyajiang/yt-mnt/pkg/ies"
	_ "github.com/yinyajiang/yt-mnt/pkg/ies/instagram"
	_ "github.com/yinyajiang/yt-mnt/pkg/ies/youtube"
)

func buildFeed(url string) (*model.Feed, error) {
	ie, err := ies.GetIE(url)
	if err != nil {
		return nil, err
	}
	info, err := ie.Extract(url)
	if err != nil {
		return nil, err
	}
	if info.MediaType != model.MediaTypeUser && info.MediaType != model.MediaTypePlaylist {
		err = fmt.Errorf("feed unsupported media type: %d", info.MediaType)
		return nil, err
	}

	feed := &model.Feed{
		IE:          ie.Name(),
		OriginalURL: url,
		URL:         info.URL,
		Title:       info.Title,
		Thumbnail:   info.Thumbnail,
		QueryID:     info.MediaID,
		Note:        info.Note,
		EntryCount:  info.QueryEntryCount,
	}
	if info.MediaType == model.MediaTypeUser {
		feed.Type = model.FeedTypeUser
	} else if info.MediaType == model.MediaTypePlaylist {
		feed.Type = model.FeedTypePlaylist
	}

	for _, entry := range info.Entries {
		common.SortFormats(entry.Formats)
		feed.Entries = append(feed.Entries, &model.FeedEntry{
			MediaEntry: *entry,
			IE:         ie.Name(),
			Status:     model.FeedEntryStatusNew,
		})
	}
	return feed, nil
}

func updateFeed(feed *model.Feed) error {
	ie, err := ies.GetIE(feed.IE)
	if err != nil {
		return err
	}
	info := &model.MediaEntry{
		MediaID:         feed.QueryID,
		Title:           feed.Title,
		URL:             feed.URL,
		Note:            feed.Note,
		QueryEntryCount: feed.EntryCount,
	}
	for _, entry := range feed.Entries {
		info.Entries = append(info.Entries, &entry.MediaEntry)
	}
	err = ie.Update(info)
	if err != nil {
		return err
	}

	feed.EntryCount = info.QueryEntryCount
	for _, subinfo := range info.Entries {
		if !subinfo.IsNew {
			continue
		}
		common.SortFormats(subinfo.Formats)
		feed.Entries = append(feed.Entries, &model.FeedEntry{
			MediaEntry: *subinfo,
			IE:         ie.Name(),
			Status:     model.FeedEntryStatusNew,
		})
	}
	return nil
}
