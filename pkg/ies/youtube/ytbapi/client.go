package ytbapi

import (
	"context"
	"github.com/yinyajiang/yt-mnt/model"
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

func (c *Client) Channel(chnnelID string) (*model.MediaEntry, error) {
	var channelPart = []string{"snippet", "contentDetails", "statistics"}
	call := c.service.Channels.List(channelPart)
	call = call.Id(chnnelID)
	response, err := call.Do()
	if err != nil {
		return nil, err
	}
	item := response.Items[0]
	ret := &model.MediaEntry{
		URL:  "https://www.youtube.com/channel/" + chnnelID,
		Note: "youtube-channel",
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

func (c *Client) Playlist(playlistID string) (*model.MediaEntry, error) {
	var playlistPart = []string{"snippet", "contentDetails"}
	call := c.service.Playlists.List(playlistPart)
	call = call.Id(playlistID)
	response, err := call.Do()
	if err != nil {
		return nil, err
	}
	item := response.Items[0]
	ret := &model.MediaEntry{
		URL:     "https://www.youtube.com/playlist?list=" + playlistID,
		Note:    "youtube-playlist",
		MediaID: playlistID,
	}
	if item.Snippet != nil {
		ret.Title = item.Snippet.Title
		ret.Description = item.Snippet.Description
	}
	if item.Snippet.Thumbnails != nil && item.Snippet.Thumbnails.Default != nil {
		ret.Thumbnail = item.Snippet.Thumbnails.Default.Url
	}
	if item.ContentDetails != nil {
		ret.QueryEntryCount = item.ContentDetails.ItemCount
	}
	return ret, nil
}

func (c *Client) PlaylistsVideo(playlistID string, latestCount ...int64) ([]*model.MediaEntry, error) {
	return ies.HelperGetSubItems(playlistID,
		func(playlistQueryID string, nextPage *model.NextPage) ([]*model.MediaEntry, error) {
			return c.PlaylistsVideoWithPage(playlistQueryID, nextPage, -1)
		}, latestCount...)
}

func (c *Client) PlaylistsVideoWithPage(playlistID string, nextPage *model.NextPage, maxCount int64) ([]*model.MediaEntry, error) {
	if nextPage == nil {
		return c.PlaylistsVideo(playlistID, maxCount)
	}
	if nextPage.IsEnd {
		return nil, nil
	}
	var playlistsVideoPart = []string{"snippet", "contentDetails"}
	if maxCount > 50 || maxCount <= 0 {
		maxCount = 50
	}

	call := c.service.PlaylistItems.List(playlistsVideoPart).PlaylistId(playlistID).MaxResults(maxCount)
	if nextPage.NextPageID != "" {
		call = call.PageToken(nextPage.NextPageID)
	}
	response, err := call.Do()
	if err != nil {
		return nil, err
	}
	ret := []*model.MediaEntry{}
	for _, item := range response.Items {
		video := &model.MediaEntry{}
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
			video.MediaType = model.MediaTypeVideo
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

func (c *Client) ChannelsPlaylist(chnnelID string) ([]*model.MediaEntry, error) {
	return ies.HelperGetSubItems(chnnelID,
		func(chnnelID string, nextPage *model.NextPage) ([]*model.MediaEntry, error) {
			return c.ChannelsPlaylistWithPage(chnnelID, nextPage)
		})
}

func (c *Client) ChannelsPlaylistWithPage(chnnelID string, nextPage *model.NextPage) ([]*model.MediaEntry, error) {
	if nextPage == nil {
		return c.ChannelsPlaylist(chnnelID)
	}
	if nextPage.IsEnd {
		return nil, nil
	}

	var channelsPlaylistPart = []string{"id", "snippet", "contentDetails"}
	call := c.service.Playlists.List(channelsPlaylistPart).ChannelId(chnnelID).MaxResults(50)
	if nextPage.NextPageID != "" {
		call = call.PageToken(nextPage.NextPageID)
	}
	response, err := call.Do()
	if err != nil {
		return nil, err
	}

	ret := []*model.MediaEntry{}
	for _, item := range response.Items {
		playlist := &model.MediaEntry{
			Note:    "youtube-playlist",
			MediaID: item.Id,
			URL:     "https://www.youtube.com/playlist?list=" + item.Id,
		}
		if item.Snippet != nil {
			playlist.Title = item.Snippet.Title
			playlist.Description = item.Snippet.Description
			if item.Snippet.Thumbnails != nil && item.Snippet.Thumbnails.Default != nil {
				playlist.Thumbnail = item.Snippet.Thumbnails.Default.Url
			}
		}
		if item.ContentDetails != nil {
			playlist.QueryEntryCount = item.ContentDetails.ItemCount
		}
		ret = append(ret, playlist)
	}
	nextPage.NextPageID = response.NextPageToken
	if nextPage.NextPageID == "" {
		nextPage.IsEnd = true
	}
	return ret, nil
}
