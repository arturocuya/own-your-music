package main

import (
	// "database/sql"

	"context"
	"fmt"
	"log"
	"ownyourmusic/providers"
	"ownyourmusic/types"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
)

type LambdaEvent struct {
	InputTrack types.InputTrack `json:"input_track"`
}

type LambdaResponse struct {
	Result *types.PurchaseableTrack `json:"result"`
	Err    string                   `json:"error"`
}

func handler(ctx context.Context, event LambdaEvent) (LambdaResponse, error) {
	amz := providers.AmazonMusicProvider{}
	song := types.InputTrack{
		Name:   "This Charming Man - 2011 Remaster",
		Artist: "The Smiths",
		Album:  "The Smiths",
	}

	match, err := amz.FindSong(&song, ctx)

	fmt.Printf("match: %+v\n", match)

	if err != nil {
		return LambdaResponse{
			Result: nil,
			Err:    err.Error(),
		}, nil
	}

	return LambdaResponse{
		Result: match,
		Err:    "",
	}, nil
}

func main() {
	lambda.Start(handler)
}

func main2() {
	initLogger()

	// we have a main database that containes the cached list of spotify songs
	// and also the cached results for bandcamp, amazon, etc.
	dbPath := "./database.sqlite"

	db, err := sqlx.Open("sqlite3", dbPath)

	if err != nil {
		log.Fatal("error opening database: ", err)
	}

	defer db.Close()

	err = db.Ping()

	if err != nil {
		log.Fatal("error pinging db", err)
	}

	_, err = db.Exec(`create table if not exists spotify_songs(
		added_at string,
		provider_name string,
		provider_id string,
		name string,
		artist string,
		album string
	)`)

	if err != nil {
		log.Fatalf("error creating table spotify_songs: %v", err)
	}

	_, err = db.Exec(`create table if not exists kvstore(
			key text primary key,
			value text
		)`)

	if err != nil {
		log.Fatalf("error creating table kvstore: %v", err)
	}

	e := echo.New()
	e.Static("/public", "public")

	e.GET("/", Homepage)
	e.POST("/update-spotify-credentials", updateSpotifyCredentials)
	e.GET("/spotify-auth-callback", spotifyAuthCallback)
	e.GET("/sse", serverSentEvents)
	e.GET("/load-spotify-songs", loadSpotifySongs)
	e.GET("/find-songs", findSongs)

	e.Logger.Fatal(e.Start(":8081"))
}
