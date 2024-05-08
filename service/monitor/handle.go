package monitor

import (
	"github.com/yinyajiang/yt-mnt/model"
	"github.com/yinyajiang/yt-mnt/pkg/ies"
)

type ExploreHandle struct {
	ie          string
	first       model.MediaEntry
	nextPage    ies.NextPage
	linkInfo    ies.LinkInfo
	BeforeCount int64
}

func (e *ExploreHandle) IsValid() bool {
	return (e.ie != "") &&
		(e.linkInfo.MediaID != "" || e.linkInfo.LinkID != "") &&
		(e.linkInfo.MediaType == model.MediaTypePlaylist ||
			e.linkInfo.MediaType == model.MediaTypePlaylistGroup ||
			e.linkInfo.MediaType == model.MediaTypeUser)
}

func (e *ExploreHandle) IsEnd() bool {
	return e.nextPage.IsEnd
}
