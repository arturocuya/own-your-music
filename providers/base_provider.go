package providers

import (
	"context"
	"ownyourmusic/types"
)

type Provider interface {
	GetProviderName() string
	FindSong(track *types.InputTrack, parentCtx context.Context) (*types.PurchaseableTrack, error)
}
