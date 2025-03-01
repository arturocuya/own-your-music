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
	Index  int    `db:"index"`
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

func fetchSpotifySongsOld(clientId string, clientSecret string) []SpotifySong {
	auth := spotifyauth.New(
		spotifyauth.WithRedirectURL(SPOTIFY_CALLBACK_URL),
		spotifyauth.WithScopes(spotifyauth.ScopeUserReadPrivate, spotifyauth.ScopeUserLibraryRead),
		spotifyauth.WithClientID(clientId),
		spotifyauth.WithClientSecret(clientSecret),
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
			Album:  track.Album.Name,
			Index:  i + 1,
		})
		fmt.Printf("Retrieved track #%d \"%s\" by %s \n", i+1, track.Name, track.Artists[0].Name)
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
				Album:  track.Album.Name,
				Index:  i + offset + 1,
			})
			fmt.Printf("Retrieved track #%d \"%s\" by %s \n", i+offset+1, track.Name, track.Artists[0].Name)
		}

		offset += len(userTracks.Tracks)
	}

	return tracks
}
