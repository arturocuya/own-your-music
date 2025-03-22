package providers

import (
	"ownyourmusic/types"
)

type Provider interface {
	GetProviderName() string
	FindSong(track *types.InputTrack) *types.PurchaseableTrack
}
