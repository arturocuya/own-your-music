package providers

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"ownyourmusic/types"
	"ownyourmusic/utils"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

type AmazonMusicProvider struct{}

func (p AmazonMusicProvider) GetProviderName() string {
	return types.AMAZON_MUSIC_PROVIDER
}

func (p AmazonMusicProvider) FindSong(track *types.InputTrack) *types.PurchaseableTrack {
	// TODO: this will run on a goroutine. how do we keep track of logs in same goroutine?
	log.Printf("amz music checking: %s by %s from %s\n", track.Name, track.Artist, track.Album)

	var match *types.PurchaseableTrack

	/*
		uncomment below to test non headless mode
	*/

	// allocatorContext, cancelAllocator := chromedp.NewExecAllocator(context.Background(), append(
	// 	chromedp.DefaultExecAllocatorOptions[:],
	// 	chromedp.Flag("headless", false),
	// )...)

	// defer cancelAllocator()

	// ctx, cancel := chromedp.NewContext(allocatorContext)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var buf string

	device := GetRandomChromeDevice()

	log.Println("will use device: ", device.Name)

	var pageTitle string

	if err := chromedp.Run(ctx,
		chromedp.Emulate(device),
		chromedp.Navigate(fmt.Sprintf("https://www.amazon.com/s?k=%s&i=digital-music&s=exact-aware-popularity-rank", url.QueryEscape(track.Name+" "+track.Artist))),
		chromedp.WaitVisible("body", chromedp.ByQuery),
		chromedp.Title(&pageTitle),
	); err != nil {
		log.Println("error: from chromedp: ", err)
		return nil
	}

	// TODO: retry with another device
	if pageTitle == "Sorry! Something went wrong!" {
		log.Println("error: amazon returned sorry page")
		return nil
	}

	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(".s-result-list.s-search-results"),
		chromedp.OuterHTML(".s-result-list.s-search-results", &buf),
	); err != nil {
		log.Println("error: from chromedp: ", err)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(buf))

	if err != nil {
		log.Println("error: creating goquery document: ", err)
		return nil
	}

	sections := doc.Find("span[data-csa-c-type='item']")

	sections.Each(func(i int, s *goquery.Selection) {
		if match != nil {
			return
		}

		headingNode := s.Find("[data-cy='title-recipe']")
		heading := headingNode.Text()

		splittedHeading := strings.Split(heading, "by")

		songName := splittedHeading[0]

		if !utils.MusicItemEquals(track.Name, songName) {
			return
		}

		/*
			heading might include song name and artist
			example: "This Charming Man (2011 Remaster) by The Smiths

			or: tv off [feat. Lefty Gunplay] [Explicit]
			(which is by Kendrick but Amazon doesn't include his name in the title)
		*/
		if len(splittedHeading) == 2 {
			if !strings.Contains(utils.SanitizeForComparison(splittedHeading[1]), utils.SanitizeForComparison(track.Artist)) {
				return
			}
		}

		// example: "Or $1.29 to buy MP3"
		buyOption := s.Find("[data-cy='secondary-offer-recipe']").Text()

		if strings.TrimSpace(buyOption) == "" {
			return
		}

		if !strings.Contains(utils.SanitizeForComparison(buyOption), "or") || !strings.Contains(utils.SanitizeForComparison(buyOption), "to buy mp3") {
			log.Printf("error: unexpected wording in buy option: %s\n", buyOption)
			return
		}

		priceText := strings.ReplaceAll(buyOption, "Or", "")
		priceText = strings.ReplaceAll(priceText, "to buy MP3", "")
		priceText = strings.TrimSpace(priceText)

		var currencyCode string

		if strings.HasPrefix(priceText, "$") {
			currencyCode = "USD"
		} else {
			splittedPrice := strings.Fields(priceText)

			if len(splittedPrice) > 1 {
				currencyCode = splittedPrice[0]
			} else {
				log.Printf("error: unexpected structure in buy option: %s\n", buyOption)
				return
			}
		}

		price, err := utils.ParsePrice(priceText, currencyCode)

		if err != nil {
			log.Printf("error: unexpected structure in buy option: %s\n", buyOption)
			return
		}

		songUrlNode := headingNode.Children().First().Nodes[0]

		if songUrlNode.Data == "h2" {
			// we were expecting a link to be the first child of the heading, but it's an h2
			// this happens on small devices. the link is in the parent block
			// to facilitate tapping i guess?
			songUrlNode = headingNode.Parent().Nodes[0]
		}

		if songUrlNode.Data != "a" {
			log.Printf("error: could not find song url node for song: %s by %s", track.Name, track.Artist)
			return
		}

		var songUrl string
		for _, attr := range songUrlNode.Attr {
			if attr.Key == "href" {
				songUrl = "https://www.amazon.com" + attr.Val
				break
			}
		}

		parsedUrl, _ := url.Parse(songUrl)
		queryParams := parsedUrl.Query()

		// this is what tells the browser to scroll to the right song
		trackAsIn := queryParams.Get("trackAsin")

		match = &types.PurchaseableTrack{
			InputTrack: track,
			Name:       strings.TrimSpace(songName),
			Subheading: buyOption,
			SongUrl:    fmt.Sprintf("%s?trackAsin=%s", strings.Split(songUrl, "?")[0], trackAsIn),
			AlbumUrl:   "", // TODO: do we need this?
			RawPrice:   priceText,
			Price:      price,
		}
	})

	return match
}
