package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

type SpotifySong struct {
	Name   string `db:"name"`
	Artist string `db:"artist"`
	Index  int    `db:"index"`
}

func fetchSpotifySongs() []SpotifySong {
	auth := spotifyauth.New(
		spotifyauth.WithRedirectURL("http://localhost:8080/callback"),
		spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopeUserLibraryRead),
		spotifyauth.WithClientID(os.Getenv("SPOTIFY_CLIENT_ID")),
		spotifyauth.WithClientSecret(os.Getenv("SPOTIFY_CLIENT_SECRET")),
	)

	ch := make(chan *spotify.Client)

	state := uuid.New().String()

	startSpotifyCallbackServer(auth, state, ch)

	url := auth.AuthURL(state)

	fmt.Printf("login here to start process: %v\n", url)
	OpenUrlInBrowser(url)

	client := <-ch

	var tracks []SpotifySong

	// TODO: parallelize somehow
	userTracks, err := client.CurrentUsersTracks(context.Background())

	if err != nil {
		log.Fatal("error getting current user tracks at offset 0: ", err)
	}

	for i := 0; i < len(userTracks.Tracks); i++ {
		track := userTracks.Tracks[i]
		tracks = append(tracks, SpotifySong{
			Name:   track.Name,
			Artist: track.Artists[0].Name,
			Index:  i,
		})
		fmt.Printf("Retrieved track #%d \"%s\" by %s \n", i, track.Name, track.Artists[0].Name)
	}

	offset := len(userTracks.Tracks)

	for userTracks.Next != "" {
		userTracks, err = client.CurrentUsersTracks(context.Background(), spotify.Offset(offset))

		if err != nil {
			log.Fatalf("error getting current user tracks at offset %d: %s", offset, err)
		}

		for i := 0; i < len(userTracks.Tracks); i++ {
			track := userTracks.Tracks[i]
			tracks = append(tracks, SpotifySong{
				Name:   track.Name,
				Artist: track.Artists[0].Name,
				Index:  i + offset,
			})
			fmt.Printf("Retrieved track #%d \"%s\" by %s \n", i+offset, track.Name, track.Artists[0].Name)
		}

		offset += len(userTracks.Tracks)
	}

	return tracks
}
