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
	"github.com/chromedp/chromedp/device"
)

type AmazonMusicProvider struct{}

func (p AmazonMusicProvider) GetProviderName() string {
	return types.AMAZON_MUSIC_PROVIDER
}

func (p AmazonMusicProvider) FindSong(track *types.InputTrack) *types.PurchaseableTrack {
	// TODO: this will run on a goroutine. how do we keep track of logs in same goroutine?
	log.Printf("amz music checking: %s by %s from %s\n", track.Name, track.Artist, track.Album)

	var match *types.PurchaseableTrack

	// uncomment to test non headless mode
	// allocatorContext, cancelAllocator := chromedp.NewExecAllocator(context.Background(), append(
	// 	chromedp.DefaultExecAllocatorOptions[:],
	// 	chromedp.Flag("headless", false),
	// )...)

	// defer cancelAllocator()

	// ctx, cancel := chromedp.NewContext(allocatorContext)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var buf string

	if err := chromedp.Run(ctx,
		chromedp.Emulate(device.IPadPro11landscape),
		chromedp.Navigate(fmt.Sprintf("https://www.amazon.com/s?k=%s&i=digital-music&s=exact-aware-popularity-rank", url.QueryEscape(track.Name+" "+track.Artist))),
		chromedp.WaitVisible(".s-result-list.s-search-results"),
		chromedp.OuterHTML(".s-result-list.s-search-results", &buf),
	); err != nil {
		log.Println("error running chromedp: ", err)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(buf))

	if err != nil {
		log.Println("error creating goquery document: ", err)
		return nil
	}

	sections := doc.Find("span[data-csa-c-type='item']")

	sections.Each(func(i int, s *goquery.Selection) {
		if match != nil {
			return
		}

		// includes song name and artist
		// example: "This Charming Man (2011 Remaster) by The Smiths
		headingNode := s.Find("[data-cy='title-recipe']")
		heading := headingNode.Text()

		splittedHeading := strings.Split(heading, " by ")

		if len(splittedHeading) != 2 {
			log.Printf("error: unexpected heading size %d for: %s\n", len(splittedHeading), heading)
			return
		}

		songName := splittedHeading[0]
		artist := splittedHeading[1]

		if !utils.MusicItemEquals(track.Name, songName) {
			return
		}

		if !strings.Contains(utils.SanitizeForComparison(artist), utils.SanitizeForComparison(track.Artist)) {
			return
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

		headingHeir := headingNode.Children().First().Nodes[0]

		if headingHeir.Data != "a" {
			log.Printf("error: first child of heading was not an 'a' tag. instead: %s\n", headingHeir.Data)
			return
		}

		var songUrl string
		for _, attr := range headingHeir.Attr {
			if attr.Key == "href" {
				songUrl = attr.Val
				break
			}
		}

		match = &types.PurchaseableTrack{
			InputTrack: track,
			Name: songName,
			Subheading: buyOption,
			SongUrl: songUrl,
			AlbumUrl: "", // TODO: do we need this?
			RawPrice: priceText,
			Price: price,
		}
	})

	return match
}
