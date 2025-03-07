package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
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

// search for album. check if name and artist matches. enter album. check if song name matches.
func findSongInBandcamp(track *InputTrack) *PurchaseableTrack {
	fmt.Printf("checking #%d: %s by %s from %s\n", track.Idx, track.Name, track.Artist, track.Album)
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

	searchCollector.Visit(fmt.Sprintf(
		"https://bandcamp.com/search?q=%s&item_type=a&from=results",
		url.QueryEscape(track.Album),
	))

	searchCollector.Wait()

	// visit each album page to confirm artist

	albumMatch := ""

	fmt.Printf("searching in %d possible album matches\n", len(possibleAlbumMatches))

	for _, albumUrl := range possibleAlbumMatches {
		fmt.Println("trying possible album match", albumUrl)

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
				fmt.Println("found album match", albumUrl)
				albumMatch = albumUrl
			} else {
				fmt.Printf("album artist '%s' did not match '%s'\n", artistName, track.Artist)
			}
		})

		albumCollector.Visit(albumUrl)
		albumCollector.Wait()

		if albumMatch != "" {
			break
		}
	}

	if albumMatch != "" {
		match = findSongInAlbumPage(track, albumMatch)
	}

	if match == nil {
		fmt.Println("could not find song by album. will try to find by track...")

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

		searchCollector.Visit(fmt.Sprintf(
			"https://bandcamp.com/search?q=%s&item_type=t&from=results",
			url.QueryEscape(track.Name),
		))

		searchCollector.Wait()

		fmt.Printf("searching %d possible track matches\n", len(possibleTrackMatches))

		for _, url := range possibleTrackMatches {
			fmt.Println("trying possible track match", url)
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
					fmt.Printf("track artist '%s' did not match '%s'\n", artistName, track.Artist)
				}
			})
			searchCollector.Visit(url)
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

	fmt.Printf("found track match: '%s' at url %s \n", match.Name, match.SongUrl)

	// now let's find the song details like price

	detailsCollector := getNewBandcampCollector()

	detailsCollector.OnHTML(".buyItem.digital", func(e *colly.HTMLElement) {
		priceText := e.ChildText("li.buyItem.digital > .ft > .ft.main-button > span > span.base-text-color")

		fmt.Printf("price found: '%v'\n", priceText)

		if priceText != "" {
			currencyText := e.ChildText(".ft:nth-child(1) > span:nth-child(2) > span:nth-child(2)")

			fmt.Println("currencty found", currencyText)

			match.RawPrice = priceText

			price, err := parseBandcampPrice(priceText, currencyText)

			if err == nil {
				match.Price = price
			} else {
				fmt.Println("could not parse price. details:", err)
			}
		} else {
			nameYourPriceText := e.ChildText(".buyItemExtra.buyItemNyp.secondaryText")

			if nameYourPriceText == "name your price" {
				match.RawPrice = nameYourPriceText
				match.Price = money.New(0, money.USD)
			}
		}
	})

	fmt.Println("will search for price...")
	detailsCollector.Visit(match.SongUrl)
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

	c.Visit(albumPageUrl)

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
			fmt.Println("colly on error: ", err, r.Headers.Get("Retry-After"))
		}

		if err.Error() == "Too Many Requests" {
			// TODO: not sure if it's ok to sleep in the coroutine
			// but i guess it's ok as long as i don't send to the channel
			// before the timer ends
			time.Sleep(30 * time.Second)
		}
	})

	return c
}
