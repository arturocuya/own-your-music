package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

type TrackAndMatch struct {
	Track InputTrack
	Match PurchaseableTrack
}

func Homepage(c echo.Context) error {
	db, err := OpenDatabase()

	if err != nil {
		log.Fatal("error opening database: ", err)
	}

	defer db.Close()

	var pageData struct {
		NeedsCredentials    bool
		Tracks              []TrackAndMatch
		AuthUrl             string
		CanLoadSpotifySongs bool
		CanFindSongs        bool
	}

	var tracks []InputTrack

	err = db.Select(&tracks, "select * from spotify_songs order by \"idx\" asc")

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error fetching existing spotify songs: %v", err))
	}

	for _, track := range tracks {
		pageData.Tracks = append(pageData.Tracks,
			TrackAndMatch{
				Track: track,
				// TODO: insert cached match
				Match: PurchaseableTrack{},
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

		pageData.AuthUrl = auth.AuthURL(state)
	}

	pageData.CanLoadSpotifySongs = token != nil && token.Valid()

	pageData.CanFindSongs = !pageData.NeedsCredentials && len(pageData.Tracks) > 0

	tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/track.html", "templates/match-result.html"))

	err = tmpl.Execute(c.Response(), pageData)

	return err
}
