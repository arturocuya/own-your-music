package main

import (
	"github.com/jmoiron/sqlx"
)

const DB_DRIVER = "sqlite3"
const DB_PATH = "./database.sqlite"

func OpenDatabase() (*sqlx.DB, error) {
	return sqlx.Open("sqlite3", DB_PATH)
}
