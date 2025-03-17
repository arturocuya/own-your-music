package types

import (
	"strconv"

	"github.com/Rhymond/go-money"
)

type InputTrack struct {
	Name   string `db:"name"`
	Artist string `db:"artist"`
	Album  string `db:"album"`
	Idx    int    `db:"idx"`
}

func (t InputTrack) StrIdx() string {
	return strconv.Itoa(t.Idx)
}

type PurchaseableTrack struct {
	SongIdx    int
	Name       string
	Subheading string
	SongUrl    string
	AlbumUrl   string
	RawPrice   string
	Price      *money.Money
}

type TrackAndMatch struct {
	Track InputTrack
	Match PurchaseableTrack
}
