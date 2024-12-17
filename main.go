package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err.Error())
	}

	auth := spotifyauth.New(
		spotifyauth.WithRedirectURL("http://localhost:8080/callback"),
		spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopeUserLibraryRead),
		spotifyauth.WithClientID(os.Getenv("SPOTIFY_CLIENT_ID")),
		spotifyauth.WithClientSecret(os.Getenv("SPOTIFY_CLIENT_SECRET")),
	)

	fmt.Println("client id:", os.Getenv("SPOTIFY_ID"))

	ch := make(chan *spotify.Client)

	state := uuid.New().String()

	e := echo.New()

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

		return c.String(http.StatusOK, "ok")
	})

	go func() {
		e.Logger.Fatal(e.Start(":8080"))
	}()

	url := auth.AuthURL(state)

	fmt.Printf("login here: %v", url)

	client := <-ch

	tracks, err := client.CurrentUsersTracks(context.Background())

	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(tracks.Tracks); i++ {
		track := tracks.Tracks[i]
		fmt.Println(i, track.Name, track.Artists[0].Name)
	}
}
