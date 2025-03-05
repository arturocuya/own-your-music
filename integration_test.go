package main

import (
	"fmt"
	"testing"

	"github.com/Rhymond/go-money"
)

func TestHappyPath(t *testing.T) {
	song := InputTrack{
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

	fmt.Printf("match found: %+v\n", match)
}

func TestSpecialCharacters(t *testing.T) {
	song := InputTrack{
		Idx:    1,
		Name:   "still feel.",
		Artist: "half•alive",
		Album:  "Now, Not Yet",
	}

	match := findSongInBandcamp(&song)

	if match == nil {
		t.Fatalf("match not found for %+v\n", song)
	}

	fmt.Printf("match found: %+v\n", match)
}

func TestSongThatShouldntExist(t *testing.T) {
	song := InputTrack{
		Idx:    1,
		Name:   "Not Like Us",
		Artist: "Kendrick Lamar",
		Album:  "Not Like Us",
	}

	match := findSongInBandcamp(&song)

	if match != nil {
		t.Fatal("found song that shouldn't exist in bandcamp", song.Name, "by", song.Artist)
	}
}

func TestSongWithForeignPriceAndNoAlbum(t *testing.T) {
	song := InputTrack{
		Idx:    1,
		Name:   "Fløjlstordensky",
		Artist: "Hong Kong",
		Album:  "Fløjlstordensky",
	}

	match := findSongInBandcamp(&song)

	// TODO: expect price in DKK

	if match == nil {
		t.Fatal("could not find song")
	}
}

func TestFreeNameYourPriceSong(t *testing.T) {
	song := InputTrack{
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
