package monitor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/yinyajiang/yt-mnt/pkg/common"
	"github.com/yinyajiang/yt-mnt/pkg/db"
	"github.com/yinyajiang/yt-mnt/pkg/downloader"
	_ "github.com/yinyajiang/yt-mnt/pkg/downloader/direct"
	"github.com/yinyajiang/yt-mnt/pkg/ies"
	"github.com/yinyajiang/yt-mnt/pkg/ies/instagram"
	_ "github.com/yinyajiang/yt-mnt/pkg/ies/instagram"
	"github.com/yinyajiang/yt-mnt/pkg/ies/youtube"
	_ "github.com/yinyajiang/yt-mnt/pkg/ies/youtube"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type downloadingStat struct {
	id       uint
	bundleID uint
	cancel   context.CancelFunc
}

type Monitor struct {
	storage *db.DBStorage
	_db     *gorm.DB

	lock                               sync.RWMutex
	downloading                        map[uint]*downloadingStat
	externalDownloadingStatManagerFunc ExternalDownloadingStatManagerFunc

	_lastBundle           Bundle
	_lastBundleDirty      bool
	_lastBundlePreload    bool
	_lastBundleAssetCount bool
}

type MonitorOption struct {
	Verbose                            bool
	IEToken                            ies.IETokens
	AssetTableName                     string
	BundleTableName                    string
	LastDownloadingTableName           string
	RegistDownloader                   []downloader.Downloader
	DBOption                           db.DBOption
	ExternalDownloadingStatManagerFunc ExternalDownloadingStatManagerFunc
}

func NewMonitor(opt MonitorOption) (*Monitor, error) {
	err := ies.InitIE(opt.IEToken)
	if err != nil {
		return nil, err
	}

	for _, downer := range opt.RegistDownloader {
		downloader.Regist(downer)
	}

	storage, err := db.NewStorage(opt.DBOption, opt.Verbose,
		&Asset{
			_tabname: opt.AssetTableName,
		},
		&Bundle{
			_tabname: opt.BundleTableName,
		},
		&LastDownloading{
			_tabname: opt.LastDownloadingTableName,
		},
	)
	if err != nil {
		return nil, err
	}

	m := &Monitor{
		storage:                            storage,
		_db:                                storage.GormDB(),
		downloading:                        make(map[uint]*downloadingStat),
		externalDownloadingStatManagerFunc: opt.ExternalDownloadingStatManagerFunc,
	}
	return m, nil
}

func (m *Monitor) SetExternalDownloadingStatManagerFunc(f ExternalDownloadingStatManagerFunc) {
	m.externalDownloadingStatManagerFunc = f
}

func (m *Monitor) Close(recordDownloadings ...bool) {
	if len(recordDownloadings) > 0 && recordDownloadings[0] {
		m.RecordDownloadings()
	}
	m.StopAllDownloading(true)
	m.storage.Close()
}

func (m *Monitor) LocalDownloaderStageSaver(dir string) downloader.DownloaderStageSaver {
	return downloader.NewLocalDirStageSaver(dir)
}

func (m *Monitor) CreateExplorerCacher(cacheCount ...int) *ExplorerCaches {
	return NewExplorerCaches(cacheCount...)
}

func (m *Monitor) OpenExplorer(url string, isPlain bool, opt ...ies.ParseOptions) (*Explorer, error) {
	ie, err := ies.GetIE(url)
	if err != nil {
		return nil, err
	}
	info, rootToken, err := ie.ParseRoot(url, opt...)
	if err != nil {
		return nil, err
	}
	explorer := newExplorer()
	explorer.ie = ie
	explorer.url = url
	explorer.rootInfo = *info
	explorer.rootToken = *rootToken
	if isPlain {
		explorer.SetPlain()
	}
	return explorer, nil
}

func (m *Monitor) OpenItemExplorer(parentExplorer *Explorer, index int) (*Explorer, error) {
	item, err := parentExplorer.Item(index, false)
	if err != nil {
		return nil, err
	}
	return m.OpenExplorer(item.URL, false)
}

func (m *Monitor) RecordDownloadings() error {
	m.storage.DeleteAll(&LastDownloading{})
	ids := m.getDownloadingsID()
	if len(ids) == 0 {
		return nil
	}
	for _, id := range ids {
		m.storage.Create(&LastDownloading{
			AssetID: id,
		})
	}
	return nil
}

func (m *Monitor) GetLastDownloadings() []uint {
	var lasts []LastDownloading
	m._db.Find(&lasts)
	ids := []uint{}
	for _, last := range lasts {
		ids = append(ids, last.AssetID)
	}
	return ids
}

func (m *Monitor) SubscriptionID(url string) (id uint, ok bool) {
	var ids []uint
	m._db.Model(&Bundle{}).Where(&Bundle{
		URL:        url,
		BundleType: BundleTypeFeed,
	}).Pluck("id", &ids)
	if len(ids) > 0 {
		return ids[0], true
	}
	return 0, false
}

func (m *Monitor) UpdateFeed(feedid uint, dir, quality string) (newAssets []*Asset, err error) {
	var feed Bundle
	err = m._db.First(&feed, &Bundle{
		Model: gorm.Model{
			ID: feedid,
		},
		BundleType: BundleTypeFeed,
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
	newAssets, err = m.saveAssets(feed.IE, "", newEntries, &feed, dir, quality)
	if err != nil {
		return
	}

	if m._lastBundle.ID == feedid {
		m._lastBundleDirty = true
	}

	err = m.storage.Updates(&Bundle{
		Model: gorm.Model{
			ID: feedid,
		},
		LastUpdate: time.Now(),
	})
	return
}

func (m *Monitor) Unsubscribe(id uint, deleteEmpty bool) (isDeleted bool, err error) {
	if deleteEmpty {
		assetCount := int64(0)
		if e := m._db.Model(&Asset{}).Where(&Asset{
			BundleID: id,
		}).Count(&assetCount).Error; e != nil {
			assetCount = 1
		}
		isDeleted = assetCount == 0
	}
	if isDeleted {
		m.DeleteBundle(id, false)
		return
	} else {
		err = m.storage.WhereUpdates(&Bundle{
			Model: gorm.Model{
				ID: id,
			},
		}, &Bundle{
			BundleType: BundleTypeGeneric,
		})
		return
	}
}

func (m *Monitor) DeleteAsset(id uint, remainFinished bool) {
	if id <= 0 {
		return
	}
	m.StopDownloading(id, true)
	m._lastBundleDirty = true

	var aset Asset
	m._db.First(&aset, &Asset{
		Model: gorm.Model{
			ID: id,
		},
	})
	m.deleteDownloaderItem(&aset, remainFinished)

	m.storage.Delete(&Asset{
		Model: gorm.Model{
			ID: id,
		},
	})
}

func (m *Monitor) DeleteBundle(id uint, remainFinished bool) {
	if id <= 0 {
		return
	}
	ids := m.allDownloadingBundleAssetID(id)
	for _, id := range ids {
		m.StopDownloading(id, true)
	}

	deleteBundle := true
	assets := []*Asset{}

	if remainFinished {
		m._db.Model(&Asset{}).Not(&Asset{
			Status: AssetStatusFinished,
		}).Where(&Asset{
			BundleID: id,
		}).Find(&assets)

		m._db.Model(&Asset{}).Not(&Asset{
			Status: AssetStatusFinished,
		}).Where(&Asset{
			BundleID: id,
		}).Unscoped().Delete(&Asset{})

		var count int64
		m._db.Model(&Asset{}).Where(&Asset{
			BundleID: id,
		}).Count(&count)
		deleteBundle = count == 0
	} else {
		m._db.Find(&assets, &Asset{
			BundleID: id,
		})
		m.storage.Delete(&Asset{
			BundleID: id,
		})
		deleteBundle = true
	}

	if deleteBundle {
		m.storage.Delete(&Bundle{
			Model: gorm.Model{
				ID: id,
			},
		})
	}

	for _, aset := range assets {
		m.deleteDownloaderItem(aset, remainFinished)
	}
}

func (m *Monitor) ClearTypeBundles(remainFinished bool, bundleTypes ...int) []uint {
	deleted := []uint{}
	bundles, _ := m.ListTypeBundles(false, false, bundleTypes...)
	for _, bundle := range bundles {
		deleted = append(deleted, bundle.ID)
		m.DeleteBundle(bundle.ID, remainFinished)
	}
	return deleted
}

func (m *Monitor) deleteDownloaderItem(asset *Asset, remainFinished bool) {
	if asset == nil {
		return
	}
	deleteFile := !remainFinished
	if !deleteFile && asset.Status != AssetStatusFinished {
		deleteFile = true
	}
	d := downloader.GetByName(asset.Downloader)
	if d == nil {
		return
	}
	d.Delete(downloader.DeleteOptions{
		HasAudioFormat:   asset.AudioFormat != nil,
		DownloadFileDir:  asset.DownloadFileDir,
		DownloadFileStem: asset.DownloadFileStem,
		DownloadFileExt:  asset.DownloadFileExt,
		DownloaderData:   asset.DownloaderData,
	}, deleteFile)
}

func (m *Monitor) ClearAll() {
	m.StopAllDownloading(true)
	bundles, _ := m.ListBundlesByWheres(false, false)
	for _, bundle := range bundles {
		m.DeleteBundle(bundle.ID, false)
	}
	m.storage.DeleteAll(&Bundle{})

	var asets []*Asset
	m._db.Find(&asets)
	for _, aset := range asets {
		m.deleteDownloaderItem(aset, true)
	}
	m.storage.DeleteAll(&Asset{})
	m.storage.DeleteAll(&LastDownloading{})
}

func (m *Monitor) GetBundle(id uint, preload bool, assetCount bool) (*Bundle, error) {
	var bundle Bundle
	var err error

	if !m._lastBundleDirty && m._lastBundle.ID == id && m._lastBundlePreload == preload &&
		(m._lastBundleAssetCount || !assetCount) {
		bundle = m._lastBundle
		return &bundle, nil
	}

	defer func() {
		if bundle.ID > 0 && err == nil {
			m._lastBundleDirty = false
			m._lastBundle = bundle
			m._lastBundlePreload = preload
			m._lastBundleAssetCount = assetCount
		}
	}()

	bundles, err := m.ListBundlesByWheres(preload, assetCount, &Bundle{
		Model: gorm.Model{
			ID: id,
		},
	})
	if err != nil {
		return nil, err
	}
	bundle = *bundles[0]
	return &bundle, err
}

func (m *Monitor) ListTypeBundles(preload bool, assetCount bool, bundleTypes ...int) ([]*Bundle, error) {
	bundle := []*Bundle{}
	for _, t := range bundleTypes {
		bundle = append(bundle, &Bundle{
			BundleType: t,
		})
	}
	return m.ListBundlesByWheres(preload, assetCount, bundle...)
}

func (m *Monitor) ListBundlesByWheres(preload bool, assetCount bool, orwheres ...*Bundle) ([]*Bundle, error) {
	var bundles []*Bundle
	var err error
	var tx *gorm.DB
	if preload {
		tx = m._db.Preload(clause.Associations).Order("id DESC")
	} else {
		tx = m._db.Order("id DESC")
	}
	if len(orwheres) > 1 {
		for _, or := range orwheres {
			tx = tx.Or(or)
		}
		err = tx.Find(&bundles).Error
	} else if len(orwheres) == 1 {
		err = tx.Find(&bundles, orwheres[0]).Error
	} else {
		err = tx.Find(&bundles).Error
	}
	if err == nil && assetCount {
		for _, bundle := range bundles {
			m._db.Model(&Asset{}).Where(&Asset{
				BundleID: bundle.ID,
			}).Count(&bundle.AssetCount)
			m._db.Model(&Asset{}).Where(&Asset{
				Status:   AssetStatusFinished,
				BundleID: bundle.ID,
			}).Count(&bundle.AssetFinishedCount)
		}
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

func (m *Monitor) ListAssets(bundleID uint) (assets []*Asset, err error) {
	err = m._db.Where(&Asset{
		BundleID: bundleID,
	}).Order("id DESC").Find(&assets).Error
	return
}

func (m *Monitor) FirstAsset(bundleID uint) (*Asset, error) {
	var asset Asset
	err := m._db.Where(&Asset{
		BundleID: bundleID,
	}).Order("id DESC").First(&asset).Error
	return &asset, err
}

func (m *Monitor) ListAssetsWithOffset(bundleID uint, offset, limit int) (assets []*Asset, err error) {
	if limit <= 0 && offset <= 0 {
		assets, err = m.ListAssets(bundleID)
		return
	}
	if limit == 0 {
		limit = -1
	}

	err = m._db.Where(&Asset{
		BundleID: bundleID,
	}).Offset(offset).Limit(limit).Order("id DESC").Find(&assets).Error
	return
}

func (m *Monitor) SubscribeSelected(explorer *Explorer) ([]*Bundle, error) {
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
	selectedBundles, err := m.selectedBundlesMedia(explorer, false)
	if err != nil {
		return nil, err
	}

	var saveBundles []*ies.MediaEntry
	var existBundles []*Bundle
	for _, entry := range selectedBundles {
		feedType := mediaType2FeedType(entry.MediaType)
		if feedType == 0 {
			err = fmt.Errorf("feed unsupported media type: %d", entry.MediaType)
			continue
		}
		var exist Bundle
		if m._db.First(&exist, &Bundle{
			URL:        entry.URL,
			BundleType: BundleTypeFeed,
			FeedType:   feedType,
		}).Error == nil {
			existBundles = append(existBundles, &exist)
		} else {
			saveBundles = append(saveBundles, entry)
		}
	}
	if len(saveBundles) == 0 {
		if len(existBundles) != 0 {
			err = fmt.Errorf("feed already exists")
		}
		return existBundles, err
	}

	result, err := m.saveBundles(explorer.ie.Name(), nil, saveBundles, BundleTypeFeed, "", "", false)
	if err != nil {
		return nil, err
	}
	result = append(result, existBundles...)
	return result, nil
}

func (m *Monitor) SubscribeURLAndAddAsset(url string, entries []*ies.MediaEntry, dir, quality string) (*Bundle, error) {
	if _, ok := m.SubscriptionID(url); ok {
		return nil, fmt.Errorf("feed already exists")
	}

	explorer, err := m.OpenExplorer(url, false)
	if err != nil {
		return nil, err
	}
	defer explorer.Close()
	explorer.Select(IndexRoot)
	feeds, err := m.SubscribeSelected(explorer)
	if err != nil {
		return nil, err
	}
	feed := feeds[0]
	if len(entries) != 0 {
		var assets []*Asset
		assets, err = m.saveAssets(feed.IE, "", entries, feed, dir, quality)
		feed.AssetCount = int64(len(assets))
	}
	return feed, err
}

func (m *Monitor) Convert2SubscribeURL(hintURL string, feedType int) (subscribeURL string, err error) {
	if feedType <= 0 {
		return hintURL, nil
	}
	if feedType == FeedTypeUser {
		if youtube.IsYoutubeURL(hintURL) {
			if kind, _, e := youtube.ParseYoutubeURL(hintURL); e != nil || kind != youtube.KindChannel {
				id, e := youtube.ParseWebpageChannelID(hintURL)
				if e != nil {
					err = e
					return
				}
				subscribeURL, err = youtube.GenYoutubeURL(id, "", "")
			} else {
				subscribeURL = hintURL
			}
		} else if instagram.IsInstragramURL(hintURL) {
			if _, _, err = instagram.ParseInstagramURL(hintURL); err != nil {
				return
			}
			subscribeURL = hintURL
		} else {
			err = fmt.Errorf("unsupported subscribe site")
			return
		}
	} else if feedType == FeedTypePlaylist {
		if youtube.IsYoutubeURL(hintURL) {
			if kind, _, e := youtube.ParseYoutubeURL(hintURL); kind != youtube.KindPlaylist {
				if e == nil {
					e = fmt.Errorf("url not is a playlist")
				}
				err = e
				return
			} else {
				subscribeURL = hintURL
			}
		} else {
			err = fmt.Errorf("playlist unsupported subscribe site")
			return
		}
	} else {
		err = fmt.Errorf("unsupported subscribe type")
		return
	}
	return
}

func (m *Monitor) AddExternalGenericBundle(usedDowner string, bundle *ies.MediaEntry, dir, quality string) (*Bundle, error) {
	if usedDowner == "" {
		return nil, fmt.Errorf("downloader not specified")
	}
	bundles, err := m.saveBundles("external", func(b *Bundle) string {
		return usedDowner
	}, []*ies.MediaEntry{bundle}, BundleTypeGeneric, dir, quality, false)
	if err != nil {
		return nil, err
	}
	return bundles[0], nil
}

func (m *Monitor) AddExternalFeedBundle(feedIE string, bundle *ies.MediaEntry, dir, quality string, saveFeedBundleAssets bool) (*Bundle, error) {
	if feedIE == "" {
		return nil, fmt.Errorf("feedIE not specified")
	}
	bundles, err := m.saveBundles(feedIE, nil, []*ies.MediaEntry{bundle}, BundleTypeFeed, dir, quality, saveFeedBundleAssets)
	if err != nil {
		return nil, err
	}
	return bundles[0], nil
}

func (m *Monitor) AssetbasedSelected(explorer *Explorer, preProcBundle func(*Bundle), dir, quality string) ([]*Bundle, error) {
	switch explorer.RootMediaType() {
	case ies.MediaTypeUser:
		if explorer.firstSelectedIndex() == IndexUser {
			explorer.loadNextAll()
		}
	}

	bundles, err := m.selectedBundlesMedia(explorer, true)
	if err != nil {
		return nil, err
	}
	return m.saveBundles(explorer.ie.Name(), func(b *Bundle) (downer string) {
		if preProcBundle != nil {
			preProcBundle(b)
		}
		return ""
	}, bundles, BundleTypeGeneric, dir, quality, false)
}

func (m *Monitor) StopDownloading(id uint, wait bool) {
	stat, ok := m.getDownloading(id)
	if ok {
		stat.cancel()
	}
	if wait {
		for {
			if _, ok := m.getDownloading(id); !ok {
				return
			}
			time.Sleep(time.Millisecond * 500)
		}
	}
}

func (m *Monitor) StopAllDownloading(wait bool) {
	m.lock.Lock()
	for _, stat := range m.downloading {
		stat.cancel()
	}
	m.lock.Unlock()
	if wait {
		for {
			if m.getDownloadingCount() == 0 {
				return
			}
			time.Sleep(time.Millisecond * 500)
		}
	}
}

func (m *Monitor) StopBundleDownloading(bundleID uint, wait bool) {
	m.lock.Lock()
	for _, stat := range m.downloading {
		if stat.bundleID == bundleID {
			stat.cancel()
		}
	}
	m.lock.Unlock()
	if wait {
		for {
			if len(m.allDownloadingBundleAssetID(bundleID)) == 0 {
				return
			}
			time.Sleep(time.Millisecond * 500)
		}
	}
}

func (m *Monitor) AsyncDownloadAsset(id uint, newAssetDir string,
	begin func(asset *Asset),
	sink_ downloader.ProgressSink,
	result func(asset *Asset, err error),
	notCheckStatus ...bool) error {
	_, err := m.GetAsset(id)
	if err != nil {
		return err
	}
	go func() {
		asset, err := m.DownloadAsset(context.Background(), id, newAssetDir, begin, sink_, notCheckStatus...)
		if result != nil {
			result(asset, err)
		}
	}()
	return nil
}

func (m *Monitor) DownloadAsset(ctx context.Context, id uint, newAssetDir string, begin func(asset *Asset), sink_ downloader.ProgressSink, notCheckStatus ...bool) (*Asset, error) {
	asset, err := m.GetAsset(id)
	if err != nil {
		return asset, err
	}
	if begin != nil {
		begin(asset)
	}
	if asset.DownloadFileStem == "" {
		asset.DownloadFileStem = asset.Title
	}
	if asset.DownloadFileDir == "" {
		asset.DownloadFileDir = newAssetDir
	}
	if !(len(notCheckStatus) > 0 && notCheckStatus[0]) && asset.Status == AssetStatusFinished {
		if fileutil.IsExist(asset.FilePath()) {
			return asset, nil
		}
	}

	if m.externalDownloadingStatManagerFunc.GetExternalDownloadingCount != nil &&
		m.externalDownloadingStatManagerFunc.GetMaxConcurrentCount != nil {
		max := m.externalDownloadingStatManagerFunc.GetMaxConcurrentCount()
		if max > 0 {
			left := max - m.externalDownloadingStatManagerFunc.GetExternalDownloadingCount() - m.getDownloadingCount()
			if left <= 0 {
				return asset, m.externalDownloadingStatManagerFunc.OverMaxConcurrentErr
			}
		}
	}

	sink := func(total, downloaded, speed, eta int64, percent float64) {
		asset.DownloadTotalSize = total
		asset.DownloadedSize = downloaded
		asset.DownloadPercent = percent
		if sink_ != nil {
			sink_(total, downloaded, speed, eta, percent)
		}
	}

	d := downloader.GetByName(asset.Downloader)
	if d == nil {
		return asset, fmt.Errorf("downloader not found: %s", asset.Downloader)
	}

	qualityFormat := ies.Format{}
	if asset.QualityFormat != nil {
		qualityFormat = *asset.QualityFormat
	}
	audioFormat := ies.Format{}
	if asset.AudioFormat != nil {
		audioFormat = *asset.AudioFormat
	}

	ctx, cancel := context.WithCancel(ctx)
	m.addDownloading(&downloadingStat{
		id:       asset.ID,
		bundleID: asset.BundleID,
		cancel:   cancel,
	})
	defer cancel()
	defer m.removeDownloading(asset.ID)

	asset.Status = AssetStatusDownloading
	ok, err := d.Download(ctx, downloader.DownloadOptions{
		URL:                 asset.URL,
		Quality:             asset.Quality,
		DownloadedSize:      asset.DownloadedSize,
		DownloadPercent:     asset.DownloadPercent,
		DownloadFileDir:     asset.DownloadFileDir,
		MainDownloadFormat:  qualityFormat,
		AudioDownloadFormat: audioFormat,

		DownloadFileStem: &asset.DownloadFileStem,
		DownloadFileExt:  &asset.DownloadFileExt,
		DownloaderData:   &asset.DownloaderData,

		RefillInfo: downloader.RefillInfo{
			Title:     &asset.Title,
			Thumbnail: &asset.Thumbnail,
			Duration:  &asset.Duration,
		},
	}, sink)
	if err != nil {
		if common.IsCtxDone(ctx) {
			asset.Status = AssetStatusCanceled
		} else {
			if ok {
				asset.Status = AssetStatusDownloading
			} else {
				asset.Status = AssetStatusFail
			}
		}
	} else {
		asset.Status = AssetStatusFinished
		asset.DownloadTotalSize, _ = fileutil.FileSize(asset.FilePath())
		asset.DownloadPercent = 100
	}

	if e := m.storage.Save(asset); e != nil {
		log.Printf("db save fail: %s", e)
	}
	return asset, err
}

func (m *Monitor) GetDownloadingStatFunc() (
	GetDownloadingCount func() int,
	StopAllDownloading func(),
) {
	GetDownloadingCount = func() int {
		return m.getDownloadingCount()
	}
	StopAllDownloading = func() {
		m.StopAllDownloading(true)
	}
	return
}

func (m *Monitor) selectedBundlesMedia(explorer *Explorer, isDeepDownloadable bool) ([]*ies.MediaEntry, error) {
	selected, err := explorer.Selected(true)
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

/* 每个元素表示一个bundle(user/playlist),元素的子项一定是可以下载的媒体类型 */
func (m *Monitor) bundleMediaEntryDeepAnalysis(url string) []*ies.MediaEntry {
	explorer, err := m.OpenExplorer(url, false)
	if err != nil {
		return nil
	}
	explorer.loadNextAll()

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

func (m *Monitor) saveBundles(ie string, preSaveBundle func(b *Bundle) (downer string), bundleMedias []*ies.MediaEntry, saveBundleType int, dir, quality string, saveFeedBundleAssets bool) (bundles []*Bundle, err error) {
	bundles = make([]*Bundle, 0)
	for _, entry := range bundleMedias {
		if m.storage.IsClosed() {
			err = fmt.Errorf("closed")
			break
		}

		bundle := &Bundle{
			IE:         ie,
			BundleType: saveBundleType,
			FeedType:   mediaType2FeedType(entry.MediaType),
			URL:        entry.URL,
			Title:      entry.Title,
			Thumbnail:  entry.Thumbnail,
			MediaID:    entry.MediaID,
			LastUpdate: time.Now(),
			Uploader:   entry.Uploader,
		}
		if saveBundleType == BundleTypeFeed && bundle.FeedType == 0 {
			err = fmt.Errorf("feed unsupported media type: %d", entry.MediaType)
			continue
		}
		downerName := ""
		if preSaveBundle != nil {
			downerName = preSaveBundle(bundle)
		}
		err = m.storage.Create(bundle)
		if err != nil {
			continue
		}
		if saveFeedBundleAssets || saveBundleType != BundleTypeFeed {
			assets, err := m.saveAssets(ie, downerName, entry.Entries, bundle, dir, quality)
			if err == nil {
				bundle.Assets = assets
				bundle.AssetCount = int64(len(assets))
			}
		}
		bundles = append(bundles, bundle)
	}
	if len(bundles) > 0 {
		err = nil
	}
	return bundles, err
}

func (m *Monitor) saveAssets(ie, downerName string, entryies []*ies.MediaEntry, owner *Bundle, dir, quality string) (retAssets []*Asset, err error) {
	entryies = plain(entryies)
	retAssets = make([]*Asset, 0, len(entryies))

	var downer downloader.Downloader
	if downerName != "" {
		downer = downloader.GetByName(downerName)
	}
	if downer == nil {
		downer = downloader.GetByIE(ie)
	}

	if quality == "" {
		quality = "best"
	}

	lastBeginStem := ""
	stemSuffIndex := 1
	for _, entry := range entryies {
		if m.storage.IsClosed() {
			err = fmt.Errorf("closed")
			break
		}

		var qualityFormat *ies.Format
		var audioFormat *ies.Format
		if downer.IsNeedFormat() {
			if len(entry.Formats) == 0 {
				err = fmt.Errorf("no format found for entry: %s", entry.URL)
				continue
			}
			qualityFormat = entry.Formats[selectQualityFormatByResolution(entry.Formats, quality)]
			if ies.MediaTypeVideo == entry.MediaType && qualityFormat.FormatType == ies.FormatTypeVideo {
				if audioIndex := selectAudioFormatByResolution(entry.Formats, quality); audioIndex >= 0 {
					audioFormat = entry.Formats[audioIndex]
				}
			}
		}
		asset := &Asset{
			Status: AssetStatusNew,

			Title:         entry.Title,
			URL:           entry.URL,
			Quality:       quality,
			Thumbnail:     entry.Thumbnail,
			QualityFormat: qualityFormat,
			AudioFormat:   audioFormat,

			DownloadFileDir: dir,
			Downloader:      downer.Name(),
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
		if asset.Title == "" {
			if owner != nil {
				asset.Title = owner.IE + "(" + time.Now().Format("2006-01-02 15-04-05") + ")"
			} else {
				asset.Title = time.Now().Format("2006-01-02 15-04-05")
			}
		}

		if owner != nil {
			asset.BundleTitle = owner.Title
			asset.BundleID = owner.ID
		}

		stem := common.ReplaceWrongFileChars(asset.Title)
		if len(stem) > 100 {
			stem = stem[:100]
		}
		if stem != lastBeginStem {
			lastBeginStem = stem
			stemSuffIndex = 1
		} else {
			stem = fmt.Sprintf("%s(%d)", stem, stemSuffIndex)
			stemSuffIndex++
		}

		asset.DownloadFileStem = stem

		if err = m.storage.Create(asset); err == nil {
			retAssets = append(retAssets, asset)
		}
	}
	if len(retAssets) > 0 {
		err = nil
	}
	return
}

func (m *Monitor) addDownloading(stat *downloadingStat) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.downloading[stat.id] = stat
}

func (m *Monitor) removeDownloading(id uint) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.downloading, id)
}

func (m *Monitor) getDownloading(id uint) (*downloadingStat, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	stat, ok := m.downloading[id]
	return stat, ok
}

func (m *Monitor) getDownloadingsID() []uint {
	m.lock.RLock()
	defer m.lock.RUnlock()
	ids := make([]uint, 0, len(m.downloading))
	for id := range m.downloading {
		ids = append(ids, id)
	}
	return ids
}

func (m *Monitor) getDownloadingCount() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.downloading)
}

func (m *Monitor) allDownloadingBundleAssetID(bundleID uint) []uint {
	m.lock.RLock()
	defer m.lock.RUnlock()
	ids := make([]uint, 0, len(m.downloading))
	for _, stat := range m.downloading {
		if stat.bundleID == bundleID {
			ids = append(ids, stat.id)
		}
	}
	return ids
}

func mediaType2FeedType(mediaType int) int {
	if mediaType == ies.MediaTypeUser {
		return FeedTypeUser
	} else if mediaType == ies.MediaTypePlaylist {
		return FeedTypePlaylist
	} else {
		return 0
	}
}
