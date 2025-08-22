package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbDriver = "sqlite3"
	dbSource = "./data/amway.db"
)

// DB is the global database connection pool.
var DB *sql.DB

// InitDB initializes the SQLite database and creates tables if they don't exist.
func InitDB() {
	var err error
	DB, err = sql.Open(dbDriver, dbSource)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// createTables is defined in migrate.go
	createTables()

	log.Println("Database connection initialized successfully.")
}
