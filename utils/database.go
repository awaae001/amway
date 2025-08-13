package utils

import (
	"amway/model"
	"database/sql"
	"fmt"
	"log"
	"time"

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

// IsUserBanned checks if a user is in the banned_users table.
func IsUserBanned(userID string) (bool, error) {
	db, err := sql.Open("sqlite3", "./data/amway.db")
	if err != nil {
		return false, err
	}
	defer db.Close()

	var id string
	err = db.QueryRow("SELECT user_id FROM banned_users WHERE user_id = ?", userID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // User is not banned
		}
		return false, err // An actual error occurred
	}

	return true, nil // User is found in the banned list
}

// AddSubmission adds a new submission to the recommendations table.
func AddSubmission(userID, url, title, content string) (string, error) {
	db, err := sql.Open("sqlite3", "./data/amway.db")
	if err != nil {
		return "", err
	}
	defer db.Close()

	// Generate a unique ID (using timestamp + userID for simplicity)
	submissionID := fmt.Sprintf("%d_%s", time.Now().Unix(), userID)

	stmt, err := db.Prepare(`INSERT INTO recommendations(id, author_id, content, post_url, created_at, guild_id)
		VALUES(?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	// Combine title and content for the content field
	fullContent := fmt.Sprintf("**%s**\n\n%s", title, content)

	_, err = stmt.Exec(submissionID, userID, fullContent, url, time.Now().Unix(), 0)
	if err != nil {
		return "", err
	}

	return submissionID, nil
}

// UpdateSubmissionStatus updates the status of a submission in recommendations table.
func UpdateSubmissionStatus(submissionID, status string) error {
	db, err := sql.Open("sqlite3", "./data/amway.db")
	if err != nil {
		return err
	}
	defer db.Close()

	// We'll use is_blocked field to track status: 0=pending, 1=approved, 2=rejected, 3=ignored
	var isBlocked int
	switch status {
	case "approved":
		isBlocked = 1
	case "rejected":
		isBlocked = 2
	case "ignored":
		isBlocked = 3
	default: // pending
		isBlocked = 0
	}

	stmt, err := db.Prepare("UPDATE recommendations SET is_blocked = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(isBlocked, submissionID)
	return err
}

// DeleteSubmission removes a submission from the recommendations table.
func DeleteSubmission(submissionID string) error {
	db, err := sql.Open("sqlite3", "./data/amway.db")
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("DELETE FROM recommendations WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(submissionID)
	return err
}

// BanUser adds a user to the banned_users table.
func BanUser(userID, reason string) error {
	db, err := sql.Open("sqlite3", "./data/amway.db")
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT OR REPLACE INTO banned_users(user_id, reason, timestamp) VALUES(?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(userID, reason, time.Now().Unix())
	return err
}

// GetSubmission retrieves a submission by its ID from recommendations table.
func GetSubmission(submissionID string) (*model.Submission, error) {
	db, err := sql.Open("sqlite3", "./data/amway.db")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	row := db.QueryRow("SELECT id, author_id, content, post_url, created_at FROM recommendations WHERE id = ?", submissionID)

	var sub model.Submission
	err = row.Scan(&sub.ID, &sub.UserID, &sub.Content, &sub.URL, &sub.Timestamp)
	if err != nil {
		return nil, err
	}

	return &sub, nil
}
