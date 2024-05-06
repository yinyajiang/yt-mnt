package common

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/convertor"
)

type ResolutionInfo struct {
	Resolution    string
	ResolutionNum int64
	FPS           int64
	W             int64
	H             int64
}

var regexpWH = regexp.MustCompile(`(\d+)[xX](\d+)`)
var regexpPFPS = regexp.MustCompile(`(\d+)[piIP](\d*)`)

func ParseResolutionInfo(s string) (ret ResolutionInfo, ok bool) {
	if s == "" {
		return
	}

	if matchs := regexpPFPS.FindStringSubmatch(s); len(matchs) >= 2 {
		// 1080p、1080p60

		ret.Resolution = matchs[0]
		if len(matchs) >= 2 && matchs[1] != "" {
			ret.ResolutionNum, _ = convertor.ToInt(matchs[1])
		}
		if len(matchs) == 3 && matchs[2] != "" {
			ret.FPS, _ = convertor.ToInt(matchs[2])
		}
	} else if wh := regexpWH.FindStringSubmatch(s); len(wh) == 3 {
		//1920*1080

		ret.W, _ = strconv.ParseInt(wh[1], 0, 64)
		ret.H, _ = strconv.ParseInt(wh[2], 0, 64)
		ret.ResolutionNum = WHToP(ret.W, ret.H)
		if ret.ResolutionNum == 0 {
			return
		}
		ret.Resolution = fmt.Sprintf("%dp", ret.ResolutionNum)
	}

	if ret.ResolutionNum == 0 {
		switch strings.ToLower(ret.Resolution) {
		case "uhd":
			ret.ResolutionNum = 2160
		case "qhd":
			ret.ResolutionNum = 1440
		case "fhd":
			ret.ResolutionNum = 1080
		case "hd":
			ret.ResolutionNum = 720
		case "sd":
			ret.ResolutionNum = 480
		case "ld":
			ret.ResolutionNum = 360
		default:
			return ret, false
		}
	}
	if (ret.W == 0 || ret.H == 0) && ret.ResolutionNum != 0 {
		ret.W, ret.H = PToWH(ret.ResolutionNum)
	}
	ok = true
	return
}

func PToWH(p int64) (w, h int64) {
	if p <= 0 {
		return
	}
	switch {
	case p <= 144:
		w, h = 192, 144
	case p <= 240:
		w, h = 320, 240
	case p <= 360:
		w, h = 360, 480
	case p <= 480:
		w, h = 640, 480
	case p <= 540:
		w, h = 960, 540
	case p <= 720:
		w, h = 1280, 720
	case p <= 1080:
		w, h = 1920, 1080
	case p <= 1440:
		w, h = 2560, 1440
	case p <= 2160:
		w, h = 4096, 2160
	case p <= 4320:
		w, h = 7680, 4320
	default:
		w, h = p, p
	}
	return
}

func WHToP(w, h int64) int64 {
	if h == 0 {
		return 0
	}

	p1 := int64(144)
	if w*h >= 4840*2160 {
		p1 = 4320
	} else if w*h >= 3840*1920 {
		p1 = 2160
	} else if w*h >= 2560*1280 {
		p1 = 1440
	} else if w*h >= 1920*960 {
		p1 = 1080
	} else if w*h >= 1000*570 {
		p1 = 720
	} else if w*h >= 640*320 {
		p1 = 480
	} else if w*h >= 512*288 {
		p1 = 380
	} else if w*h >= 426*214 {
		p1 = 360
	} else if w*h >= 320*240 {
		p1 = 240
	}

	if w < h {
		h = w
	}

	p2 := int64(144)
	if h >= 4000 {
		p2 = 4320
	} else if h >= 2000 {
		p2 = 2160
	} else if h >= 1440 {
		p2 = 1440
	} else if h >= 1080 {
		p2 = 1080
	} else if h >= 720 {
		p2 = 720
	} else if h >= 480 {
		p2 = 480
	} else if h >= 360 {
		p2 = 360
	} else if h >= 240 {
		p2 = 240
	}

	if w == 0 && p1 == 144 && p2 == 144 {
		return 0
	}

	if p1 >= p2 {
		return p1
	}
	return p2
}
