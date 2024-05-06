package insapi

import (
	"io"
	"net/http"
	"regexp"
)

var _userNameMapID = map[string]string{}

func UserName2ID(username string) string {
	if id, ok := _userNameMapID[username]; ok {
		return id
	}

	rsp, err := http.Get("https://www.instagram.com/" + username)
	if err != nil {
		return ""
	}
	defer rsp.Body.Close()
	by, err := io.ReadAll(rsp.Body)
	if err != nil {
		return ""
	}
	reID := regexp.MustCompile(`"id":\s*"([0-9]+)"`)
	submatchs := reID.FindStringSubmatch(string(by))
	if len(submatchs) == 2 {
		_userNameMapID[username] = submatchs[1]
		return submatchs[1]
	}
	return ""
}
