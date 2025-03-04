package main

import (
	// "database/sql"

	"log"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err.Error())
	}

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
		idx integer,
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

	/*
		// check if spotify songs table exists
		query := "select name from sqlite_master where type='table' and name=?"
		row := db.QueryRow(query, "spotify_songs")
		var name string

		err = row.Scan(&name)

		if err != nil {
			if err != sql.ErrNoRows {
				log.Fatal("error checking spotify_songs table existence: ", err)
			} else {
				_, err = db.Exec("create table spotify_songs (\"index\" integer, name string, artist string, album string)")

				if err != nil {
					log.Fatal("error creating table spotify_songs: ", err)
				}

				fmt.Println("created new table: spotify_songs")
			}
		}

		// check if table is empty
		query = "select count(*) from spotify_songs"
		var count int
		err = db.QueryRow(query).Scan(&count)

		if err != nil {
			log.Fatal("error checking if spotify_songs is empty: ", err)
		}

		var tracks []SpotifySong

		if count == 0 {
			fmt.Println("will fetch spotify songs")

			tracks = fetchSpotifySongs()

			fmt.Println("num of tracks retrieved: ", len(tracks))

			_, err = db.NamedExec("insert into spotify_songs (name, artist, \"index\", album) values (:name, :artist, :index, :album)", tracks)

			if err != nil {
				log.Fatal("error inserting tracks as batch: ", err)
			}

			fmt.Printf("inserted %d tracks as batch\n", len(tracks))
		} else {
			fmt.Println("spotify_songs has contents. skipping fetching songs")

			err = db.Select(&tracks, "select * from spotify_songs order by \"index\" asc")

			if err != nil {
				log.Fatal("error fetching existing spotify songs: ", err)
			}
		}

		// TODO: parallelize somehow
		for _, track := range tracks {
			match := findSongInBandcamp(&track)

			if match != nil {
				fmt.Printf("\tMatch found! %s / %s : %s\n", match.Name, match.Subheading, match.SongUrl)
			}
		}
	*/
}
