package ies

import (
	"time"
)

type cacheInfo struct {
	timeStamp time.Time
	rootInfo  *MediaEntry
	rootToken *RootToken
	url       string
}

type middleInfoExtractor struct {
	ie    InfoExtractor
	cache []cacheInfo
}

func (m *middleInfoExtractor) ParseRoot(link string, options ...ParseOptions) (*MediaEntry, *RootToken, error) {
	if len(m.cache) > 10 {
		m.cache = make([]cacheInfo, 0)
	}

	for _, cache := range m.cache {
		if cache.url == link && time.Since(cache.timeStamp) < time.Minute*30 {
			return cache.rootInfo, cache.rootToken, nil
		}
	}

	rootInfo, rootToken, err := m.ie.ParseRoot(link, options...)
	if err == nil {
		m.cache = append(m.cache, cacheInfo{
			timeStamp: time.Now(),
			rootInfo:  rootInfo,
			rootToken: rootToken,
			url:       link,
		})
	}
	return rootInfo, rootToken, err
}

func (m *middleInfoExtractor) ConvertToUserRoot(rootToken *RootToken, rootInfo *MediaEntry) error {
	if rootToken != nil && rootInfo != nil {
		if rootToken.MediaType == MediaTypeUser && rootInfo.MediaType == MediaTypeUser {
			return nil
		}
	}
	return m.ie.ConvertToUserRoot(rootToken, rootInfo)
}

func (m *middleInfoExtractor) ExtractPage(root *RootToken, nextPage *NextPageToken) ([]*MediaEntry, error) {
	entrys, err := m.ie.ExtractPage(root, nextPage)
	if err == nil {
		for _, entry := range entrys {
			sortEntryFormats(entry)
		}
	}
	return entrys, err
}

func (m *middleInfoExtractor) ExtractAllAfterTime(parentMediaID string, afterTime time.Time) ([]*MediaEntry, error) {
	entrys, err := m.ie.ExtractAllAfterTime(parentMediaID, afterTime)
	if err == nil {
		for _, entry := range entrys {
			sortEntryFormats(entry)
		}
	}
	return entrys, err
}

func (m *middleInfoExtractor) IsMatched(url string) bool {
	return m.ie.IsMatched(url)
}

func (m *middleInfoExtractor) Name() string {
	return m.ie.Name()
}
