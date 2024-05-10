package common

import (
	"net/url"
	"path/filepath"
)

func URLDotExt(u string) string {
	info, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return filepath.Ext(info.Path)
}
