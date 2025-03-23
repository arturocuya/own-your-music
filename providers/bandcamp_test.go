package providers_test

import (
	"ownyourmusic/providers"
	"ownyourmusic/types"
	"testing"

	"github.com/Rhymond/go-money"
)

func TestHappyPath(t *testing.T) {
	bc := providers.BandcampProvider{}
	song := types.InputTrack{
		Name:   "The Bay",
		Artist: "Metronomy",
		Album:  "The English Riviera",
	}

	match := bc.FindSong(&song)

	if match == nil {
		t.Fatalf("match not found for %+v\n", song)
	}

	if match.Name != "The Bay" {
		t.Fatalf("unexpected song name: \"%v\"\n", match.Name)
	}

	if match.RawPrice != "€1" {
		t.Fatalf("unexpected raw price \"%v\"\n", match.RawPrice)
	}

	oneEuro := money.New(100, money.EUR)

	priceEql, err := match.Price.Equals(oneEuro)

	if err != nil {
		t.Fatal(err)
	}

	if !priceEql {
		t.Fatalf("price \"%v\" does not match \"%v\"\n", match.Price, oneEuro)
	}
}

func TestSpecialCharacters(t *testing.T) {
	bc := providers.BandcampProvider{}
	song := types.InputTrack{
		Name:   "still feel.",
		Artist: "half•alive",
		Album:  "Now, Not Yet",
	}

	match := bc.FindSong(&song)

	if match == nil {
		t.Fatalf("match not found for %+v\n", song)
	}
}

func TestSongThatShouldntExist(t *testing.T) {
	bc := providers.BandcampProvider{}
	song := types.InputTrack{
		Name:   "Not Like Us",
		Artist: "Kendrick Lamar",
		Album:  "Not Like Us",
	}

	match := bc.FindSong(&song)

	if match != nil {
		t.Fatalf("found song that shouldn't exist in bandcamp: %+v", match)
	}
}

func TestSongWithForeignPriceAndNoAlbum(t *testing.T) {
	bc := providers.BandcampProvider{}
	song := types.InputTrack{
		Name:   "Fløjlstordensky",
		Artist: "Hong Kong",
		Album:  "Fløjlstordensky",
	}

	match := bc.FindSong(&song)

	if match == nil {
		t.Fatal("could not find song")
	}

	if match.Price == nil {
		t.Fatal("price is nil")
	}

	if match.Price.Currency().Code != money.DKK {
		t.Fatal("unexpected price currency")
	}
}

func TestFreeNameYourPriceSong(t *testing.T) {
	bc := providers.BandcampProvider{}
	song := types.InputTrack{
		Name:   "Tree Among Shrubs",
		Artist: "Men I Trust",
		Album:  "Untourable Album",
	}

	match := bc.FindSong(&song)

	if match.Price == nil {
		t.Fatal("free song price was not set at all")
	}

	if equals, _ := match.Price.Equals(money.New(0, money.USD)); !equals {
		t.Fatal("free song price is not USD 0")
	}
}

func TestSongWithAlbumThatShouldNotExist(t *testing.T) {
	bc := providers.BandcampProvider{}
	song := types.InputTrack{
		Name:   "reincarnated",
		Artist: "Kendrick Lamar",
		Album:  "GNX",
	}

	match := bc.FindSong(&song)

	if match != nil {
		t.Fatalf("this song should not exist, but match was found: %+v", match)
	}
}

func TestFindJapaneseSong(t *testing.T) {
	bc := providers.BandcampProvider{}
	song := types.InputTrack{
		Name:   "愛のせいで",
		Artist: "ZOMBIE-CHANG",
		Album:  "PETIT PETIT PETIT",
	}
	match := bc.FindSong(&song)

	if match == nil {
		t.Fatalf("this song by zombie chang should exist!")
	}
}

func TestFindSongWithEllipsis(t *testing.T) {
	bc := providers.BandcampProvider{}
	song := types.InputTrack{
		Name:   "Y Dime (feat. Tomasa del Real)",
		Artist: "Ms Nina",
		Album:  "Y Dime",
	}
	match := bc.FindSong(&song)

	if match == nil {
		t.Fatalf("this song by ms nina should exist!")
	}
}

func TestSongWithNoAlbumMatchButHasTrackMatch(t *testing.T) {
	bc := providers.BandcampProvider{}
	song := types.InputTrack{
		Name:   "Run Your Mouth",
		Artist: "The Marías",
		Album:  "Submarine",
	}
	match := bc.FindSong(&song)

	if match == nil {
		t.Fatalf("this the marias song should exist!")
	}

	if match.InputTrack == nil {
		t.Fatalf("match did not have input track ref")
	}
}
