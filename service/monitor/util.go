package monitor

import (
	"errors"

	"github.com/yinyajiang/yt-mnt/pkg/ies/instagram"
	"github.com/yinyajiang/yt-mnt/pkg/ies/youtube"
)

func SiteToURL(site, usr, playlist string) (url string, err error) {
	switch site {
	case "youtube":
		return youtube.GenYoutubeURL("", usr, playlist)
	case "instagram":
		return instagram.GenInstagramURL(usr)
	default:
		err = errors.New("not support website")
	}
	return
}
