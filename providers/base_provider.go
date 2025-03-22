package providers

import (
	"ownyourmusic/types"
)

type Provider interface {
	FindSong(track *types.InputTrack) *types.PurchaseableTrack
}
