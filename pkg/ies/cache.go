package ies

import (
	"time"

	"github.com/yinyajiang/yt-mnt/model"
)

type cacheInfo struct {
	timeStamp time.Time
	media     *model.MediaEntry
	url       string
}

type cacheInfoExtractor struct {
	ie    InfoExtractor
	cache []cacheInfo
}

func (c *cacheInfoExtractor) Parse(link string) (*model.MediaEntry, error) {
	isMatchedCache := func(cache cacheInfo) bool {
		if cache.url == link && time.Since(cache.timeStamp) < time.Minute*5 {
			return true
		}
		return false
	}

	for _, cache := range c.cache {
		if isMatchedCache(cache) {
			return cache.media, nil
		}
		if cache.media.MediaType == model.MediaTypePlaylistGroup {
			for _, entry := range cache.media.Entries {
				if isMatchedCache(cacheInfo{
					timeStamp: cache.timeStamp,
					media:     entry,
					url:       link,
				}) {
					return entry, nil
				}
			}
		}
	}

	media, err := c.ie.Parse(link)
	if err == nil {
		c.cache = append(c.cache, cacheInfo{
			timeStamp: time.Now(),
			media:     media,
			url:       link,
		})
	}
	return media, err
}

func (c *cacheInfoExtractor) ExtractPage(linkInfo LinkInfo, nextPage *NextPage) ([]*model.MediaEntry, error) {
	return c.ie.ExtractPage(linkInfo, nextPage)
}

func (c *cacheInfoExtractor) UpdateMedia(update *model.MediaEntry) error {
	return c.ie.UpdateMedia(update)
}

func (c *cacheInfoExtractor) IsMatched(url string) bool {
	return c.ie.IsMatched(url)
}

func (c *cacheInfoExtractor) Name() string {
	return c.ie.Name()
}
