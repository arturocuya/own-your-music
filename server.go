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
