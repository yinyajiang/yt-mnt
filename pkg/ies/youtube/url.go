package youtube

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

const (
	KindChannel       = "channel"
	KindPlaylist      = "playlist"
	KindPlaylistGroup = "playlist_group"
)

func IsYoutubeURL(link string) bool {
	return strings.Contains(link, "youtube.com")
}

func GenYoutubeURL(channle, usr, playlist string) (url string, err error) {
	if channle != "" {
		url = "https://www.youtube.com/channel/" + channle
	} else if usr != "" {
		if !strings.HasPrefix(usr, "@") {
			usr = "@" + usr
		}
		url = "https://www.youtube.com/" + usr + "/playlists"
	} else if playlist != "" {
		url = "https://www.youtube.com/playlist?list=" + playlist
	} else {
		err = errors.New("youtube user or playlistID is required")
	}
	return
}

func ParseYoutubeURL(link string) (kind, id string, err error) {
	if !strings.HasPrefix(link, "http") {
		link = "https://" + link
	}
	parsed, err := url.Parse(link)
	if err != nil {
		log.Println(err)
		return
	}

	path := parsed.EscapedPath()

	// https://www.youtube.com/playlist?list=PLCB9F975ECF01953C
	// https://www.youtube.com/watch?v=rbCbho7aLYw&list=PLMpEfaKcGjpWEgNtdnsvLX6LzQL0UC0EM
	if strings.HasPrefix(path, "/playlist") || strings.HasPrefix(path, "/watch") {
		id = parsed.Query().Get("list")
		if id == "" {
			err = errors.New("invalid playlist link")
			return
		}
		kind = KindPlaylist
		return
	}

	// - https://www.youtube.com/channel/UC5XPnUk8Vvv_pWslhwom6Og
	// - https://www.youtube.com/channel/UCrlakW-ewUT8sOod6Wmzyow/videos
	if strings.HasPrefix(path, "/channel") {
		parts := strings.Split(parsed.EscapedPath(), "/")
		if len(parts) <= 2 || parts[2] == "" {
			err = errors.New("invalid youtube channel link")
			return
		}
		kind = KindChannel
		id = parts[2]
		return
	}

	// - https://www.youtube.com/@fxigr1/playlists
	if strings.HasPrefix(path, "/@") && (strings.HasSuffix(path, "/playlists") || strings.HasSuffix(path, "/playlists/")) {
		id, err = ParseWebpageChannelID(parsed.String())
		if err != nil {
			return
		}
		kind = KindPlaylistGroup
		return
	}

	// - https://www.youtube.com/user/fxigr1
	// - https://www.youtube.com/@fxigr1
	if strings.HasPrefix(path, "/user") || strings.HasPrefix(path, "/@") {
		id, err = ParseWebpageChannelID(parsed.String())
		if err != nil {
			return
		}
		kind = KindChannel
		return
	}
	err = errors.New("unsupported link format")
	return
}

var (
	channelRegexp = regexp.MustCompile(`href="https://www.youtube.com/channel/([^"]+)"`)
)

func ParseWebpageChannelID(u string) (string, error) {
	var resp *http.Response
	resp, err := http.Get(u)
	if err != nil {
		return "", err
	}
	html, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if i := strings.Index(string(html), `rel="canonical"`); i != -1 {
		gs := channelRegexp.FindStringSubmatch(string(html[i:]))
		if len(gs) <= 1 {
			return "", errors.New("failed to parse channel id from user page")
		}
		return gs[1], nil
	} else {
		return "", errors.New("failed to parse channel id from user page, no canonical link found")
	}
}
