package main

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/gocolly/colly/v2"
	"golang.org/x/text/currency"
)

type PurchaseableTrack struct {
	SongIdx    int
	Name       string
	Subheading string
	SongUrl    string
	AlbumUrl   string
	RawPrice   string
	Price      *money.Money
}

var (
	eepyTime = sync.NewCond(&sync.Mutex{})
	isEep    = false
)

// pauseRequests sets the paused flag and, after the specified duration,
// clears it and signals waiting workers.
func pauseRequests(duration time.Duration) {
	eepyTime.L.Lock()
	if !isEep {
		isEep = true
		eepyTime.L.Unlock()

		go func() {
			log.Println("eepying for", duration)
			time.Sleep(duration)
			log.Println("time to awek")
			eepyTime.L.Lock()
			isEep = false
			eepyTime.Broadcast() // signal all waiting workers
			eepyTime.L.Unlock()
		}()
	} else {
		eepyTime.L.Unlock()
	}
}

// search for album. check if name and artist matches. enter album. check if song name matches.
func findSongInBandcamp(track *InputTrack) *PurchaseableTrack {
	log.Printf("checking #%d: %s by %s from %s\n", track.Idx, track.Name, track.Artist, track.Album)
	searchCollector := getNewBandcampCollector()

	var match *PurchaseableTrack

	var possibleAlbumMatches []string

	searchCollector.OnHTML(".results", func(e *colly.HTMLElement) {
		e.ForEachWithBreak(".searchresult", func(i int, h *colly.HTMLElement) bool {
			itemType := h.ChildText(".result-info .itemtype")

			if itemType != "ALBUM" {
				return true
			}

			albumName := h.ChildText(".result-info .heading")

			if !musicItemEquals(track.Album, albumName) {
				return true
			}

			subheading := h.ChildText(".result-info .subhead")

			// example subheading: "by Digitalism"
			if !strings.Contains(sanitizeForComparison(subheading), sanitizeForComparison(track.Artist)) {
				return true
			}

			albumUrl := h.ChildAttr(".result-info .heading a", "href")

			possibleAlbumMatches = append(possibleAlbumMatches, strings.Split(albumUrl, "?")[0])

			return true
		})
	})

	visitPage(searchCollector, fmt.Sprintf(
		"https://bandcamp.com/search?q=%s&item_type=a&from=results",
		url.QueryEscape(track.Album),
	))

	searchCollector.Wait()

	// visit each album page to confirm artist

	albumMatch := ""

	log.Printf("searching in %d possible album matches\n", len(possibleAlbumMatches))

	for _, albumUrl := range possibleAlbumMatches {
		log.Println("trying possible album match", albumUrl)

		albumCollector := getNewBandcampCollector()

		albumCollector.OnHTML("#bio-container", func(e *colly.HTMLElement) {
			artistName := e.ChildText("#band-name-location > span:nth-child(1)")

			isMatch := false

			if containsEastAsianCharacters(track.Artist) {
				isMatch = musicItemEquals(track.Artist, artistName)
			} else {
				isMatch = sanitizeForComparison(track.Artist) == sanitizeForComparison(artistName)
			}

			if isMatch {
				log.Println("found album match", albumUrl)
				albumMatch = albumUrl
			} else {
				log.Printf("album artist '%s' did not match '%s'\n", artistName, track.Artist)
			}
		})

		visitPage(albumCollector, albumUrl)

		albumCollector.Wait()

		if albumMatch != "" {
			break
		}
	}

	if albumMatch != "" {
		match = findSongInAlbumPage(track, albumMatch)
	}

	if match == nil {
		log.Println("could not find song by album. will try to find by track...")

		var possibleTrackMatches []string

		// song could not be found by album. find it by track name
		searchCollector.OnHTML(".results", func(e *colly.HTMLElement) {
			e.ForEachWithBreak(".searchresult", func(i int, h *colly.HTMLElement) bool {
				itemType := h.ChildText(".result-info .itemtype")

				if itemType != "TRACK" {
					return true
				}

				songName := h.ChildText(".result-info .heading")

				if !musicItemEquals(track.Name, songName) {
					return true
				}

				// TODO: make this configurable
				alternativeKeywords := []string{"remix", "clean]", "clean)", "edit]", "edit)", "mashup"}

				for _, altKeyword := range alternativeKeywords {
					if !strings.Contains(songName, altKeyword) && strings.Contains(sanitizeForComparison(songName), sanitizeForComparison(altKeyword)) {
						return true
					}
				}

				subheading := strings.ToLower(h.ChildText(".result-info .subhead"))

				if strings.Contains(sanitizeForComparison(subheading), sanitizeForComparison(track.Artist)) {
					songUrl := h.ChildText(".result-info .itemurl")
					possibleTrackMatches = append(possibleTrackMatches, strings.Split(songUrl, "?")[0])

					return false
				} else {
					return true
				}
			})
		})

		visitPage(searchCollector, fmt.Sprintf(
			"https://bandcamp.com/search?q=%s&item_type=t&from=results",
			url.QueryEscape(track.Name),
		))

		searchCollector.Wait()

		log.Printf("searching %d possible track matches\n", len(possibleTrackMatches))

		for _, url := range possibleTrackMatches {
			log.Println("trying possible track match", url)
			searchCollector.OnHTML("#bio-container", func(e *colly.HTMLElement) {
				artistName := e.ChildText("#band-name-location > span:nth-child(1)")

				isMatch := false

				if containsEastAsianCharacters(track.Artist) {
					isMatch = musicItemEquals(track.Artist, artistName)
				} else {
					isMatch = sanitizeForComparison(track.Artist) == sanitizeForComparison(artistName)
				}

				if isMatch {
					songName := e.ChildText("h2.trackTitle")
					match = &PurchaseableTrack{
						Name:    songName,
						SongUrl: strings.Split(url, "?")[0],
					}
				} else {
					log.Printf("track artist '%s' did not match '%s'\n", artistName, track.Artist)
				}
			})

			visitPage(searchCollector, url)

			searchCollector.Wait()

			if match != nil {
				break
			}
		}
	}

	if match == nil {
		return nil
	}

	match.SongIdx = track.Idx

	log.Printf("found track match: '%s' at url %s \n", match.Name, match.SongUrl)

	// now let's find the song details like price

	detailsCollector := getNewBandcampCollector()

	detailsCollector.OnHTML(".buyItem.digital", func(e *colly.HTMLElement) {
		priceText := e.ChildText("li.buyItem.digital > .ft > .ft.main-button > span > span.base-text-color")

		log.Printf("price found: '%v'\n", priceText)

		if priceText != "" {
			currencyText := e.ChildText(".ft:nth-child(1) > span:nth-child(2) > span:nth-child(2)")

			log.Println("currencty found", currencyText)

			match.RawPrice = priceText

			price, err := parseBandcampPrice(priceText, currencyText)

			if err == nil {
				match.Price = price
			} else {
				log.Println("could not parse price. details:", err)
			}
		} else {
			nameYourPriceText := e.ChildText(".buyItemExtra.buyItemNyp.secondaryText")

			if nameYourPriceText == "name your price" {
				match.RawPrice = nameYourPriceText
				match.Price = money.New(0, money.USD)
			}
		}
	})

	log.Println("will search for price...")

	visitPage(detailsCollector, match.SongUrl)

	detailsCollector.Wait()

	return match
}

func findSongInAlbumPage(track *InputTrack, albumPageUrl string) *PurchaseableTrack {
	c := getNewBandcampCollector()

	var match *PurchaseableTrack

	c.OnHTML(".track_table", func(table *colly.HTMLElement) {
		table.ForEachWithBreak(".track_row_view", func(_ int, trackRow *colly.HTMLElement) bool {
			title := trackRow.ChildText(".track-title")

			if musicItemEquals(track.Name, title) {
				path := trackRow.ChildAttr(".title a", "href")
				match = &PurchaseableTrack{
					Name:    title,
					SongUrl: fmt.Sprintf("%s%s", getBaseURL(albumPageUrl), path),
				}
				return false
			}

			return true
		})
	})

	visitPage(c, albumPageUrl)

	c.Wait()

	return match
}

// reference: https://get.bandcamp.help/hc/en-us/articles/23020726236823-Which-currencies-does-Bandcamp-support
func parseBandcampPrice(rawPrice string, currencyCode string) (*money.Money, error) {
	_, err := currency.ParseISO(currencyCode)

	if err != nil {
		return nil, err
	}

	priceNumber := removeNonNumericPrefixSuffix(rawPrice)
	splitted := strings.Split(priceNumber, ".")

	units, err := strconv.Atoi(splitted[0])

	if err != nil {
		return nil, err
	}

	amount := int64(units * 100)

	if len(splitted) > 1 {
		cents, err := strconv.Atoi(splitted[1])

		if err != nil {
			return nil, err
		}

		amount += int64(cents)
	}

	return money.New(amount, currencyCode), nil
}

func getNewBandcampCollector() *colly.Collector {
	c := colly.NewCollector()

	c.OnError(func(r *colly.Response, err error) {
		if r != nil && r.Headers != nil {
			log.Println("colly on error: ", err, r.Headers.Get("Retry-After"))
		}

		if err.Error() == "Too Many Requests" {
			// TODO: not sure if it's ok to sleep in the coroutine
			// but i guess it's ok as long as i don't send to the channel
			// before the timer ends
			log.Println("got too many requests. eepy time...")
			pauseRequests(3 * time.Minute)
		}
	})

	return c
}

func visitPage(c *colly.Collector, url string) {
	// Check if a pause is in effect. If so, wait until it's cleared.
	eepyTime.L.Lock()
	for isEep {
		eepyTime.Wait()
	}
	eepyTime.L.Unlock()
	c.Visit(url)

	// experimentally i found that aprox. 110 requests can be made to bandcamp
	// before we hit "too many requests" errors, so if we wait 0.545 seconds between
	// requests, we wouldn't hit the limit. 650ms has a bit more margin just to be sure.
	pauseRequests(650 * time.Millisecond)
}
