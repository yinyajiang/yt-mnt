package ies

import (
	"log"
	"math"

	"github.com/yinyajiang/yt-mnt/model"

	"github.com/duke-git/lancet/v2/slice"
)

type GetSubItemCount = func(mediaID string) (int64, error)
type GetSubItems = func(mediaID string, latestCount ...int64) ([]*model.MediaEntry, error)
type GetSubItemsWithPage = func(mediaID string, nextPage *NextPage) ([]*model.MediaEntry, error)

func HelperUpdateSubItems(entry *model.MediaEntry, getSubItemCount GetSubItemCount, getSubItems GetSubItems) error {
	curCount, err := getSubItemCount(entry.MediaID)
	if curCount == 0 || err != nil {
		return err
	}
	if curCount == entry.QueryEntryCount {
		return nil
	}
	latestCount := curCount - entry.QueryEntryCount
	if entry.QueryEntryCount == 0 {
		latestCount = math.MaxInt64
	}
	items, err := getSubItems(entry.MediaID, latestCount)
	if err != nil {
		return err
	}
	for _, item := range items {
		if slice.ContainBy(entry.Entries, func(cur *model.MediaEntry) bool {
			return item.MediaID == cur.MediaID
		}) {
			continue
		}
		item.SetNew(true)
		entry.Entries = append(entry.Entries, item)
	}
	entry.QueryEntryCount = curCount
	return nil
}

func HelperGetSubItems(mediaID string, getSubItemsWithPageID GetSubItemsWithPage, latestCount ...int64) ([]*model.MediaEntry, error) {
	leftCount := int64(0)
	if len(latestCount) > 0 {
		leftCount = latestCount[0]
	}
	if leftCount <= 0 {
		leftCount = math.MaxInt64
	}
	ret := make([]*model.MediaEntry, 0)
	nextPage := NextPage{}
	for {
		if leftCount <= 0 || nextPage.IsEnd {
			break
		}
		medias, err := getSubItemsWithPageID(mediaID, &nextPage)
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

func ToLinkInfo(m *model.MediaEntry) LinkInfo {
	return LinkInfo{
		LinkID:    m.LinkID,
		MediaID:   m.MediaID,
		MediaType: m.MediaType,
	}
}

func Options(options []ParseOptions) ParseOptions {
	if len(options) == 0 {
		return ParseOptions{
			MustCount: true,
		}
	}
	return options[0]
}
