package monitor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yinyajiang/yt-mnt/model"
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
	storage, err := storage.NewStorage(dbpath, verbose,
		&model.Feed{},
		&model.Assets{},
	)
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

func (m *Monitor) OpenExplorer(url string, opt ...ies.ParseOptions) (*Explorer, error) {
	ie, err := ies.GetIE(url)
	if err != nil {
		return nil, err
	}
	info, err := ie.Parse(url, opt...)
	if err != nil {
		return nil, err
	}
	var explorer Explorer
	explorer.ie = ie.Name()
	explorer.root = *info
	explorer.pageIndex = 0
	return &explorer, nil
}

func (m *Monitor) UpdateFeed(feedid uint, quality string) (newAssets []*model.Assets, err error) {
	var feed model.Feed
	err = m._db.First(&feed, &model.Feed{
		Model: gorm.Model{
			ID: feedid,
		},
	}).Error
	if err != nil {
		return
	}
	ie, err := ies.GetIE(feed.IE)
	if err != nil {
		return
	}
	newEntries, err := ie.ExtractAllAfterTime(feed.MediaID, feed.LastUpdate)
	if err != nil {
		return
	}
	if len(newEntries) == 0 {
		return
	}
	newAssets, err = m.saveMediaEntryes2Assets(feed.IE, newEntries, feed.ID, 0, quality)
	if err != nil {
		return
	}
	err = m._db.Updates(&model.Feed{
		Model: gorm.Model{
			ID: feedid,
		},
		LastUpdate: time.Now(),
	}).Error
	return
}

func (m *Monitor) Unsubscribe(feedid uint) {
	m._db.Unscoped().Delete(&model.Feed{
		Model: gorm.Model{
			ID: feedid,
		},
	})
	m._db.Unscoped().Delete(&model.Assets{
		OwnerFeedID: feedid,
	})
}

func (m *Monitor) DeleteBundle(bundleid uint) {
	m._db.Unscoped().Delete(&model.Bundles{
		Model: gorm.Model{
			ID: bundleid,
		},
	})
	m._db.Unscoped().Delete(&model.Assets{
		OwnerBundleID: bundleid,
	})
}

func (m *Monitor) Clear() {
	m._db.Unscoped().Where("1 = 1").Delete(&model.Feed{})
	m._db.Unscoped().Where("1 = 1").Delete(&model.Assets{})
	m._db.Unscoped().Where("1 = 1").Delete(&model.Bundles{})
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

func (m *Monitor) GetAsset(id uint) (*model.Assets, error) {
	var entry model.Assets
	err := m._db.First(&entry, &model.Assets{
		Model: gorm.Model{
			ID: id,
		},
	}).Error
	return &entry, err
}

func (m *Monitor) SubscribeSelected(explorer *Explorer) ([]*model.Feed, error) {
	bundles := m.selectedBundlesMedia(explorer, false)
	if len(bundles) == 0 {
		return nil, nil
	}
	return m.saveBundles2Feed(explorer.ie, bundles)
}

func (m *Monitor) AssetbasedSelected(explorer *Explorer, quality string) ([]*model.Bundles, error) {
	bundles := m.selectedBundlesMedia(explorer, true)
	if len(bundles) == 0 {
		return nil, nil
	}
	return m.saveBundles2Assets(explorer.ie, quality, bundles)
}

func (m *Monitor) DownloadAsset(ctx context.Context, id uint, sink downloader.ProgressSink) error {
	asset, err := m.GetAsset(id)
	if err != nil {
		return err
	}
	modelValue := asset.Model

	d := downloader.GetByName(asset.Downloader)

	ok, err := d.Download(ctx, downloader.DownloadOptions{
		URL:            asset.URL,
		Quality:        asset.Quality,
		DownloadDir:    asset.DownloadDir,
		DownloadFile:   asset.DownloadFile,
		DownloadFormat: *asset.DownloadFormat,
		DownloaderData: &asset.DownloaderData,
	}, sink)
	//防止被修改了
	asset.Model = modelValue
	if err != nil {
		if ok {
			asset.Status = model.AssetStatusDownloading
		} else {
			asset.Status = model.AssetStatusFail
		}
	} else {
		asset.Status = model.AssetStatusFinished
	}
	if e := m._db.Save(asset).Error; e != nil {
		log.Printf("db save fail: %s", e)
	}
	return err
}

func (m *Monitor) selectedBundlesMedia(explorer *Explorer, isDeepDownloadable bool) []*ies.MediaEntry {
	root := explorer.Root()
	bundles := []*ies.MediaEntry{}
	for _, item := range explorer.Selected(true) {
		switch item.MediaType {
		case ies.MediaTypeVideo, ies.MediaTypeAudio, ies.MediaTypeImage, ies.MediaTypeCarousel:
			if isDeepDownloadable {
				root.Entries = append(root.Entries, item)
			}
		case ies.MediaTypeUser, ies.MediaTypePlaylist:
			if len(item.Entries) != 0 || !isDeepDownloadable {
				bundles = append(bundles, item)
			} else {
				bundles = append(bundles, m.bundleMediaEntryDeepAnalysis(item.URL)...)
			}
		case ies.MediaTypePlaylistGroup:
			for _, subItem := range item.Entries {
				if isDeepDownloadable {
					bundles = append(bundles, m.bundleMediaEntryDeepAnalysis(subItem.URL)...)
				} else {
					bundles = append(bundles, subItem)
				}
			}
		}
	}
	if len(root.Entries) != 0 {
		bundles = append(bundles, root)
	}
	return bundles
}

/* 每个item表示一个bundle(user/playlist),bundle的子项一定是可以下载的媒体类型 */
func (m *Monitor) bundleMediaEntryDeepAnalysis(url string) []*ies.MediaEntry {
	explorer, err := m.OpenExplorer(url)
	if err != nil {
		return nil
	}
	explorer.ExplorerAll()
	explorer.SetPageIndexToHead()

	root := explorer.Root()
	ret := make([]*ies.MediaEntry, 0)

	for _, item := range explorer.Page() {
		switch item.MediaType {
		case ies.MediaTypeAudio, ies.MediaTypeVideo, ies.MediaTypeImage, ies.MediaTypeCarousel:
			root.Entries = append(root.Entries, item)
		case ies.MediaTypeUser, ies.MediaTypePlaylist:
			if len(item.Entries) != 0 {
				ret = append(ret, item)
			} else {
				ret = append(ret, m.bundleMediaEntryDeepAnalysis(item.URL)...)
			}
		case ies.MediaTypePlaylistGroup:
			for _, subItem := range item.Entries {
				ret = append(ret, m.bundleMediaEntryDeepAnalysis(subItem.URL)...)
			}
		}
	}
	if len(root.Entries) != 0 {
		ret = append(ret, root)
	}
	return ret
}

func (m *Monitor) saveBundles2Feed(ie string, bundles []*ies.MediaEntry) (feeds []*model.Feed, err error) {
	feeds = make([]*model.Feed, 0)
	for _, bundle := range bundles {
		feed := &model.Feed{
			IE:         ie,
			URL:        bundle.URL,
			Title:      bundle.Title,
			Thumbnail:  bundle.Thumbnail,
			MediaID:    bundle.MediaID,
			LastUpdate: time.Now(),
		}
		if bundle.MediaType == ies.MediaTypeUser {
			feed.FeedType = model.FeedUser
		} else if bundle.MediaType == ies.MediaTypePlaylist {
			feed.FeedType = model.FeedPlaylist
		} else {
			err = fmt.Errorf("feed unsupported media type: %d", bundle.MediaType)
			continue
		}
		err = m._db.Create(feed).Error
		if err == nil {
			feeds = append(feeds, feed)
		}
	}
	return feeds, err
}

func (m *Monitor) saveBundles2Assets(ie string, quality string, mediaBundles []*ies.MediaEntry) (bundles []*model.Bundles, err error) {
	bundles = make([]*model.Bundles, 0)
	for _, mediaBundle := range mediaBundles {
		bundle := &model.Bundles{
			URL:       mediaBundle.URL,
			Title:     mediaBundle.Title,
			Thumbnail: mediaBundle.Thumbnail,
		}
		err = m._db.Create(bundle).Error
		if err != nil {
			continue
		}

		subAssets, err := m.saveMediaEntryes2Assets(ie, mediaBundle.Entries, 0, bundle.ID, quality)
		if err == nil {
			bundle.Assets = subAssets
			bundles = append(bundles, bundle)
		}
	}
	if len(bundles) > 0 {
		err = nil
	}
	return bundles, err
}

func (m *Monitor) saveMediaEntryes2Assets(ie string, entryies []*ies.MediaEntry, ownerFeedID, ownerBundleID uint, quality string) (retAssets []*model.Assets, err error) {
	entryies = plain(entryies)

	retAssets = make([]*model.Assets, len(entryies))

	downer := downloader.GetByIE(ie)

	for _, entry := range entryies {
		var qualityFormat *ies.Format
		if downer.IsNeedFormat() {
			if index := selectFormatByResolution(entry.Formats, quality); index >= 0 {
				qualityFormat = entry.Formats[index]
			}
		}
		asset := &model.Assets{
			Status: model.AssetStatusNew,

			Title:          entry.Title,
			URL:            entry.URL,
			Quality:        quality,
			Thumbnail:      entry.Thumbnail,
			DownloadFormat: qualityFormat,

			Downloader: downer.Name(),
		}
		switch entry.MediaType {
		case ies.MediaTypeVideo:
			asset.Type = model.AssetTypeVideo
		case ies.MediaTypeAudio:
			asset.Type = model.AssetTypeAudio
		case ies.MediaTypeImage:
			asset.Type = model.AssetTypeImage
		default:
			err = fmt.Errorf("unsupported media type: %d", asset.Type)
			log.Println(err)
			continue
		}

		if ownerFeedID != 0 {
			asset.OwnerFeedID = ownerFeedID
		} else if ownerBundleID != 0 {
			asset.OwnerBundleID = ownerBundleID
		}
		if e := m._db.Create(asset).Error; e == nil {
			retAssets = append(retAssets, asset)
		} else {
			err = e
		}
	}
	if len(retAssets) > 0 {
		err = nil
	}
	return
}
