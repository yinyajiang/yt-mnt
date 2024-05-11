package ies

import (
	"log"
	"math"
	"time"

	"github.com/duke-git/lancet/v2/slice"
)

type GetSubItemCount = func(parentID string) (int64, error)
type GetSubItemsOrderWithPage = func(parentID string, nextPage *NextPageToken) ([]*MediaEntry, error)
type GetSubItemsWithPage = func(parentID string, nextPage *NextPageToken) ([]*MediaEntry, error)

func HelperGetSubItemsByTime(parentID string, getSubItemsWithPageID GetSubItemsOrderWithPage, afterTime time.Time) (retItems []*MediaEntry, err error) {
	retItems = make([]*MediaEntry, 0)
	if afterTime.IsZero() {
		retItems, err = HelperGetSubItems(parentID, getSubItemsWithPageID)
		if err != nil {
			return
		}
	} else {
		nextPage := NextPageToken{}
	pageloop:
		for {
			if nextPage.IsEnd {
				break
			}
			pageItems, e := getSubItemsWithPageID(parentID, &nextPage)
			if e != nil {
				if len(retItems) == 0 {
					err = e
					return
				}
				break pageloop
			}
			for _, item := range pageItems {
				if item.UploadDate.Before(afterTime) {
					break pageloop
				}
				retItems = append(retItems, item)
			}
		}
	}
	return
}

func HelperGetSubItems(mediaID string, getSubItemsWithPageID GetSubItemsWithPage, latestCount ...int64) ([]*MediaEntry, error) {
	leftCount := int64(0)
	if len(latestCount) > 0 {
		leftCount = latestCount[0]
	}
	if leftCount <= 0 {
		leftCount = math.MaxInt64
	}
	ret := make([]*MediaEntry, 0)
	nextPage := NextPageToken{}
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

func HelperGetSubItemsByDiffCount(mediaID string, beforeCount int64, beforeSubItems []*MediaEntry, outAllCount *int64, getSubItemCount GetSubItemCount, getSubItemsWithPageID GetSubItemsWithPage) (retItems []*MediaEntry, err error) {
	retItems = make([]*MediaEntry, 0)
	allCount, err := getSubItemCount(mediaID)
	if allCount == 0 || err != nil {
		return
	}
	if allCount == beforeCount {
		return
	}
	if outAllCount != nil {
		*outAllCount = allCount
	}

	latestCount := allCount - beforeCount
	if beforeCount <= 0 {
		latestCount = math.MaxInt64
	}
	latestItems, err := HelperGetSubItems(mediaID, getSubItemsWithPageID, latestCount)
	if err != nil {
		return
	}
	for _, item := range latestItems {
		if slice.ContainBy(beforeSubItems, func(cur *MediaEntry) bool {
			return item.MediaID == cur.MediaID
		}) {
			continue
		}
		retItems = append(retItems, item)
	}
	return
}
