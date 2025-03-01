package main

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

type SpotifySong struct {
	Name   string `db:"name"`
	Artist string `db:"artist"`
	Album  string `db:"album"`
	Idx  int    `db:"idx"`
}

const SPOTIFY_CALLBACK_URL = "http://localhost:8081/spotify-auth-callback"

func getSpotifyAuth() (*spotifyauth.Authenticator, error) {
	spotifyClientId, err := GetKeyValue(KEY_SPOTIFY_CLIENT_ID)
	if err != nil {
		return nil, err
	}

	spotifyClientSecret, err := GetKeyValue(KEY_SPOTIFY_CLIENT_SECRET)
	if err != nil {
		return nil, err
	}

	return spotifyauth.New(
		spotifyauth.WithRedirectURL(SPOTIFY_CALLBACK_URL),
		spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopeUserLibraryRead),
		spotifyauth.WithClientID(spotifyClientId),
		spotifyauth.WithClientSecret(spotifyClientSecret),
	), nil
}
