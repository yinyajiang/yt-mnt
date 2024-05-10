package ies

import (
	"fmt"

	"github.com/yinyajiang/yt-mnt/pkg/common"

	"github.com/duke-git/lancet/v2/slice"
)

func Options(options []ParseOptions) ParseOptions {
	if len(options) == 0 {
		return ParseOptions{}
	}
	return options[0]
}

func sortEntryFormats(entry *MediaEntry) {
	if len(entry.Formats) != 0 {
		sortFormats(entry.Formats)
	}
	for _, entry := range entry.Entries {
		sortEntryFormats(entry)
	}
}

func sortFormats(formats []*Format) {
	if len(formats) == 0 {
		return
	}
	slice.SortBy(formats, func(a, b *Format) bool {
		ar, _ := common.ParseResolutionInfo(fmt.Sprintf("%dx%d", a.Width, a.Height))
		br, _ := common.ParseResolutionInfo(fmt.Sprintf("%dx%d", b.Width, b.Height))
		return ar.ResolutionNum > br.ResolutionNum
	})
}
