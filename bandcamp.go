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
	fmt.Printf("v1: checking #%d: %s by %s from %s\n", track.Index, track.Name, track.Artist, track.Album)
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
// matches: 266 / 1130
func findSongInBandcampV2(track *SpotifySong) {
	fmt.Printf("v2: checking #%d: %s by %s from %s\n", track.Index, track.Name, track.Artist, track.Album)
	c := colly.NewCollector(
		colly.AllowedDomains("bandcamp.com"),
	)

	c.OnHTML(".results", func(e *colly.HTMLElement) {
		e.ForEachWithBreak(".searchresult", func(i int, h *colly.HTMLElement) bool {
			itemType := h.ChildText(".result-info .itemtype")

			if itemType != "ALBUM" {
				return true
			}

			albumName := strings.ToLower(h.ChildText(".result-info .heading"))

			if !strings.Contains(albumName, strings.ToLower(track.Album)) {
				return true
			}

			subheading := strings.ToLower(h.ChildText(".result-info .subhead"))

			// example subheading: "by Digitalism"
			if !strings.Contains(subheading, strings.ToLower(track.Artist)) {
				return true
			}

			albumUrl := h.ChildAttr(".result-info .heading a", "href")

			ch := make(chan bool)
			go findSongInAlbumPage(track, albumUrl, ch)
			value := <-ch

			return value
		})
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("colly on error: ", err, r.Headers.Get("Retry-After"))

		if err.Error() == "Too Many Requests" {
			time.Sleep(3 * time.Minute)
		}
	})

	c.Visit(fmt.Sprintf(
		"https://bandcamp.com/search?q=%s&item_type=a&from=results",
		url.QueryEscape(track.Album),
	))

	c.Wait()
}

func findSongInAlbumPage(track *SpotifySong, albumPageUrl string, returnCh chan bool) {
	c := colly.NewCollector()
	found := false

	c.OnScraped(func(r *colly.Response) {
		returnCh <- !found
		close(returnCh)
	})

	c.OnHTML(".track_table", func(table *colly.HTMLElement) {
		table.ForEachWithBreak(".track_row_view", func(_ int, trackRow *colly.HTMLElement) bool {
			title := strings.ToLower(trackRow.ChildText(".track-title"))

			if strings.Contains(title, strings.ToLower(track.Name)) {
				path := trackRow.ChildAttr(".title a", "href")
				fmt.Printf("\tMatch found! %s : %s\n", title, fmt.Sprintf("%s%s", getBaseURL(albumPageUrl), path))
				found = true
				return false
			}

			return true
		})
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("colly on error: ", err, r.Headers.Get("Retry-After"))

		if err.Error() == "Too Many Requests" {
			// TODO: not sure if it's ok to sleep in the coroutine
			// but i guess it's ok as long as i don't send to the channel
			// before the timer ends
			time.Sleep(3 * time.Minute)
		}

		returnCh <- !found
		close(returnCh)
	})

	c.Visit(albumPageUrl)

	c.Wait()
}
