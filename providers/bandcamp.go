package providers

import (
	"fmt"
	"log"
	"net/url"
	"ownyourmusic/types"
	"ownyourmusic/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/gocolly/colly/v2"
	"golang.org/x/text/currency"
)

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

type BandcampProvider struct{}

func (p BandcampProvider) GetProviderName() string {
	return BANDCAMP_PROVIDER
}

func (p BandcampProvider) FindSong(track *types.InputTrack) *types.PurchaseableTrack {
	log.Printf("checking: %s by %s from %s\n", track.Name, track.Artist, track.Album)
	searchCollector := p.getNewBandcampCollector()

	var match *types.PurchaseableTrack

	var possibleAlbumMatches []string

	searchCollector.OnHTML(".results", func(e *colly.HTMLElement) {
		e.ForEachWithBreak(".searchresult", func(i int, h *colly.HTMLElement) bool {
			itemType := h.ChildText(".result-info .itemtype")

			if itemType != "ALBUM" {
				return true
			}

			albumName := h.ChildText(".result-info .heading")

			if !utils.MusicItemEquals(track.Album, albumName) {
				return true
			}

			subheading := h.ChildText(".result-info .subhead")

			// example subheading: "by Digitalism"
			if !strings.Contains(utils.SanitizeForComparison(subheading), utils.SanitizeForComparison(track.Artist)) {
				return true
			}

			albumUrl := h.ChildAttr(".result-info .heading a", "href")

			possibleAlbumMatches = append(possibleAlbumMatches, strings.Split(albumUrl, "?")[0])

			return true
		})
	})

	p.visitPage(searchCollector, fmt.Sprintf(
		"https://bandcamp.com/search?q=%s&item_type=a&from=results",
		url.QueryEscape(track.Album),
	))

	searchCollector.Wait()

	// visit each album page to confirm artist

	albumMatch := ""

	log.Printf("searching in %d possible album matches\n", len(possibleAlbumMatches))

	for _, albumUrl := range possibleAlbumMatches {
		log.Println("trying possible album match", albumUrl)

		albumCollector := p.getNewBandcampCollector()

		albumCollector.OnHTML("#bio-container", func(e *colly.HTMLElement) {
			artistName := e.ChildText("#band-name-location > span:nth-child(1)")

			isMatch := false

			if utils.ContainsEastAsianCharacters(track.Artist) {
				isMatch = utils.MusicItemEquals(track.Artist, artistName)
			} else {
				isMatch = utils.SanitizeForComparison(track.Artist) == utils.SanitizeForComparison(artistName)
			}

			if isMatch {
				log.Println("found album match", albumUrl)
				albumMatch = albumUrl
			} else {
				log.Printf("album artist '%s' did not match '%s'\n", artistName, track.Artist)
			}
		})

		p.visitPage(albumCollector, albumUrl)

		albumCollector.Wait()

		if albumMatch != "" {
			break
		}
	}

	if albumMatch != "" {
		match = p.findSongInAlbumPage(track, albumMatch)
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

				if !utils.MusicItemEquals(track.Name, songName) {
					return true
				}

				// TODO: make this configurable
				alternativeKeywords := []string{"remix", "clean]", "clean)", "edit]", "edit)", "mashup"}

				for _, altKeyword := range alternativeKeywords {
					if !strings.Contains(songName, altKeyword) && strings.Contains(utils.SanitizeForComparison(songName), utils.SanitizeForComparison(altKeyword)) {
						return true
					}
				}

				subheading := strings.ToLower(h.ChildText(".result-info .subhead"))

				if strings.Contains(utils.SanitizeForComparison(subheading), utils.SanitizeForComparison(track.Artist)) {
					songUrl := h.ChildText(".result-info .itemurl")
					possibleTrackMatches = append(possibleTrackMatches, strings.Split(songUrl, "?")[0])

					return false
				} else {
					return true
				}
			})
		})

		p.visitPage(searchCollector, fmt.Sprintf(
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

				if utils.ContainsEastAsianCharacters(track.Artist) {
					isMatch = utils.MusicItemEquals(track.Artist, artistName)
				} else {
					isMatch = utils.SanitizeForComparison(track.Artist) == utils.SanitizeForComparison(artistName)
				}

				if isMatch {
					songName := e.ChildText("h2.trackTitle")
					match = &types.PurchaseableTrack{
						Name:    songName,
						SongUrl: strings.Split(url, "?")[0],
					}
				} else {
					log.Printf("track artist '%s' did not match '%s'\n", artistName, track.Artist)
				}
			})

			p.visitPage(searchCollector, url)

			searchCollector.Wait()

			if match != nil {
				break
			}
		}
	}

	if match == nil {
		return nil
	}

	log.Printf("found track match: '%s' at url %s \n", match.Name, match.SongUrl)

	// now let's find the song details like price

	detailsCollector := p.getNewBandcampCollector()

	detailsCollector.OnHTML(".buyItem.digital", func(e *colly.HTMLElement) {
		priceText := e.ChildText("li.buyItem.digital > .ft > .ft.main-button > span > span.base-text-color")

		log.Printf("price found: '%v'\n", priceText)

		if priceText != "" {
			currencyText := e.ChildText(".ft:nth-child(1) > span:nth-child(2) > span:nth-child(2)")

			log.Println("currencty found", currencyText)

			match.RawPrice = priceText

			price, err := p.parseBandcampPrice(priceText, currencyText)

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

	p.visitPage(detailsCollector, match.SongUrl)

	detailsCollector.Wait()

	return match
}

func (p BandcampProvider) findSongInAlbumPage(track *types.InputTrack, albumPageUrl string) *types.PurchaseableTrack {
	c := p.getNewBandcampCollector()

	var match *types.PurchaseableTrack

	c.OnHTML(".track_table", func(table *colly.HTMLElement) {
		table.ForEachWithBreak(".track_row_view", func(_ int, trackRow *colly.HTMLElement) bool {
			title := trackRow.ChildText(".track-title")

			if utils.MusicItemEquals(track.Name, title) {
				path := trackRow.ChildAttr(".title a", "href")
				match = &types.PurchaseableTrack{
					Name:    title,
					SongUrl: fmt.Sprintf("%s%s", utils.GetBaseURL(albumPageUrl), path),
				}
				return false
			}

			return true
		})
	})

	p.visitPage(c, albumPageUrl)

	c.Wait()

	return match
}

// reference: https://get.bandcamp.help/hc/en-us/articles/23020726236823-Which-currencies-does-Bandcamp-support
func (p BandcampProvider) parseBandcampPrice(rawPrice string, currencyCode string) (*money.Money, error) {
	_, err := currency.ParseISO(currencyCode)

	if err != nil {
		return nil, err
	}

	priceNumber := utils.RemoveNonNumericPrefixSuffix(rawPrice)
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

func (p BandcampProvider) getNewBandcampCollector() *colly.Collector {
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

func (p BandcampProvider) visitPage(c *colly.Collector, url string) {
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
