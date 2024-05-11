package ytbapi

import (
	"context"
	"errors"

	"github.com/yinyajiang/yt-mnt/pkg/ies"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

type Client struct {
	service *youtube.Service
}

func New(apiKey string) (*Client, error) {
	service, err := youtube.NewService(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	c := &Client{
		service: service,
	}
	return c, nil
}

func (c *Client) Channel(chnnelID string) (*ies.MediaEntry, error) {
	var channelPart = []string{"snippet", "contentDetails", "statistics"}
	call := c.service.Channels.List(channelPart)
	call = call.Id(chnnelID)
	response, err := call.Do()
	if err != nil {
		return nil, err
	}
	item := response.Items[0]
	ret := &ies.MediaEntry{
		URL:       "https://www.youtube.com/channel/" + chnnelID,
		MediaType: ies.MediaTypeUser,
	}
	if item.Snippet != nil {
		ret.Title = item.Snippet.Title
		ret.Description = item.Snippet.Description
	}
	if item.Snippet.Thumbnails != nil && item.Snippet.Thumbnails.Default != nil {
		ret.Thumbnail = item.Snippet.Thumbnails.Default.Url
	}
	if item.ContentDetails != nil && item.ContentDetails.RelatedPlaylists != nil {
		ret.MediaID = item.ContentDetails.RelatedPlaylists.Uploads
	}
	if item.Statistics != nil {
		ret.EntryCount = int64(item.Statistics.VideoCount)
	}
	return ret, nil
}

func (c *Client) PlaylistsVideoCount(playlistID string) (int64, error) {
	call := c.service.Playlists.List([]string{"contentDetails"})
	call = call.Id(playlistID)
	response, err := call.Do()
	if err != nil {
		return 0, err
	}
	return response.Items[0].ContentDetails.ItemCount, nil
}

func (c *Client) Playlist(playlistID string) (*ies.MediaEntry, error) {
	var playlistPart = []string{"snippet", "contentDetails"}
	call := c.service.Playlists.List(playlistPart)
	call = call.Id(playlistID)
	response, err := call.Do()
	if err != nil {
		return nil, err
	}
	item := response.Items[0]
	ret := &ies.MediaEntry{
		URL:       "https://www.youtube.com/playlist?list=" + playlistID,
		MediaID:   playlistID,
		MediaType: ies.MediaTypePlaylist,
	}
	if item.Snippet != nil {
		ret.Title = item.Snippet.Title
		ret.Description = item.Snippet.Description
	}
	if item.Snippet.Thumbnails != nil && item.Snippet.Thumbnails.Default != nil {
		ret.Thumbnail = item.Snippet.Thumbnails.Default.Url
	}
	if item.ContentDetails != nil {
		ret.EntryCount = item.ContentDetails.ItemCount
	}
	return ret, nil
}

func (c *Client) PlaylistsVideo(playlistID string, latestCount ...int64) ([]*ies.MediaEntry, error) {
	return ies.HelperGetSubItems(playlistID,
		func(playlistQueryID string, nextPage *ies.NextPageToken) ([]*ies.MediaEntry, error) {
			return c.PlaylistsVideoWithPage(playlistQueryID, nextPage)
		}, latestCount...)
}

func (c *Client) PlaylistsVideoWithPage(playlistID string, nextPage *ies.NextPageToken) ([]*ies.MediaEntry, error) {
	if nextPage == nil {
		return c.PlaylistsVideo(playlistID, -1)
	}
	if nextPage.IsEnd {
		return nil, nil
	}
	if nextPage.HintPageCount <= 0 {
		nextPage.HintPageCount = 50
	}

	var playlistsVideoPart = []string{"snippet", "contentDetails"}

	call := c.service.PlaylistItems.List(playlistsVideoPart).PlaylistId(playlistID).MaxResults(nextPage.HintPageCount)
	if nextPage.NextPageID != "" {
		call = call.PageToken(nextPage.NextPageID)
	}
	response, err := call.Do()
	if err != nil {
		return nil, err
	}
	ret := []*ies.MediaEntry{}
	for _, item := range response.Items {
		video := &ies.MediaEntry{}
		if item.Snippet != nil {
			video.Title = item.Snippet.Title
			video.Description = item.Snippet.Description
			if item.Snippet.Thumbnails != nil && item.Snippet.Thumbnails.Default != nil {
				video.Thumbnail = item.Snippet.Thumbnails.Default.Url
			}
			if item.Snippet.ResourceId != nil {
				video.MediaID = item.Snippet.ResourceId.VideoId
			}
			video.URL = "https://www.youtube.com/watch?v=" + video.MediaID
			video.MediaType = ies.MediaTypeVideo
		}
		if video.MediaID == "" {
			continue
		}
		if item.ContentDetails != nil {
			video.UploadDate, _ = parseDate(item.ContentDetails.VideoPublishedAt)
		}
		ret = append(ret, video)
	}
	nextPage.NextPageID = response.NextPageToken
	if nextPage.NextPageID == "" {
		nextPage.IsEnd = true
	}
	return ret, nil
}

func (c *Client) ChannelsPlaylist(chnnelID string) ([]*ies.MediaEntry, error) {
	return ies.HelperGetSubItems(chnnelID,
		func(chnnelID string, nextPage *ies.NextPageToken) ([]*ies.MediaEntry, error) {
			return c.ChannelsPlaylistWithPage(chnnelID, nextPage)
		})
}

func (c *Client) ChannelsPlaylistCount(chnnelID string) (int64, error) {
	var channelsPlaylistPart = []string{"id", "snippet", "contentDetails"}
	call := c.service.Playlists.List(channelsPlaylistPart).ChannelId(chnnelID).MaxResults(1)
	response, err := call.Do()
	if err != nil {
		return 0, err
	}
	if response.PageInfo == nil {
		return 0, errors.New("no page info")
	}
	count := int64(response.PageInfo.TotalResults)
	return count, nil
}

func (c *Client) ChannelsPlaylistWithPage(chnnelID string, nextPage *ies.NextPageToken) ([]*ies.MediaEntry, error) {
	if nextPage == nil {
		return c.ChannelsPlaylist(chnnelID)
	}
	if nextPage.IsEnd {
		return nil, nil
	}
	if nextPage.HintPageCount <= 0 {
		nextPage.HintPageCount = 50
	}

	var channelsPlaylistPart = []string{"id", "snippet", "contentDetails"}
	call := c.service.Playlists.List(channelsPlaylistPart).ChannelId(chnnelID).MaxResults(nextPage.HintPageCount)
	if nextPage.NextPageID != "" {
		call = call.PageToken(nextPage.NextPageID)
	}
	response, err := call.Do()
	if err != nil {
		return nil, err
	}

	ret := []*ies.MediaEntry{}
	for _, item := range response.Items {
		playlist := &ies.MediaEntry{
			MediaID:   item.Id,
			URL:       "https://www.youtube.com/playlist?list=" + item.Id,
			MediaType: ies.MediaTypePlaylist,
		}
		if item.Snippet != nil {
			playlist.Title = item.Snippet.Title
			playlist.Description = item.Snippet.Description
			if item.Snippet.Thumbnails != nil && item.Snippet.Thumbnails.Default != nil {
				playlist.Thumbnail = item.Snippet.Thumbnails.Default.Url
			}
			playlist.UploadDate, _ = parseDate(item.Snippet.PublishedAt)
		}
		if item.ContentDetails != nil {
			playlist.EntryCount = item.ContentDetails.ItemCount
		}
		ret = append(ret, playlist)
	}
	nextPage.NextPageID = response.NextPageToken
	if nextPage.NextPageID == "" {
		nextPage.IsEnd = true
	}
	return ret, nil
}
