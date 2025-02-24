package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

func startSpotifyCallbackServer(auth *spotifyauth.Authenticator, state string, ch chan *spotify.Client) {
	e := echo.New()
	e.HideBanner = true

	e.GET("/callback", func(c echo.Context) error {
		token, err := auth.Token(c.Request().Context(), state, c.Request())

		if err != nil {
			return c.String(http.StatusForbidden, "could not get token")
		}

		if receivedState := c.FormValue("state"); receivedState != state {
			log.Fatalf("state mismatch: %s != %s", receivedState, state)
			return c.String(http.StatusNotFound, "state mismatch")
		}

		client := spotify.New(auth.Client(c.Request().Context(), token))

		fmt.Println("login completed!")

		ch <- client

		return c.String(http.StatusOK, "spotify auth completed. you may close this page now.")
	})

	go func() {
		e.Logger.Fatal(e.Start(":8080"))
	}()
}

func updateSpotifyCredentials(c echo.Context) error {
	clientId := c.FormValue("clientId")
	clientSecret := c.FormValue("clientSecret")

	err := SetKeyValue(KEY_SPOTIFY_CLIENT_ID, clientId)

	if err != nil {
		return err
	}

	err = SetKeyValue(KEY_SPOTIFY_CLIENT_SECRET, clientSecret)

	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

func spotifyAuthCallback(c echo.Context) error {
	state, err := GetKeyValue(KEY_SPOTIFY_AUTH_STATE)

	if err != nil {
		return c.String(http.StatusForbidden, "could not get state")
	}

	if receivedState := c.FormValue("state"); receivedState != state {
		return c.String(http.StatusOK, "state mismatch")
	}

	spotifyClientId, err := GetKeyValue(KEY_SPOTIFY_CLIENT_ID)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error retrieving spotify client id: %v", err))
	}

	spotifyClientSecret, err := GetKeyValue(KEY_SPOTIFY_CLIENT_SECRET)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("error retrieving spotify client secret: %v", err))
	}

	auth := spotifyauth.New(
		spotifyauth.WithRedirectURL(SPOTIFY_CALLBACK_URL),
		spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopeUserLibraryRead),
		spotifyauth.WithClientID(spotifyClientId),
		spotifyauth.WithClientSecret(spotifyClientSecret),
	)

	token, err := auth.Token(c.Request().Context(), state, c.Request())

	if err != nil {
		return c.String(http.StatusForbidden, "could not get token")
	}

	err = SetSpotifyToken(token)

	if err != nil {
		return c.String(http.StatusForbidden, "could not save token")
	}

	return c.NoContent(http.StatusOK)
}
