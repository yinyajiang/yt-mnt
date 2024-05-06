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

var (
	channelRegexp = regexp.MustCompile(`href="https://www.youtube.com/channel/([^"]+)"`)
)

const (
	kindChannel = iota + 1
	kindPlaylist
)

func parseYoutubeURL(link string) (kind int, id string, err error) {
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
		kind = kindPlaylist
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
		kind = kindChannel
		id = parts[2]
		return
	}

	// - https://www.youtube.com/user/fxigr1
	// - https://www.youtube.com/@fxigr1
	if strings.HasPrefix(path, "/user") || strings.HasPrefix(path, "/@") {
		var resp *http.Response
		resp, err = http.Get(parsed.String())
		if err != nil {
			err = errors.Wrapf(err, "failed to get user page: %s", parsed.String())
			return
		}
		html, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if i := strings.Index(string(html), `rel="canonical"`); i != -1 {
			gs := channelRegexp.FindStringSubmatch(string(html[i:]))
			if len(gs) <= 1 {
				err = errors.New("failed to parse channel id from user page")
				return
			}
			id = gs[1]
		} else {
			err = errors.New("failed to parse channel id from user page, no canonical link found")
			return
		}
		kind = kindChannel
		return
	}
	err = errors.New("unsupported link format")
	return
}
