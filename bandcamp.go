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
// matches: 271 / 1130
func findSongInBandcamp(track *InputTrack) *PurchaseableTrack {
	fmt.Printf("checking #%d: %s by %s from %s\n", track.Idx, track.Name, track.Artist, track.Album)
	searchCollector := colly.NewCollector(
		colly.AllowedDomains("bandcamp.com"),
	)

	var match *PurchaseableTrack

	searchCollector.OnHTML(".results", func(e *colly.HTMLElement) {
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

			matchChannel := make(chan *PurchaseableTrack)
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

	searchCollector.OnError(func(r *colly.Response, err error) {
		if r != nil && r.Headers != nil {
			fmt.Println("colly on error: ", err, r.Headers.Get("Retry-After"))
		}

		if err.Error() == "Too Many Requests" {
			// TODO: retry same song after this
			time.Sleep(30 * time.Second)
		}
	})

	searchCollector.Visit(fmt.Sprintf(
		"https://bandcamp.com/search?q=%s&item_type=a&from=results",
		url.QueryEscape(track.Album),
	))

	searchCollector.Wait()

	if match == nil {
		return nil
	}

	match.SongIdx = track.Idx

	detailsCollector := colly.NewCollector()

	detailsCollector.OnHTML(".buyItem.digital", func(e *colly.HTMLElement) {
		priceText := e.ChildText(".ft:nth-child(1) > span:nth-child(2) > span:nth-child(1)")

		fmt.Printf("price found: '%v'\n", priceText)

		//span.buyItemNyp:nth-child(2)
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

	detailsCollector.OnError(func(r *colly.Response, err error) {
		if r != nil && r.Headers != nil {
			fmt.Println("colly on error: ", err, r.Headers.Get("Retry-After"))
		}

		if err.Error() == "Too Many Requests" {
			time.Sleep(30 * time.Second)
		}
	})

	fmt.Println("will search for price...")
	detailsCollector.Visit(match.SongUrl)
	detailsCollector.Wait()

	return match
}

func findSongInAlbumPage(track *InputTrack, albumPageUrl string, matchChannel chan *PurchaseableTrack) {
	c := colly.NewCollector()

	var match *PurchaseableTrack

	c.OnScraped(func(r *colly.Response) {
		matchChannel <- match
		close(matchChannel)
	})

	c.OnHTML(".track_table", func(table *colly.HTMLElement) {
		table.ForEachWithBreak(".track_row_view", func(_ int, trackRow *colly.HTMLElement) bool {
			title := trackRow.ChildText(".track-title")

			if strings.Contains(sanitizeForComparison(title), sanitizeForComparison(track.Name)) {
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

		matchChannel <- match
		close(matchChannel)
	})

	c.Visit(albumPageUrl)

	c.Wait()
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
