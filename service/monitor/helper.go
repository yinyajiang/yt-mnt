package monitor

import (
	"fmt"

	"github.com/yinyajiang/yt-mnt/pkg/common"
	"github.com/yinyajiang/yt-mnt/pkg/ies"
)

func selectFormatByResolution(formats []*ies.Format, resolution string) (index int) {
	if len(formats) == 0 {
		return -1
	}
	if resolution == "best" {
		return 0
	}
	if resolution == "worst" {
		return len(formats) - 1
	}

	r, _ := common.ParseResolutionInfo(resolution)
	for i, f := range formats {
		fr, _ := common.ParseResolutionInfo(fmt.Sprintf("%dx%d", f.Width, f.Height))
		if fr.ResolutionNum >= r.ResolutionNum {
			index = i
		} else {
			break
		}
	}
	return
}

func plain(items []*ies.MediaEntry) []*ies.MediaEntry {
	outAll := make([]*ies.MediaEntry, 0, len(items))
	plainAppend(&outAll, items)
	return outAll
}

func plainAppend(outAll *[]*ies.MediaEntry, items []*ies.MediaEntry) {
	if outAll == nil {
		return
	}
	if *outAll == nil {
		*outAll = make([]*ies.MediaEntry, 0, len(items))
	}
	for _, item := range items {
		if item.MediaType == ies.MediaTypeVideo ||
			item.MediaType == ies.MediaTypeAudio ||
			item.MediaType == ies.MediaTypeImage {
			*outAll = append(*outAll, item)
		} else {
			plainAppend(outAll, item.Entries)
		}
	}
}
