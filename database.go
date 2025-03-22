package main

import (
	"log"
	"ownyourmusic/types"

	"github.com/jmoiron/sqlx"
)

const DB_DRIVER = "sqlite3"
const DB_PATH = "./database.sqlite"

func OpenDatabase() (*sqlx.DB, error) {
	return sqlx.Open("sqlite3", DB_PATH)
}

func SaveSpotifySongs(tracks []types.InputTrack) {
	db, err := OpenDatabase()

	if err != nil {
		log.Fatal("error opening database: ", err)
	}

	defer db.Close()

	_, err = db.NamedExec("insert into spotify_songs (name, artist, album, added_at, provider_name, provider_id) values (:name, :artist, :album, :added_at, :provider_name, :provider_id)", tracks)

	if err != nil {
		log.Fatal("error inserting tracks as batch: ", err)
	}

	log.Printf("inserted %d tracks as batch\n", len(tracks))
}

func ClearSpotifySongs() {
	db, err := OpenDatabase()

	if err != nil {
		log.Fatal("error opening database: ", err)
	}

	defer db.Close()

	_, err = db.Exec("delete from spotify_songs")

	if err != nil {
		log.Fatal("error clearing songs: ", err)
	}
}
