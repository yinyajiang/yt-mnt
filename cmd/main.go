package main

import (
	"context"
	"github.com/yinyajiang/yt-mnt/service/monitor"

	"log"
)

func main() {
	m := monitor.NewMonitor("/Users/new/Documents/GitHub/yt-mnt/db3.sqlite3", true)
	// feed, err := m.Subscribe("https://www.instagram.com/ll5161551615/")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Println("Subscribed to feed: ", feed.ID)
	err := m.DownloadEntry(context.Background(), 2, func(total, downloaded, speed int64, percent float64) {
		log.Println("Downloaded: ", downloaded, "Total: ", total, "Speed: ", speed, "Percent: ", percent)
	})
	if err != nil {
		log.Fatal(err)
	}
}
