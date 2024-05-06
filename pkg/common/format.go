package common

import (
	"fmt"
	"github.com/yinyajiang/yt-mnt/model"
	"net/url"
	"path/filepath"

	"github.com/duke-git/lancet/v2/slice"
)

func SortFormats(formats []*model.Format) {
	slice.SortBy(formats, func(a, b *model.Format) bool {
		ar, _ := ParseResolutionInfo(fmt.Sprintf("%dx%d", a.Width, a.Height))
		br, _ := ParseResolutionInfo(fmt.Sprintf("%dx%d", b.Width, b.Height))
		return ar.ResolutionNum > br.ResolutionNum
	})
}

func SelectFormat(formats []*model.Format, resolution string) (index int) {
	r, _ := ParseResolutionInfo(resolution)
	for i, f := range formats {
		fr, _ := ParseResolutionInfo(fmt.Sprintf("%dx%d", f.Width, f.Height))
		if fr.ResolutionNum >= r.ResolutionNum {
			index = i
		} else {
			break
		}
	}
	return
}

func URLDotExt(u string) string {
	info, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return filepath.Ext(info.Path)
}
