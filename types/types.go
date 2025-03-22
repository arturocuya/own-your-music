package types

import (
	"fmt"

	"github.com/Rhymond/go-money"
)

type InputTrack struct {
	ProviderName string `db:"provider_name"`
	ProviderId string `db:"provider_id"`
	AddedAt string `db:"added_at"` // ISO 8601
	Name    string `db:"name"`
	Artist  string `db:"artist"`
	Album   string `db:"album"`
}

func (t InputTrack) ComposedId() string {
	return fmt.Sprintf("%s--%s", t.ProviderName, t.ProviderId)
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
