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
		guild_id TEXT,
		original_title TEXT,
		original_author TEXT,
		recommend_title TEXT,
		recommend_content TEXT,
		original_post_timestamp TEXT,
		final_amway_message_id TEXT
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

	// SQL statement to create the 'id_counter' table for sequential ID generation.
	createIdCounterTableSQL := `
	CREATE TABLE IF NOT EXISTS id_counter (
		counter_name TEXT PRIMARY KEY,
		current_value INTEGER NOT NULL DEFAULT 0
	);`

	_, err = db.Exec(createIdCounterTableSQL)
	if err != nil {
		log.Fatalf("Failed to create id_counter table: %v", err)
	}

	// Initialize the submission counter if it doesn't exist
	_, err = db.Exec("INSERT OR IGNORE INTO id_counter(counter_name, current_value) VALUES('submission_id', 0)")
	if err != nil {
		log.Fatalf("Failed to initialize submission counter: %v", err)
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

// AddSubmission adds a new submission to the recommendations table (legacy version).
func AddSubmission(userID, url, title, content, guildID, authorNickname string) (string, error) {
	return AddSubmissionV2(userID, url, title, content, "", "", "", guildID, authorNickname)
}

// AddSubmissionV2 adds a new submission with original post info and recommendation content.
func AddSubmissionV2(userID, url, recommendTitle, recommendContent, originalTitle, originalAuthor string, originalPostTimestamp string, guildID string, authorNickname string) (string, error) {
	db, err := sql.Open("sqlite3", "./data/amway.db")
	if err != nil {
		return "", err
	}
	defer db.Close()

	// Start a transaction to ensure atomic ID generation
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Get and increment the counter
	var currentID int
	err = tx.QueryRow("SELECT current_value FROM id_counter WHERE counter_name = 'submission_id'").Scan(&currentID)
	if err != nil {
		return "", err
	}

	// Increment the counter
	newID := currentID + 1
	_, err = tx.Exec("UPDATE id_counter SET current_value = ? WHERE counter_name = 'submission_id'", newID)
	if err != nil {
		return "", err
	}

	// Generate submission ID as string
	submissionID := fmt.Sprintf("%d", newID)

	// Insert the new submission with all fields
	stmt, err := tx.Prepare(`INSERT INTO recommendations(
		id, author_id, author_nickname, content, post_url, created_at, guild_id,
		original_title, original_author, recommend_title, recommend_content, original_post_timestamp
	) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	// For legacy compatibility, if original fields are empty, use the old format
	fullContent := recommendContent
	if originalTitle == "" && originalAuthor == "" {
		fullContent = fmt.Sprintf("**%s**\n\n%s", recommendTitle, recommendContent)
	}

	_, err = stmt.Exec(
		submissionID, userID, authorNickname, fullContent, url, time.Now().Unix(), guildID,
		originalTitle, originalAuthor, recommendTitle, recommendContent, originalPostTimestamp,
	)
	if err != nil {
		return "", err
	}

	// Commit the transaction
	err = tx.Commit()
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

	row := db.QueryRow(`SELECT
		id, author_id, COALESCE(author_nickname, '') as author_nickname, content, post_url, created_at,
		COALESCE(guild_id, '') as guild_id,
		COALESCE(original_title, '') as original_title,
		COALESCE(original_author, '') as original_author,
		COALESCE(recommend_title, '') as recommend_title,
		COALESCE(recommend_content, '') as recommend_content,
		COALESCE(original_post_timestamp, '') as original_post_timestamp,
		COALESCE(final_amway_message_id, '') as final_amway_message_id
	FROM recommendations WHERE id = ?`, submissionID)

	var sub model.Submission
	err = row.Scan(
		&sub.ID, &sub.UserID, &sub.AuthorNickname, &sub.Content, &sub.URL, &sub.Timestamp,
		&sub.GuildID, &sub.OriginalTitle, &sub.OriginalAuthor,
		&sub.RecommendTitle, &sub.RecommendContent, &sub.OriginalPostTimestamp, &sub.FinalAmwayMessageID,
	)
	if err != nil {
		return nil, err
	}

	return &sub, nil
}

// UpdateFinalAmwayMessageID updates the final_amway_message_id for a submission.
func UpdateFinalAmwayMessageID(submissionID, messageID string) error {
	db, err := sql.Open("sqlite3", "./data/amway.db")
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("UPDATE recommendations SET final_amway_message_id = ? WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(messageID, submissionID)
	return err
}
