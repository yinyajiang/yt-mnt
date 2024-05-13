package monitor

import (
	"context"
	"fmt"
	"log"
	"time"

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
	storage     *storage.DBStorage
	_db         *gorm.DB
	preferences Preferences
}

func NewMonitor(dbpath string, verbose bool) *Monitor {
	storage, err := storage.NewStorage(dbpath, verbose,
		&Feed{},
		&Asset{},
		&Bundle{},
		&Preferences{},
	)
	if err != nil {
		log.Fatal(err)
	}

	m := &Monitor{
		storage: storage,
		_db:     storage.GormDB(),
	}
	if e := m._db.First(&m.preferences, Preferences{
		Name: "default",
	}).Error; e != nil {
		m.preferences = Preferences{
			Name:                "default",
			DefaultAssetQuality: "best",
		}
	}
	return m
}

func (m *Monitor) Close() {
	m.storage.Close()
}

func (m *Monitor) SetDefaultQuality(quality string) {
	m.preferences.DefaultAssetQuality = quality
	m._db.Save(&m.preferences)
}

func (m *Monitor) OpenExplorer(url string, opt ...ies.ParseOptions) (*Explorer, error) {
	ie, err := ies.GetIE(url)
	if err != nil {
		return nil, err
	}
	info, rootToken, err := ie.ParseRoot(url, opt...)
	if err != nil {
		return nil, err
	}
	var explorer Explorer
	explorer.ie = ie
	explorer.rootInfo = *info
	explorer.rootToken = *rootToken
	explorer.pageIndex = 0
	return &explorer, nil
}

func (m *Monitor) UpdateFeed(feedid uint, quality ...string) (newAssets []*Asset, err error) {
	var feed Feed
	err = m._db.First(&feed, &Feed{
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
	newAssets, err = m.saveMedia2Assets(feed.IE, newEntries, feed.ID, 0, quality...)
	if err != nil {
		return
	}
	err = m._db.Updates(&Feed{
		Model: gorm.Model{
			ID: feedid,
		},
		LastUpdate: time.Now(),
	}).Error
	return
}

func (m *Monitor) Unsubscribe(feedid uint) {
	m._db.Unscoped().Delete(&Feed{
		Model: gorm.Model{
			ID: feedid,
		},
	})
	m._db.Unscoped().Delete(&Asset{
		OwnerFeedID: feedid,
	})
}

func (m *Monitor) DeleteBundle(bundleid uint) {
	m._db.Unscoped().Delete(&Bundle{
		Model: gorm.Model{
			ID: bundleid,
		},
	})
	m._db.Unscoped().Delete(&Asset{
		OwnerBundleID: bundleid,
	})
}

func (m *Monitor) Clear() {
	m._db.Unscoped().Where("1 = 1").Delete(&Feed{})
	m._db.Unscoped().Where("1 = 1").Delete(&Asset{})
	m._db.Unscoped().Where("1 = 1").Delete(&Bundle{})
}

func (m *Monitor) GetFeedDetail(feedid uint) (*Feed, error) {
	var feed Feed
	err := m._db.Preload(clause.Associations).First(&feed, Feed{
		Model: gorm.Model{
			ID: feedid,
		},
	}).Error
	return &feed, err
}

func (m *Monitor) ListFeeds(preload bool) ([]*Feed, error) {
	var feeds []*Feed
	var err error
	if preload {
		err = m._db.Preload(clause.Associations).Find(&feeds).Error
	} else {
		err = m._db.Find(&feeds).Error
	}
	return feeds, err
}

func (m *Monitor) ListBundles(preload bool) ([]*Bundle, error) {
	var bundles []*Bundle
	var err error
	if preload {
		err = m._db.Preload(clause.Associations).Find(&bundles).Error
	} else {
		err = m._db.Find(&bundles).Error
	}
	return bundles, err
}

func (m *Monitor) GetAsset(id uint) (*Asset, error) {
	var entry Asset
	err := m._db.First(&entry, &Asset{
		Model: gorm.Model{
			ID: id,
		},
	}).Error
	return &entry, err
}

func (m *Monitor) SubscribeSelected(explorer *Explorer) ([]*Feed, error) {
	switch explorer.RootMediaType() {
	case ies.MediaTypePlaylist:
		if explorer.firstSelectedIndex() != IndexRoot {
			return nil, fmt.Errorf("unsupported subscribe selected item")
		}
	case ies.MediaTypeUser:
		if explorer.firstSelectedIndex() != IndexRoot && explorer.firstSelectedIndex() != IndexUser {
			return nil, fmt.Errorf("unsupported subscribe selected item")
		}
	case ies.MediaTypePlaylistGroup:
		break
	default:
		return nil, fmt.Errorf("explored media type unsupported subscribe")
	}
	bundles, err := m.selectedBundlesMedia(explorer, false)
	if err != nil {
		return nil, err
	}
	return m.saveBundles2Feed(explorer.ie.Name(), bundles)
}

func (m *Monitor) AssetbasedSelected(explorer *Explorer, quality ...string) ([]*Bundle, error) {
	switch explorer.RootMediaType() {
	case ies.MediaTypeUser:
		if explorer.firstSelectedIndex() == IndexUser {
			explorer.ExploreNextAll()
		}
	}

	bundles, err := m.selectedBundlesMedia(explorer, true)
	if err != nil {
		return nil, err
	}
	return m.saveBundles2Assets(explorer.ie.Name(), bundles, quality...)
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
		DownloadFormat: *asset.QualityFormat,
		DownloaderData: &asset.DownloaderData,
	}, sink)
	//防止被修改了
	asset.Model = modelValue
	if err != nil {
		if ok {
			asset.Status = AssetStatusDownloading
		} else {
			asset.Status = AssetStatusFail
		}
	} else {
		asset.Status = AssetStatusFinished
	}
	if e := m._db.Save(asset).Error; e != nil {
		log.Printf("db save fail: %s", e)
	}
	return err
}

func (m *Monitor) selectedBundlesMedia(explorer *Explorer, isDeepDownloadable bool) ([]*ies.MediaEntry, error) {
	selected, err := explorer.Selected()
	if err != nil {
		return nil, err
	}
	root := explorer.Root()
	bundles := []*ies.MediaEntry{}
	for _, item := range selected {
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
	if len(bundles) == 0 {
		err = fmt.Errorf("no bundle selected")
	}
	return bundles, err
}

/* 每个item表示一个bundle(user/playlist),bundle的子项一定是可以下载的媒体类型 */
func (m *Monitor) bundleMediaEntryDeepAnalysis(url string) []*ies.MediaEntry {
	explorer, err := m.OpenExplorer(url)
	if err != nil {
		return nil
	}
	explorer.ExploreNextAll()

	root := explorer.Root()
	ret := make([]*ies.MediaEntry, 0)

	for _, item := range explorer.AllPage() {
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

func (m *Monitor) saveBundles2Feed(ie string, bundles []*ies.MediaEntry) (feeds []*Feed, err error) {
	feeds = make([]*Feed, 0)
	for _, bundle := range bundles {
		feed := &Feed{
			IE:         ie,
			URL:        bundle.URL,
			Title:      bundle.Title,
			Thumbnail:  bundle.Thumbnail,
			MediaID:    bundle.MediaID,
			LastUpdate: time.Now(),
		}
		if bundle.MediaType == ies.MediaTypeUser {
			feed.FeedType = FeedUser
		} else if bundle.MediaType == ies.MediaTypePlaylist {
			feed.FeedType = FeedPlaylist
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

func (m *Monitor) saveBundles2Assets(ie string, mediaBundles []*ies.MediaEntry, quality ...string) (bundles []*Bundle, err error) {
	bundles = make([]*Bundle, 0)
	for _, mediaBundle := range mediaBundles {
		bundle := &Bundle{
			URL:        mediaBundle.URL,
			Title:      mediaBundle.Title,
			Thumbnail:  mediaBundle.Thumbnail,
			UploadDate: mediaBundle.UploadDate,
		}
		err = m._db.Create(bundle).Error
		if err != nil {
			continue
		}

		assets, err := m.saveMedia2Assets(ie, mediaBundle.Entries, 0, bundle.ID, quality...)
		if err == nil {
			bundle.Assets = assets
			bundles = append(bundles, bundle)
		}
	}
	if len(bundles) > 0 {
		err = nil
	}
	return bundles, err
}

func (m *Monitor) saveMedia2Assets(ie string, entryies []*ies.MediaEntry, ownerFeedID, ownerBundleID uint, quality_ ...string) (retAssets []*Asset, err error) {
	entryies = plain(entryies)
	retAssets = make([]*Asset, 0, len(entryies))
	downer := downloader.GetByIE(ie)

	quality := m.preferences.DefaultAssetQuality
	if len(quality_) != 0 {
		quality = quality_[0]
	}
	for _, entry := range entryies {
		var qualityFormat *ies.Format
		if downer.IsNeedFormat() {
			if len(entry.Formats) == 0 {
				err = fmt.Errorf("no format found for entry: %s", entry.URL)
				continue
			}
			qualityFormat = entry.Formats[selectFormatByResolution(entry.Formats, quality)]
		}
		asset := &Asset{
			Status: AssetStatusNew,

			Title:         entry.Title,
			URL:           entry.URL,
			Quality:       quality,
			Thumbnail:     entry.Thumbnail,
			QualityFormat: qualityFormat,

			Downloader: downer.Name(),
		}
		switch entry.MediaType {
		case ies.MediaTypeVideo:
			asset.Type = AssetTypeVideo
		case ies.MediaTypeAudio:
			asset.Type = AssetTypeAudio
		case ies.MediaTypeImage:
			asset.Type = AssetTypeImage
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
