package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

func Homepage(c echo.Context) error {
	db, err := sqlx.Open("sqlite3", DB_PATH)

	if err != nil {
		log.Fatal("error opening database: ", err)
	}

	defer db.Close()

	var pageData struct {
		NeedsCredentials    bool
		Tracks              []SpotifySong
		AuthUrl             string
		CanLoadSpotifySongs bool
	}

	err = db.Select(&pageData.Tracks, "select * from spotify_songs order by \"index\" asc")

	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error fetching existing spotify songs: %v", err))
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

	tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/spotify-tracks.html"))
	return tmpl.Execute(c.Response(), pageData)
}
