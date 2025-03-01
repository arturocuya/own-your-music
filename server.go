package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

var loadSpotifySongsChan = make(chan []SpotifySong)

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

	auth, err := getSpotifyAuth()
	if err != nil {
		return c.String(http.StatusInternalServerError, "could not get spotify auth object")
	}

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

func serverSentEvents(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set(echo.HeaderCacheControl, "no-cache")
	c.Response().Header().Set(echo.HeaderConnection, "keep-alive")

	c.Response().Header().Set("Access-Control-Allow-Origin", "*")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case tracks := <-loadSpotifySongsChan:
			tmpl, err := template.New("dynamic").Parse(`
				{{range .}}
					<p id="a-{{.Idx}}">{{.Name}} ({{.Album}}) -- {{.Artist}}</p>
				{{end}}
				`)
			if err != nil {
				log.Fatal(err)
			}

			var buf bytes.Buffer
			tmpl.Execute(&buf, tracks)

			content := strings.ReplaceAll(buf.String(), "\n", "")
			content = strings.ReplaceAll(content, "\t", "")

			data := fmt.Sprintf("data: %v\n\n", content)

			if _, err := c.Response().Write([]byte(data)); err != nil {
				return err
			}
			c.Response().Flush()
		}
	}
}

func loadSpotifySongs(c echo.Context) error {
	auth, err := getSpotifyAuth()

	if err != nil {
		return err
	}

	token, err := GetSpotifyToken()

	if err != nil {
		return err
	}

	go func() {
		client := spotify.New(auth.Client(c.Request().Context(), token))

		userTracks, err := client.CurrentUsersTracks(context.Background())

		if err != nil {
			log.Fatal("error getting current user tracks at offset 0: ", err)
		}

		var tracks []SpotifySong

		for i := range (len(userTracks.Tracks))	{
			track := userTracks.Tracks[i]
			tracks = append(tracks, SpotifySong{
				Name:   track.Name,
				Artist: track.Artists[0].Name,
				Album:  track.Album.Name,
				Idx:  i + 1,
			})
			fmt.Printf("Retrieved track #%d \"%s\" by %s \n", i+1, track.Name, track.Artists[0].Name)
		}

		loadSpotifySongsChan <- tracks

		ClearSpotifySongs()

		SaveSpotifySongs(tracks)

		offset := len(userTracks.Tracks)

		for userTracks.Next != "" {
			userTracks, err = client.CurrentUsersTracks(context.Background(), spotify.Offset(offset))

			if err != nil {
				log.Fatalf("error getting current user tracks at offset %d: %s", offset, err)
			}

			tracks = tracks[:0]
			for i := range (len(userTracks.Tracks))	{
				track := userTracks.Tracks[i]
				tracks = append(tracks, SpotifySong{
					Name:   track.Name,
					Artist: track.Artists[0].Name,
					Album:  track.Album.Name,
					Idx:  i + offset + 1,
				})
				fmt.Printf("Retrieved track #%d \"%s\" by %s \n", i+offset+1, track.Name, track.Artists[0].Name)
			}

			loadSpotifySongsChan <- tracks

			SaveSpotifySongs(tracks)

			offset += len(userTracks.Tracks)
		}
	}()

	return c.NoContent(http.StatusOK)
}
