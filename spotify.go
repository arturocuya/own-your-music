package main

import (
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

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
