package main

import (
	"fmt"
	"log"

	"github.com/yinyajiang/yt-mnt/service/monitor"
)

func main() {
	m := monitor.NewMonitor("/Users/new/Documents/GitHub/yt-mnt/db3.sqlite3", true)
	// feed, err := m.Subscribe("https://www.instagram.com/ll5161551615/")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Println("Subscribed to feed: ", feed.ID)
	first, handle, err := m.ExploreFirst("https://www.youtube.com/@huber0203")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(first.Entries)
	for !handle.IsEnd() {
		next, err := m.ExploreNext(handle)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(next.Entries)
	}
}
