package main

import (
	"database/sql"
	"encoding/json"

	"github.com/labstack/gommon/log"
	"golang.org/x/oauth2"
)

const KEY_SPOTIFY_CLIENT_ID = "spotify_client_id"
const KEY_SPOTIFY_CLIENT_SECRET = "spotify_client_secret"
const KEY_SPOTIFY_AUTH_STATE = "spotify_auth_state"
const KEY_SPOTIFY_AUTH_TOKEN = "spotify_auth_token"

func GetKeyValue(key string) (string, error) {
	db, openErr := OpenDatabase()

	if openErr != nil {
		log.Errorf("Error opening db %+v", openErr)
		return "", openErr
	}

	defer db.Close()

	var value string

	queryErr := db.QueryRowx("select value from kvstore where key = ?", key).Scan(&value)

	if queryErr == sql.ErrNoRows {
		return "", nil
	} else if queryErr != nil {
		log.Errorf("Error querying key \"%s\": %+v", key, queryErr)
		return "", queryErr
	}

	return value, nil
}

func SetKeyValue(key string, value string) error {
	db, openErr := OpenDatabase()

	if openErr != nil {
		log.Errorf("Error opening db %+v", openErr)
		return openErr
	}

	defer db.Close()

	_, upsertErr := db.Exec("insert or replace into kvstore (key, value) values (?, ?)", key, value)

	if upsertErr != nil {
		log.Errorf("Error upserting key \"%s\" with value \"%s\": %+v", key, value, upsertErr)
		return upsertErr
	}

	return nil
}

func SetSpotifyToken(token *oauth2.Token) error {
	tokenJSON, err := json.Marshal(token)

	if err != nil {
		return err
	}

	err = SetKeyValue("spotify_auth_token", string(tokenJSON))

	if err != nil {
		return err
	}

	return nil
}

func GetSpotifyToken() (*oauth2.Token, error) {
	tokenJSON, err := GetKeyValue("spotify_auth_token")

	if err != nil {
		return nil, err
	}

	if tokenJSON == "" {
		return nil, nil
	}

	var token oauth2.Token
	err = json.Unmarshal([]byte(tokenJSON), &token)
	if err != nil {
		return nil, err
	}

	return &token, nil
}
