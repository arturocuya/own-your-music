package providers_test

import (
	"ownyourmusic/providers"
	"ownyourmusic/types"
	"testing"

	"github.com/Rhymond/go-money"
)

func TestAmzHappyPath(t *testing.T) {
	amz := providers.AmazonMusicProvider{}
	song := types.InputTrack{
		Name:   "This Charming Man - 2011 Remaster",
		Artist: "The Smiths",
		Album:  "The Smiths",
	}

	match := amz.FindSong(&song)

	if match == nil {
		t.Fatalf("match not found for %+v\n", song)
	}

	if match.Name != "This Charming Man (2011 Remaster)" {
		t.Fatalf("unexpected song name: \"%v\"\n", match.Name)
	}

	if match.RawPrice != "$1.29" {
		t.Fatalf("unexpected raw price \"%v\"\n", match.RawPrice)
	}

	price := money.New(129, money.USD)

	priceEql, err := match.Price.Equals(price)

	if err != nil {
		t.Fatal(err)
	}

	if !priceEql {
		t.Fatalf("price \"%v\" does not match \"%v\"\n", match.Price, price)
	}
}

func TestAmzSongWithNoArtistName(t *testing.T) {
	amz := providers.AmazonMusicProvider{}
	song := types.InputTrack{
		Name:   "tv off (feat. lefty gunplay)",
		Artist: "Kendrick Lamar",
		Album:  "GNX",
	}

	match := amz.FindSong(&song)

	if match == nil {
		t.Fatalf("match not found for %+v\n", song)
	}
}
