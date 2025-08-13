package utils

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB initializes the SQLite database and creates tables if they don't exist.
// It is based on the initial user request and the provided protobuf schema for recommendations.
func InitDB() {
	db, err := sql.Open("sqlite3", "./data/amway.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// SQL statement to create the 'recommendations' table.
	// This schema is derived from the 'RecommendationSlip' protobuf message.
	createRecommendationsTableSQL := `
	CREATE TABLE IF NOT EXISTS recommendations (
		id TEXT PRIMARY KEY,
		author_id TEXT NOT NULL,
		author_nickname TEXT,
		content TEXT NOT NULL,
		post_url TEXT,
		upvotes INTEGER NOT NULL DEFAULT 0,
		questions INTEGER NOT NULL DEFAULT 0,
		downvotes INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		reviewer_id TEXT,
		is_blocked INTEGER NOT NULL DEFAULT 0,
		guild_id INTEGER NOT NULL
	);`

	_, err = db.Exec(createRecommendationsTableSQL)
	if err != nil {
		log.Fatalf("Failed to create recommendations table: %v", err)
	}

	// SQL statement to create the 'banned_users' table.
	createBannedUsersTableSQL := `
	CREATE TABLE IF NOT EXISTS banned_users (
		user_id TEXT PRIMARY KEY,
		reason TEXT,
		timestamp INTEGER NOT NULL
	);`

	_, err = db.Exec(createBannedUsersTableSQL)
	if err != nil {
		log.Fatalf("Failed to create banned_users table: %v", err)
	}

	log.Println("Database and tables initialized successfully in data/amway.db")
}
