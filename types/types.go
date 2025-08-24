package types

import (
	"fmt"

	"github.com/Rhymond/go-money"
)

type InputTrack struct {
	ProviderName string `db:"provider_name" json:"provider_name"`
	ProviderId   string `db:"provider_id" json:"provider_id"`
	AddedAt      string `db:"added_at" json:"added_at"` // ISO 8601
	Name         string `db:"name" json:"name"`
	Artist       string `db:"artist" json:"artist"`
	Album        string `db:"album" json:"album"`
}

func (t InputTrack) ComposedId() string {
	return fmt.Sprintf("%s--%s", t.ProviderName, t.ProviderId)
}

type PurchaseableTrack struct {
	InputTrack *InputTrack  `json:"input_track"`
	Name       string       `json:"name"`
	Subheading string       `json:"subheading"`
	SongUrl    string       `json:"song_url"`
	AlbumUrl   string       `json:"album_url"`
	RawPrice   string       `json:"raw_price"`
	Price      *money.Money `json:"price"`
}

type TrackAndMatch struct {
	Track InputTrack
	Match PurchaseableTrack
}
