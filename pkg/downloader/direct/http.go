package direct

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/yinyajiang/yt-mnt/pkg/downloader"
)

var _defclient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func client() *http.Client {
	if downloader.Proxy() == "" {
		return _defclient
	}
	proxyUrl, err := url.Parse(downloader.Proxy())
	if err != nil {
		return _defclient
	}
	return &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(proxyUrl),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
}

func urlSize(ctx context.Context, url string) int64 {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0
	}
	req = req.WithContext(ctx)

	resp, err := client().Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var total int64
	if len(resp.Header.Values("Content-Length")) > 0 {
		total, _ = strconv.ParseInt(resp.Header.Values("Content-Length")[0], 10, 64)
	}
	return total
}

func downloadW(ctx context.Context, url string, w io.Writer, total int64, sink downloader.ProgressSink) (err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req = req.WithContext(ctx)

	resp, err := client().Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if total <= 0 {
		if len(resp.Header.Values("Content-Length")) > 0 {
			total, _ = strconv.ParseInt(resp.Header.Values("Content-Length")[0], 10, 64)
		}
	}

	defer resp.Body.Close()

	var (
		downloaded     int64
		lastTime       = time.Now()
		lastDownloaded int64
		buf            = make([]byte, 32*1024) // 32KB buffer
	)

loop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			nRead, err := resp.Body.Read(buf)
			if err != nil && err != io.EOF {
				return err
			}
			if nRead == 0 {
				break loop
			}

			_, err = w.Write(buf[:nRead])
			if err != nil {
				return err
			}

			downloaded += int64(nRead)
			if sink == nil {
				continue
			}

			now := time.Now()
			elapsed := now.Sub(lastTime).Seconds()
			if elapsed >= 1 {
				speed := float64(downloaded-lastDownloaded) / elapsed
				lastTime = now
				lastDownloaded = downloaded
				percent := float64(0)
				eta := int64(0)
				if total > 0 {
					percent = float64(downloaded) / float64(total) * 100
					if speed > 0 {
						eta = int64(float64(total-downloaded) / speed)
					}
				}
				sink(total, downloaded, int64(speed), eta, percent, 0)
			}
		}
	}
	if sink != nil {
		sink(total, downloaded, 0, 0, 100, 0)
	}
	return nil
}

func downloadFile(ctx context.Context, url, path string, avTotal int64, sink downloader.ProgressSink) (err error) {
	os.MkdirAll(filepath.Dir(path), os.ModePerm)
	downingPath := path + ".downing"
	f, err := os.Create(downingPath)
	if err != nil {
		return
	}
	err = downloadW(ctx, url, f, avTotal, sink)
	f.Close()
	if err == nil {
		os.Rename(downingPath, path)
	}
	return err
}
