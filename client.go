package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"ownyourmusic/templates"
	"ownyourmusic/types"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

func Homepage(c echo.Context) error {
	db, err := OpenDatabase()

	if err != nil {
		log.Fatal("error opening database: ", err)
	}

	defer db.Close()

	pageData := templates.IndexConfig{}

	var pageTracks []types.TrackAndMatch
	var authUrl string

	var tracks []types.InputTrack

	err = db.Select(&tracks, "select * from spotify_songs order by \"added_at\" desc")

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error fetching existing spotify songs: %v", err))
	}

	for _, track := range tracks {
		pageTracks = append(pageTracks,
			types.TrackAndMatch{
				Track: track,
				// TODO: insert cached match
				Match: types.PurchaseableTrack{},
			})
	}

	// check if you need to onboard on spotify
	spotifyClientId, err := GetKeyValue(KEY_SPOTIFY_CLIENT_ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error retrieving spotify client id: %v", err))
	}

	spotifyClientSecret, err := GetKeyValue(KEY_SPOTIFY_CLIENT_SECRET)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error retrieving spotify client secret: %v", err))
	}

	pageData.NeedsCredentials = strings.TrimSpace(spotifyClientId) == "" || strings.TrimSpace(spotifyClientSecret) == ""

	token, err := GetSpotifyToken()

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error retrieving spotify auth token: %v", err))
	}

	if (!pageData.NeedsCredentials && token == nil) || (token != nil && !token.Valid()) {
		auth := spotifyauth.New(
			spotifyauth.WithRedirectURL(SPOTIFY_CALLBACK_URL),
			spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopeUserLibraryRead),
			spotifyauth.WithClientID(spotifyClientId),
			spotifyauth.WithClientSecret(spotifyClientSecret),
		)

		state := uuid.New().String()

		SetKeyValue(KEY_SPOTIFY_AUTH_STATE, state)

		authUrl = auth.AuthURL(state)
	}

	pageData.CanLoadSpotifySongs = token != nil && token.Valid()

	tmpl := templates.Index(authUrl, pageTracks, pageData)

	err = tmpl.Render(context.Background(), c.Response())

	return err
}
