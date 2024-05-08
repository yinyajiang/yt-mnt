package monitor

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/yinyajiang/yt-mnt/model"
	"github.com/yinyajiang/yt-mnt/pkg/common"
	"github.com/yinyajiang/yt-mnt/pkg/downloader"
	_ "github.com/yinyajiang/yt-mnt/pkg/downloader/direct"
	"github.com/yinyajiang/yt-mnt/pkg/ies"
	_ "github.com/yinyajiang/yt-mnt/pkg/ies/instagram"
	_ "github.com/yinyajiang/yt-mnt/pkg/ies/youtube"
	"github.com/yinyajiang/yt-mnt/storage"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Monitor struct {
	storage *storage.DBStorage
	_db     *gorm.DB
}

func NewMonitor(dbpath string, verbose bool) *Monitor {
	storage, err := storage.NewStorage(dbpath, verbose)
	if err != nil {
		log.Fatal(err)
	}
	return &Monitor{
		storage: storage,
		_db:     storage.GormDB(),
	}
}

func (m *Monitor) Close() {
	m.storage.Close()
}

func (m *Monitor) Subscribe(url string) (*model.Feed, error) {
	var found model.Feed
	m._db.First(&found, &model.Feed{
		OriginalURL: url,
	})
	if found.ID > 0 {
		return &found, nil
	}

	ie, err := ies.GetIE(url)
	if err != nil {
		return nil, err
	}
	info, err := ie.Parse(url)
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

	err = m._db.Create(feed).Error
	if err != nil {
		return nil, err
	}
	return feed, nil
}

func (m *Monitor) ExploreFirst(url string) (*model.MediaEntry, *ExploreHandle, error) {
	ie, err := ies.GetIE(url)
	if err != nil {
		return nil, nil, err
	}
	info, err := ie.Parse(url)
	if err != nil {
		return nil, nil, err
	}
	var handle ExploreHandle
	handle.linkInfo = ies.ToLinkInfo(info)
	handle.ie = ie.Name()
	handle.first = *info
	subs, err := ie.ExtractPage(handle.linkInfo, &handle.nextPage)
	if err != nil {
		return nil, nil, err
	}
	handle.CurrentCount += int64(len(subs))

	info.Entries = subs
	return info, &handle, nil
}

func (m *Monitor) ExploreNext(handle *ExploreHandle) (*model.MediaEntry, error) {
	if !handle.IsValid() {
		return nil, errors.New("invalid explore handle")
	}
	ie, err := ies.GetIE(handle.ie)
	if err != nil {
		return nil, err
	}
	parent := handle.first
	info := &parent

	subs, err := ie.ExtractPage(handle.linkInfo, &handle.nextPage)
	if err != nil {
		return nil, err
	}
	handle.CurrentCount += int64(len(subs))

	info.Entries = subs
	return info, nil

}

func (m *Monitor) Update(feedid uint) error {
	var feed model.Feed
	err := m._db.First(&feed, &model.Feed{
		Model: gorm.Model{
			ID: feedid,
		},
	}).Error
	if err != nil {
		return err
	}

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
	err = ie.UpdateMedia(info)
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

	for _, entry := range feed.Entries {
		if entry.ID != 0 {
			continue
		}
		entry.OwnerID = feed.ID
		m._db.Create(entry)
	}
	return nil
}

func (m *Monitor) Unsubscribe(feedid uint) {
	var feed model.Feed
	err := m._db.First(&feed, model.Feed{
		Model: gorm.Model{
			ID: feedid,
		},
	}).Error
	if err != nil {
		return
	}
	m._db.Unscoped().Delete(&model.Feed{
		Model: gorm.Model{
			ID: feedid,
		},
	})
	m._db.Unscoped().Delete(&model.FeedEntry{
		OwnerID: feedid,
	})
}

func (m *Monitor) Clear() {
	m._db.Unscoped().Where("1 = 1").Delete(&model.Feed{})
	m._db.Unscoped().Where("1 = 1").Delete(&model.FeedEntry{})
}

func (m *Monitor) SetEntryStatus(id uint, status int) error {
	err := m._db.Model(&model.FeedEntry{
		Model: gorm.Model{
			ID: id,
		},
	}).Updates(&model.FeedEntry{
		Status: status,
	}).Error
	return err
}

func (m *Monitor) GetFeedDetail(feedid uint) (*model.Feed, error) {
	var feed model.Feed
	err := m._db.Preload(clause.Associations).First(&feed, model.Feed{
		Model: gorm.Model{
			ID: feedid,
		},
	}).Error
	return &feed, err
}

func (m *Monitor) ListFeeds(preload bool) ([]*model.Feed, error) {
	var feeds []*model.Feed
	var err error
	if preload {
		err = m._db.Preload(clause.Associations).Find(&feeds).Error
	} else {
		err = m._db.Find(&feeds).Error
	}
	return feeds, err
}

func (m *Monitor) GetFeedEntry(entryid uint) (*model.FeedEntry, error) {
	var entry model.FeedEntry
	err := m._db.First(&entry, &model.FeedEntry{
		Model: gorm.Model{
			ID: entryid,
		},
	}).Error
	return &entry, err
}

func (m *Monitor) DownloadEntry(ctx context.Context, entryid uint, sink downloader.ProgressSink) error {
	entry, err := m.GetFeedEntry(entryid)
	if err != nil {
		return err
	}
	if entry.Downloader == "" {
		entry.Downloader = downloader.GetByIE(entry.IE).Name()
	}
	modelValue := entry.Model

	d := downloader.GetByName(entry.Downloader)
	ok, err := d.Download(ctx, entry, sink)
	//防止被修改了
	entry.Model = modelValue
	if err != nil {
		if ok {
			entry.Status = model.FeedEntryStatusDownloading
		} else {
			entry.Status = model.FeedEntryStatusFail
		}
	} else {
		entry.Status = model.FeedEntryStatusFinished
		entry.DownloaderStaging = ""
	}
	if e := m._db.Save(entry).Error; e != nil {
		log.Printf("db save fail: %s", e)
	}
	return err
}
