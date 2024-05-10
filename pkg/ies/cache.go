package ies

import (
	"time"
)

type cacheInfo struct {
	timeStamp time.Time
	info      *MediaEntry
	url       string
}

type cacheInfoExtractor struct {
	ie    InfoExtractor
	cache []cacheInfo
}

func (c *cacheInfoExtractor) Parse(link string, options ...ParseOptions) (*MediaEntry, error) {
	isMatchedCache := func(cache cacheInfo) bool {
		if cache.url == link && time.Since(cache.timeStamp) < time.Minute*30 {
			return true
		}
		return false
	}

	for _, cache := range c.cache {
		if isMatchedCache(cache) {
			return cache.info, nil
		}
	}

	media, err := c.ie.Parse(link, options...)
	if err == nil {
		c.cache = append(c.cache, cacheInfo{
			timeStamp: time.Now(),
			info:      media,
			url:       link,
		})
	}
	return media, err
}

func (c *cacheInfoExtractor) ExtractPage(linkInfo LinkInfo, nextPage *NextPage) ([]*MediaEntry, error) {
	entrys, err := c.ie.ExtractPage(linkInfo, nextPage)
	if err == nil {
		for _, entry := range entrys {
			sortEntryFormats(entry)
		}
	}
	return entrys, err
}

func (c *cacheInfoExtractor) ExtractAllAfterTime(paretnMediaID string, afterTime time.Time) ([]*MediaEntry, error) {
	entrys, err := c.ie.ExtractAllAfterTime(paretnMediaID, afterTime)
	if err == nil {
		for _, entry := range entrys {
			sortEntryFormats(entry)
		}
	}
	return entrys, err
}

func (c *cacheInfoExtractor) IsMatched(url string) bool {
	return c.ie.IsMatched(url)
}

func (c *cacheInfoExtractor) Name() string {
	return c.ie.Name()
}
