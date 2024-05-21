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
	_ "github.com/yinyajiang/yt-mnt/pkg/ies/instagram"
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

	lock        sync.RWMutex
	downloading map[uint]*downloadingStat
}

type MonitorOption struct {
	Verbose          bool
	IEToken          ies.IETokens
	AssetTableName   string
	BundleTableName  string
	RegistDownloader []downloader.Downloader
	DBOption         db.DBOption
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
			__tabname: opt.AssetTableName,
		},
		&Bundle{
			__tabname: opt.BundleTableName,
		},
	)
	if err != nil {
		return nil, err
	}

	m := &Monitor{
		storage:     storage,
		_db:         storage.GormDB(),
		downloading: make(map[uint]*downloadingStat),
	}
	return m, nil
}

func (m *Monitor) Close() {
	m.StopAllDownloading(true)
	m.storage.Close()
}

func (m *Monitor) LocalDownloaderStageSaver(dir string) downloader.DownloaderStageSaver {
	return downloader.NewLocalDirStageSaver(dir)
}

func (m *Monitor) CreateExplorerCacher(cacheCount ...int) *ExplorerCacher {
	return NewExplorerCacher(cacheCount...)
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
	explorer.url = url
	explorer.time = time.Now()
	explorer.rootInfo = *info
	explorer.rootToken = *rootToken
	explorer.pageIndex = 0
	return &explorer, nil
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
	newAssets, err = m.saveAssets(feed.IE, newEntries, &feed, dir, quality)
	if err != nil {
		return
	}
	err = m._db.Updates(&Bundle{
		Model: gorm.Model{
			ID: feedid,
		},
		LastUpdate: time.Now(),
	}).Error
	return
}

func (m *Monitor) Unsubscribe(id uint) error {
	err := m._db.Where(&Bundle{
		Model: gorm.Model{
			ID: id,
		},
	}).Updates(&Bundle{
		BundleType: BundleTypeGeneric,
	}).Error
	return err
}

func (m *Monitor) DeleteAsset(id uint) {
	if id <= 0 {
		return
	}
	m.StopDownloading(id, true)
	m._db.Unscoped().Delete(&Asset{
		Model: gorm.Model{
			ID: id,
		},
	})
}

func (m *Monitor) DeleteBundle(id uint) {
	if id <= 0 {
		return
	}
	ids := m.allDownloadingBundleAssetID(id)
	for _, id := range ids {
		m.StopDownloading(id, true)
	}
	m._db.Unscoped().Delete(&Bundle{
		Model: gorm.Model{
			ID: id,
		},
	})
	m._db.Unscoped().Delete(&Asset{
		BundleID: id,
	})
}

func (m *Monitor) Clear() {
	m.StopAllDownloading(true)
	m._db.Unscoped().Where("1 = 1").Delete(&Bundle{})
	m._db.Unscoped().Where("1 = 1").Delete(&Asset{})
}

func (m *Monitor) GetBundle(id uint, preload bool, assetCount bool) (*Bundle, error) {
	var bundle Bundle
	var tx *gorm.DB
	if preload {
		tx = m._db.Preload(clause.Associations)
	} else {
		tx = m._db
	}
	err := tx.First(&bundle, Bundle{
		Model: gorm.Model{
			ID: id,
		},
	}).Error

	if err == nil && assetCount {
		tx.Model(&Asset{}).Where(&Asset{
			BundleID: id,
		}).Count(&bundle.AssetCount)
	}
	return &bundle, err
}

func (m *Monitor) ListFeedBundles(preload bool, assetCount bool) ([]*Bundle, error) {
	return m.ListBundlesByWhere(preload, assetCount, &Bundle{
		BundleType: BundleTypeFeed,
	})
}

func (m *Monitor) ListGenericBundles(preload bool, assetCount bool) ([]*Bundle, error) {
	return m.ListBundlesByWhere(preload, assetCount, &Bundle{
		BundleType: BundleTypeGeneric,
	})
}

func (m *Monitor) ListAllBundles(preload bool, assetCount bool) ([]*Bundle, error) {
	return m.ListBundlesByWhere(preload, assetCount, nil)
}

func (m *Monitor) ListBundlesByWhere(preload bool, assetCount bool, where *Bundle) ([]*Bundle, error) {
	var bundles []*Bundle
	var err error
	var tx *gorm.DB
	if preload {
		tx = m._db.Preload(clause.Associations)
	} else {
		tx = m._db
	}
	if where != nil {
		err = tx.Find(&bundles, where).Error
	} else {
		err = tx.Find(&bundles).Error
	}

	if err == nil && assetCount {
		for _, bundle := range bundles {
			m._db.Model(&Asset{}).Where(&Asset{
				BundleID: bundle.ID,
			}).Count(&bundle.AssetCount)
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
	}).Find(&assets).Error
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
	for _, entry := range selectedBundles {
		feedType := mediaType2FeedType(entry.MediaType)
		if feedType == 0 {
			err = fmt.Errorf("feed unsupported media type: %d", entry.MediaType)
			continue
		}
		var count int64
		m._db.Where(&Bundle{
			URL:        entry.URL,
			BundleType: BundleTypeFeed,
			FeedType:   feedType,
		}).Count(&count)
		if count > 0 {
			err = fmt.Errorf("feed already exists: %s", entry.URL)
		} else {
			saveBundles = append(saveBundles, entry)
		}
	}
	if len(saveBundles) == 0 {
		return nil, err
	}
	return m.saveBundles(explorer.ie.Name(), saveBundles, BundleTypeFeed, "", "")
}

func (m *Monitor) AssetbasedSelected(explorer *Explorer, renameBundle func(title string) string, dir, quality string) ([]*Bundle, error) {
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
	if renameBundle != nil {
		for _, bundle := range bundles {
			bundle.Title = renameBundle(bundle.Title)
		}
	}
	return m.saveBundles(explorer.ie.Name(), bundles, BundleTypeGeneric, dir, quality)
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
			if m.isDownloadingEmpty() {
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

func (m *Monitor) DownloadAsset(ctx context.Context, id uint, newAssetDir string, sink_ downloader.ProgressSink) (*Asset, error) {
	asset, err := m.GetAsset(id)
	if err != nil {
		return asset, err
	}
	if asset.DownloadFileStem == "" {
		asset.DownloadFileStem = asset.Title
	}
	if asset.DownloadFileDir == "" {
		asset.DownloadFileDir = newAssetDir
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

	ctx, cancel := context.WithCancel(ctx)
	m.addDownloading(&downloadingStat{
		id:       asset.ID,
		bundleID: asset.BundleID,
		cancel:   cancel,
	})
	defer cancel()
	defer m.removeDownloading(asset.ID)

	ok, err := d.Download(ctx, downloader.DownloadOptions{
		URL:             asset.URL,
		Quality:         asset.Quality,
		DownloadedSize:  asset.DownloadedSize,
		DownloadPercent: asset.DownloadPercent,
		DownloadFileDir: asset.DownloadFileDir,

		DownloadFileStem: &asset.DownloadFileStem,
		DownloadFileExt:  &asset.DownloadFileExt,
		DownloadFormat:   qualityFormat,
		DownloaderData:   &asset.DownloaderData,
	}, sink)
	if err != nil {
		if ok {
			asset.Status = AssetStatusDownloading
		} else {
			asset.Status = AssetStatusFail
		}
	} else {
		asset.Status = AssetStatusFinished
		asset.DownloadTotalSize, _ = fileutil.FileSize(asset.FilePath())
		asset.DownloadPercent = 100
	}
	if e := m._db.Save(asset).Error; e != nil {
		log.Printf("db save fail: %s", e)
	}
	return asset, err
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

func (m *Monitor) saveBundles(ie string, bundleMedias []*ies.MediaEntry, saveBundleType int, dir, quality string) (bundles []*Bundle, err error) {
	bundles = make([]*Bundle, 0)
	for _, entry := range bundleMedias {
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
		err = m._db.Create(bundle).Error
		if err != nil {
			continue
		}
		if saveBundleType != BundleTypeFeed {
			assets, err := m.saveAssets(ie, entry.Entries, bundle, dir, quality)
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

func (m *Monitor) saveAssets(ie string, entryies []*ies.MediaEntry, owner *Bundle, dir, quality string) (retAssets []*Asset, err error) {
	entryies = plain(entryies)
	retAssets = make([]*Asset, 0, len(entryies))
	downer := downloader.GetByIE(ie)

	if quality == "" {
		quality = "best"
	}

	lastBeginStem := ""
	stemSuffIndex := 1
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
			var count int64
			m._db.Where(&Asset{
				DownloadFileDir:  dir,
				DownloadFileStem: stem,
			}).Count(&count)
			if count > 0 {
				stem = fmt.Sprintf("%s(%d)", stem, stemSuffIndex)
				stemSuffIndex++
			}
		} else {
			stem = fmt.Sprintf("%s(%d)", stem, stemSuffIndex)
			stemSuffIndex++
		}

		asset.DownloadFileStem = stem
		if err = m._db.Create(asset).Error; err == nil {
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

func (m *Monitor) isDownloadingEmpty() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.downloading) == 0
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
