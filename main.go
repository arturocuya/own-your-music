package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

type BandcampArtist struct {
	Name     string
	Location string
	StoreUrl string
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err.Error())
	}

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
		fmt.Printf("checking #%d: %s by %s from %s\n", track.Index, track.Name, track.Artist, track.Album)
		c := colly.NewCollector(
			colly.AllowedDomains("bandcamp.com"),
		)

		c.OnHTML(".results", func(e *colly.HTMLElement) {
			e.ForEachWithBreak(".searchresult", func(i int, h *colly.HTMLElement) bool {
				itemType := h.ChildText(".result-info .itemtype")

				if itemType != "TRACK" {
					return true
				}

				songName := h.ChildText(".result-info .heading")

				if songName != track.Name {
					return true
				}

				subheading := strings.ToLower(h.ChildText(".result-info .subhead"))
				if strings.Contains(subheading, strings.ToLower(track.Artist)) && strings.Contains(subheading, strings.ToLower(track.Album)) {
					url := h.ChildText(".result-info .itemurl")

					fmt.Printf("Match found! %s / %s : %s\n", songName, subheading, url)
					return false
				} else {
					return true
				}
			})
		})

		c.OnError(func(r *colly.Response, err error) {
			fmt.Println("colly on error: ", err, r.Headers.Get("Retry-After"))

			if err.Error() == "Too Many Requests" {
				time.Sleep(3 * time.Minute)
			}
		})

		c.Visit(fmt.Sprintf(
			"https://bandcamp.com/search?q=%s&item_type=t&from=results",
			url.QueryEscape(track.Name),
		))

		c.Wait()
	}
}
