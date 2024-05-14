package downloader

var _proxy string

func SetProxy(p string) {
	_proxy = p
}

func Proxy() string {
	return _proxy
}
