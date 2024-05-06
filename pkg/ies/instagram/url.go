package instagram

import (
	"errors"
	"regexp"
)

var (
	storyRegexp = regexp.MustCompile(`instagram\.com/stories/([^/]+)/?`)
	userRegexp  = regexp.MustCompile(`instagram\.com/([^/]+)/?`)
)

const (
	kindUser = iota
	kindStory
)

func parseInstagramURL(link string) (kind int, user string, err error) {
	matchs := storyRegexp.FindStringSubmatch(link)
	if len(matchs) == 2 {
		user = matchs[1]
		kind = kindStory
		return
	}

	matchs = userRegexp.FindStringSubmatch(link)
	if len(matchs) == 2 {
		user = matchs[1]
		kind = kindUser
	}
	if user == "p" {
		user = ""
	}
	if user == "" {
		err = errors.New("invalid Instagram URL")
	}
	return
}
