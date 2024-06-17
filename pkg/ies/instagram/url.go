package instagram

import (
	"errors"
	"regexp"
	"strings"
)

var (
	storyRegexp = regexp.MustCompile(`instagram\.com/stories/([^/]+)/?`)
	userRegexp  = regexp.MustCompile(`instagram\.com/([^/]+)/?`)
)

const (
	KindUser = iota
	KindStory
)

func GenInstagramURL(usr string) (url string, err error) {
	if usr != "" {
		if strings.HasPrefix(usr, "@") && len(usr) > 1 {
			usr = usr[1:]
		}
		url = "https://www.instagram.com/" + usr + "/"
	} else {
		err = errors.New("instagram user is required")
	}
	return
}

func IsInstragramURL(link string) bool {
	return strings.Contains(link, "instagram.com")
}

func ParseInstagramURL(link string) (kind int, user string, err error) {
	matchs := storyRegexp.FindStringSubmatch(link)
	if len(matchs) == 2 {
		user = matchs[1]
		kind = KindStory
		return
	}

	matchs = userRegexp.FindStringSubmatch(link)
	if len(matchs) == 2 {
		user = matchs[1]
		kind = KindUser
	}
	if user == "p" {
		user = ""
	}
	if user == "" {
		err = errors.New("invalid Instagram URL")
	}
	return
}
