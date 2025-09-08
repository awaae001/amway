package db

import (
	"log"
	"strings"
)

// createTables creates the necessary tables in the database if they don't exist.
func createTables() {
	// SQL statement to create the 'recommendations' table.
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
		status TEXT NOT NULL DEFAULT 'pending',
		guild_id TEXT,
		original_title TEXT,
		original_author TEXT,
		recommend_title TEXT,
		recommend_content TEXT,
		original_post_timestamp TEXT,
		final_amway_message_id TEXT,
		is_deleted INTEGER NOT NULL DEFAULT 0,
		is_anonymous INTEGER NOT NULL DEFAULT 0,
		vote_file_id TEXT,
		thread_message_id TEXT NOT NULL DEFAULT '0'
	);`

	_, err := DB.Exec(createRecommendationsTableSQL)
	if err != nil {
		log.Fatalf("Failed to create recommendations table: %v", err)
	}

	// SQL statement to create the 'users' table.
	createUsersTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		user_id TEXT PRIMARY KEY,
		featured_count INTEGER NOT NULL DEFAULT 0,
		rejected_count INTEGER NOT NULL DEFAULT 0
	);`

	_, err = DB.Exec(createUsersTableSQL)
	if err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}

	// SQL statement to create the 'id_counter' table for sequential ID generation.
	createIdCounterTableSQL := `
	CREATE TABLE IF NOT EXISTS id_counter (
		counter_name TEXT PRIMARY KEY,
		current_value INTEGER NOT NULL DEFAULT 0
	);`

	_, err = DB.Exec(createIdCounterTableSQL)
	if err != nil {
		log.Fatalf("Failed to create id_counter table: %v", err)
	}

	// Initialize the submission counter if it doesn't exist
	_, err = DB.Exec("INSERT OR IGNORE INTO id_counter(counter_name, current_value) VALUES('submission_id', 0)")
	if err != nil {
		log.Fatalf("Failed to initialize submission counter: %v", err)
	}

	// SQL statement to create the 'submission_reactions' table.
	createSubmissionReactionsTableSQL := `
	CREATE TABLE IF NOT EXISTS submission_reactions (
		submission_id TEXT NOT NULL,
		message_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		emoji_name TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		PRIMARY KEY (submission_id, user_id)
	);`

	_, err = DB.Exec(createSubmissionReactionsTableSQL)
	if err != nil {
		log.Fatalf("Failed to create submission_reactions table: %v", err)
	}

	// Add is_deleted column if it doesn't exist (migration for existing databases)
	_, err = DB.Exec("ALTER TABLE recommendations ADD COLUMN is_deleted INTEGER NOT NULL DEFAULT 0")
	if err != nil && !isColumnExistsError(err) {
		log.Printf("Failed to add is_deleted column, it might already exist: %v", err)
	}

	// Add is_anonymous column if it doesn't exist (migration for existing databases)
	_, err = DB.Exec("ALTER TABLE recommendations ADD COLUMN is_anonymous INTEGER NOT NULL DEFAULT 0")
	if err != nil && !isColumnExistsError(err) {
		log.Printf("Failed to add is_anonymous column, it might already exist: %v", err)
	}

	// Add status column if it doesn't exist (migration for existing databases)
	_, err = DB.Exec("ALTER TABLE recommendations ADD COLUMN status TEXT NOT NULL DEFAULT 'pending'")
	if err != nil && !isColumnExistsError(err) {
		log.Printf("Failed to add status column, it might already exist: %v", err)
	}

	// Add vote_file_id column if it doesn't exist (migration for existing databases)
	_, err = DB.Exec("ALTER TABLE recommendations ADD COLUMN vote_file_id TEXT")
	if err != nil && !isColumnExistsError(err) {
		log.Printf("Failed to add vote_file_id column, it might already exist: %v", err)
	}

	// Migration for ban system
	_, err = DB.Exec("ALTER TABLE users ADD COLUMN ban_count INTEGER NOT NULL DEFAULT 0")
	if err != nil && !isColumnExistsError(err) {
		log.Printf("Failed to add ban_count column, it might already exist: %v", err)
	}

	_, err = DB.Exec("ALTER TABLE users ADD COLUMN is_permanently_banned BOOLEAN NOT NULL DEFAULT 0")
	if err != nil && !isColumnExistsError(err) {
		log.Printf("Failed to add is_permanently_banned column, it might already exist: %v", err)
	}

	_, err = DB.Exec("ALTER TABLE users ADD COLUMN banned_until INTEGER")
	if err != nil && !isColumnExistsError(err) {
		log.Printf("Failed to add banned_until column, it might already exist: %v", err)
	}

	log.Println("Database tables initialized successfully.")
}

// isColumnExistsError checks if the error is due to column already existing
func isColumnExistsError(err error) bool {
	return strings.Contains(err.Error(), "duplicate column name")
}
