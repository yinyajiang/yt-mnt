package monitor

import (
	"context"
	"github.com/yinyajiang/yt-mnt/model"
	"github.com/yinyajiang/yt-mnt/pkg/downloader"
	_ "github.com/yinyajiang/yt-mnt/pkg/downloader/direct"
	"github.com/yinyajiang/yt-mnt/storage"
	"log"

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
	feed, err := buildFeed(url)
	if err != nil {
		return nil, err
	}
	err = m._db.Create(feed).Error
	if err != nil {
		return nil, err
	}
	return feed, nil
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
	err = updateFeed(&feed)
	if err != nil {
		return err
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
