package main

import (
	"ownyourmusic/types"
	"testing"

	"github.com/Rhymond/go-money"
)

func TestHappyPath(t *testing.T) {
	song := types.InputTrack{
		Idx:    1,
		Name:   "The Bay",
		Artist: "Metronomy",
		Album:  "The English Riviera",
	}

	match := findSongInBandcamp(&song)

	if match == nil {
		t.Fatalf("match not found for %+v\n", song)
	}

	if match.SongIdx != song.Idx {
		t.Fatalf("song idx %v does not match original %v\n", match.SongIdx, song.Idx)
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
	song := types.InputTrack{
		Idx:    1,
		Name:   "still feel.",
		Artist: "half•alive",
		Album:  "Now, Not Yet",
	}

	match := findSongInBandcamp(&song)

	if match == nil {
		t.Fatalf("match not found for %+v\n", song)
	}
}

func TestSongThatShouldntExist(t *testing.T) {
	song := types.InputTrack{
		Idx:    1,
		Name:   "Not Like Us",
		Artist: "Kendrick Lamar",
		Album:  "Not Like Us",
	}

	match := findSongInBandcamp(&song)

	if match != nil {
		t.Fatalf("found song that shouldn't exist in bandcamp: %+v", match)
	}
}

func TestSongWithForeignPriceAndNoAlbum(t *testing.T) {
	song := types.InputTrack{
		Idx:    1,
		Name:   "Fløjlstordensky",
		Artist: "Hong Kong",
		Album:  "Fløjlstordensky",
	}

	match := findSongInBandcamp(&song)

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
	song := types.InputTrack{
		Idx:    1,
		Name:   "Tree Among Shrubs",
		Artist: "Men I Trust",
		Album:  "Untourable Album",
	}

	match := findSongInBandcamp(&song)

	if match.Price == nil {
		t.Fatal("free song price was not set at all")
	}

	if equals, _ := match.Price.Equals(money.New(0, money.USD)); !equals {
		t.Fatal("free song price is not USD 0")
	}
}

func TestSongWithAlbumThatShouldNotExist(t *testing.T) {
	song := types.InputTrack{
		Idx:    1,
		Name:   "reincarnated",
		Artist: "Kendrick Lamar",
		Album:  "GNX",
	}

	match := findSongInBandcamp(&song)

	if match != nil {
		t.Fatalf("this song should not exist, but match was found: %+v", match)
	}
}

func TestFindJapaneseSong(t *testing.T) {
	song := types.InputTrack{
		Idx:    1,
		Name:   "愛のせいで",
		Artist: "ZOMBIE-CHANG",
		Album:  "PETIT PETIT PETIT",
	}
	match := findSongInBandcamp(&song)

	if match == nil {
		t.Fatalf("this song by zombie chang should exist!")
	}
}

// TODO: what about "Y Dime" vs "Y ... Dime" from Ms Nina?
func TestFindSongWithEllipsis(t *testing.T) {
	song := types.InputTrack{
		Idx:    1,
		Name:   "Y Dime (feat. Tomasa del Real)",
		Artist: "Ms Nina",
		Album:  "Y Dime",
	}
	match := findSongInBandcamp(&song)

	if match == nil {
		t.Fatalf("this song by ms nina should exist!")
	}
}
