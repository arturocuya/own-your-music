package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// search for the song directly. find if title, artist and album matches.
// matches: 163 / 1130
func findSongInBandcampV1(track *SpotifySong) {
	fmt.Printf("checking #%d: %s by %s from %s\n", track.Index, track.Name, track.Artist, track.Album)
	c := colly.NewCollector(
		colly.AllowedDomains("bandcamp.com"),
	)

	c.OnHTML(".results", func(e *colly.HTMLElement) {
		e.ForEachWithBreak(".searchresult", func(i int, h *colly.HTMLElement) bool {
			itemType := h.ChildText(".result-info .itemtype")

			if itemType != "TRACK" {
				return true
			}

			songName := h.ChildText(".result-info .heading")

			if songName != track.Name {
				return true
			}

			subheading := strings.ToLower(h.ChildText(".result-info .subhead"))
			if strings.Contains(subheading, strings.ToLower(track.Artist)) && strings.Contains(subheading, strings.ToLower(track.Album)) {
				url := h.ChildText(".result-info .itemurl")

				fmt.Printf("Match found! %s / %s : %s\n", songName, subheading, url)
				return false
			} else {
				return true
			}
		})
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("colly on error: ", err, r.Headers.Get("Retry-After"))

		if err.Error() == "Too Many Requests" {
			time.Sleep(3 * time.Minute)
		}
	})

	c.Visit(fmt.Sprintf(
		"https://bandcamp.com/search?q=%s&item_type=t&from=results",
		url.QueryEscape(track.Name),
	))

	c.Wait()
}

// search for album. check if name and artist matches. enter album. check if song name matches.
func findSongInBandcampV2(track *SpotifySong) {
	// TODO
}
