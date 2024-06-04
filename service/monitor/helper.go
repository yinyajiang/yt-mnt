package monitor

import (
	"fmt"

	"github.com/yinyajiang/yt-mnt/pkg/common"
	"github.com/yinyajiang/yt-mnt/pkg/ies"
)

func selectQualityFormatByResolution(formats []*ies.Format, resolution string) (index int) {
	if len(formats) == 0 {
		return -1
	}
	if resolution == "best" {
		for i, f := range formats {
			if f.FormatType == ies.FormatTypeComplete {
				return i
			}
		}
	}
	if resolution == "worst" {
		for i := len(formats) - 1; i >= 0; i-- {
			if formats[i].FormatType == ies.FormatTypeComplete {
				return i
			}
		}
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

func selectAudioFormatByResolution(formats []*ies.Format, resolution string) (index int) {
	_ = resolution
	if len(formats) == 0 {
		return -1
	}
	for i, f := range formats {
		if f.FormatType == ies.FormatTypeAudio {
			return i
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
		if len(item.Entries) == 0 {
			*outAll = append(*outAll, item)
		} else {
			plainAppend(outAll, item.Entries)
		}
	}
}
