package common

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

func URLDotExt(u string) string {
	info, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return filepath.Ext(info.Path)
}

var reWrongFileChars = regexp.MustCompile(`[\x{1}-\x{6}\x{e}-\x{19}\x{1b}-\x{1f}"<>\|\a\t\n\v\f\r\:\*\?\\\/]`)

func ReplaceWrongFileChars(stem string) string {
	stem = strings.ReplaceAll(strings.ReplaceAll(stem, "\\", "_"), "/", "_")
	return reWrongFileChars.ReplaceAllString(stem, "_")
}
