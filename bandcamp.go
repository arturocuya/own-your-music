package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

type BandcampMatch struct {
	Name       string
	Subheading string
	SongUrl    string
	AlbumUrl   string
}

// search for album. check if name and artist matches. enter album. check if song name matches.
// matches: 271 / 1130
func findSongInBandcamp(track *SpotifySong) *BandcampMatch {
	fmt.Printf("v2: checking #%d: %s by %s from %s\n", track.Index, track.Name, track.Artist, track.Album)
	c := colly.NewCollector(
		colly.AllowedDomains("bandcamp.com"),
	)

	var match *BandcampMatch

	c.OnHTML(".results", func(e *colly.HTMLElement) {
		e.ForEachWithBreak(".searchresult", func(i int, h *colly.HTMLElement) bool {
			itemType := h.ChildText(".result-info .itemtype")

			if itemType != "ALBUM" {
				return true
			}

			albumName := h.ChildText(".result-info .heading")

			if !strings.Contains(sanitizeForComparison(albumName), sanitizeForComparison(track.Album)) {
				return true
			}

			subheading := h.ChildText(".result-info .subhead")

			// example subheading: "by Digitalism"
			if !strings.Contains(sanitizeForComparison(subheading), sanitizeForComparison(track.Artist)) {
				return true
			}

			albumUrl := h.ChildAttr(".result-info .heading a", "href")

			matchChannel := make(chan *BandcampMatch)
			go findSongInAlbumPage(track, albumUrl, matchChannel)
			match = <-matchChannel

			if match != nil {
				match.Subheading = subheading
				if strings.Contains(albumUrl, "?") {
					albumUrl = strings.Split(albumUrl, "?")[0]
				}
				match.AlbumUrl = albumUrl
			}

			return match == nil
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

	return match
}

func findSongInAlbumPage(track *SpotifySong, albumPageUrl string, matchChannel chan *BandcampMatch) {
	c := colly.NewCollector()

	var match *BandcampMatch

	c.OnScraped(func(r *colly.Response) {
		matchChannel <- match
		close(matchChannel)
	})

	c.OnHTML(".track_table", func(table *colly.HTMLElement) {
		table.ForEachWithBreak(".track_row_view", func(_ int, trackRow *colly.HTMLElement) bool {
			title := trackRow.ChildText(".track-title")

			if strings.Contains(sanitizeForComparison(title), sanitizeForComparison(track.Name)) {
				path := trackRow.ChildAttr(".title a", "href")
				match = &BandcampMatch{
					Name:    title,
					SongUrl: fmt.Sprintf("%s%s", getBaseURL(albumPageUrl), path),
				}
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

		matchChannel <- match
		close(matchChannel)
	})

	c.Visit(albumPageUrl)

	c.Wait()
}
