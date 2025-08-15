package utils

import (
	"amway/model"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

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

	createTables()
}

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
		is_blocked INTEGER NOT NULL DEFAULT 0,
		guild_id TEXT,
		original_title TEXT,
		original_author TEXT,
		recommend_title TEXT,
		recommend_content TEXT,
		original_post_timestamp TEXT,
		final_amway_message_id TEXT,
		is_deleted INTEGER NOT NULL DEFAULT 0
	);`

	_, err := DB.Exec(createRecommendationsTableSQL)
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

	_, err = DB.Exec(createBannedUsersTableSQL)
	if err != nil {
		log.Fatalf("Failed to create banned_users table: %v", err)
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

	// Add is_deleted column if it doesn't exist (migration for existing databases)
	_, err = DB.Exec("ALTER TABLE recommendations ADD COLUMN is_deleted INTEGER NOT NULL DEFAULT 0")
	if err != nil && !isColumnExistsError(err) {
		log.Fatalf("Failed to add is_deleted column: %v", err)
	}

	log.Println("Database and tables initialized successfully in", dbSource)
}

// isColumnExistsError checks if the error is due to column already existing
func isColumnExistsError(err error) bool {
	return strings.Contains(err.Error(), "duplicate column name")
}

// IsUserBanned checks if a user is in the banned_users table.
func IsUserBanned(userID string) (bool, error) {
	var id string
	err := DB.QueryRow("SELECT user_id FROM banned_users WHERE user_id = ?", userID).Scan(&id)
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
	tx, err := DB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback() // Rollback on error

	var currentID int
	err = tx.QueryRow("SELECT current_value FROM id_counter WHERE counter_name = 'submission_id'").Scan(&currentID)
	if err != nil {
		return "", err
	}

	newID := currentID + 1
	_, err = tx.Exec("UPDATE id_counter SET current_value = ? WHERE counter_name = 'submission_id'", newID)
	if err != nil {
		return "", err
	}

	submissionID := fmt.Sprintf("%d", newID)

	stmt, err := tx.Prepare(`INSERT INTO recommendations(
		id, author_id, author_nickname, content, post_url, created_at, guild_id,
		original_title, original_author, recommend_title, recommend_content, original_post_timestamp
	) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return "", err
	}
	defer stmt.Close()

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

	return submissionID, tx.Commit()
}

// UpdateSubmissionStatus updates the status of a submission in recommendations table.
func UpdateSubmissionStatus(submissionID, status string) error {
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

	_, err := DB.Exec("UPDATE recommendations SET is_blocked = ? WHERE id = ?", isBlocked, submissionID)
	return err
}

// DeleteSubmission removes a submission from the recommendations table.
func DeleteSubmission(submissionID string) error {
	_, err := DB.Exec("DELETE FROM recommendations WHERE id = ?", submissionID)
	return err
}

// BanUser adds a user to the banned_users table.
func BanUser(userID, reason string) error {
	_, err := DB.Exec("INSERT OR REPLACE INTO banned_users(user_id, reason, timestamp) VALUES(?, ?, ?)", userID, reason, time.Now().Unix())
	return err
}

// rowScanner is an interface that can be satisfied by *sql.Row or *sql.Rows.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanSubmission scans a row into a Submission struct.
func scanSubmission(scanner rowScanner) (*model.Submission, error) {
	var sub model.Submission
	err := scanner.Scan(
		&sub.ID, &sub.UserID, &sub.AuthorNickname, &sub.Content, &sub.URL, &sub.Timestamp,
		&sub.GuildID, &sub.OriginalTitle, &sub.OriginalAuthor,
		&sub.RecommendTitle, &sub.RecommendContent, &sub.OriginalPostTimestamp, &sub.FinalAmwayMessageID,
		&sub.Upvotes, &sub.Questions, &sub.Downvotes,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil, nil if no submission is found
		}
		return nil, err
	}
	return &sub, nil
}

// GetSubmission retrieves a submission by its ID from recommendations table (excludes deleted).
func GetSubmission(submissionID string) (*model.Submission, error) {
	row := DB.QueryRow(`SELECT
		id, author_id, COALESCE(author_nickname, '') as author_nickname, content, post_url, created_at,
		COALESCE(guild_id, '') as guild_id,
		COALESCE(original_title, '') as original_title,
		COALESCE(original_author, '') as original_author,
		COALESCE(recommend_title, '') as recommend_title,
		COALESCE(recommend_content, '') as recommend_content,
		COALESCE(original_post_timestamp, '') as original_post_timestamp,
		COALESCE(final_amway_message_id, '') as final_amway_message_id,
		upvotes, questions, downvotes
	FROM recommendations WHERE id = ? AND is_deleted = 0`, submissionID)

	return scanSubmission(row)
}

// UpdateFinalAmwayMessageID updates the final_amway_message_id for a submission.
func UpdateFinalAmwayMessageID(submissionID, messageID string) error {
	_, err := DB.Exec("UPDATE recommendations SET final_amway_message_id = ? WHERE id = ?", messageID, submissionID)
	return err
}

// GetSubmissionByMessageID retrieves a submission by its final message ID (excludes deleted).
func GetSubmissionByMessageID(messageID string) (*model.Submission, error) {
	row := DB.QueryRow(`SELECT
		id, author_id, COALESCE(author_nickname, '') as author_nickname, content, post_url, created_at,
		COALESCE(guild_id, '') as guild_id,
		COALESCE(original_title, '') as original_title,
		COALESCE(original_author, '') as original_author,
		COALESCE(recommend_title, '') as recommend_title,
		COALESCE(recommend_content, '') as recommend_content,
		COALESCE(original_post_timestamp, '') as original_post_timestamp,
		COALESCE(final_amway_message_id, '') as final_amway_message_id,
		upvotes, questions, downvotes
	FROM recommendations WHERE final_amway_message_id = ? AND is_deleted = 0`, messageID)

	return scanSubmission(row)
}

// UpdateReactionCount updates the reaction counts for a submission.
func UpdateReactionCount(submissionID string, emojiName string, increment int) error {
	var fieldToUpdate string
	switch emojiName {
	case "👍":
		fieldToUpdate = "upvotes"
	case "❓":
		fieldToUpdate = "questions"
	case "👎":
		fieldToUpdate = "downvotes"
	default:
		return nil // Ignore other reactions
	}

	query := fmt.Sprintf("UPDATE recommendations SET %s = %s + ? WHERE id = ?", fieldToUpdate, fieldToUpdate)
	_, err := DB.Exec(query, increment, submissionID)
	return err
}

// MarkSubmissionDeleted marks a submission as deleted (soft delete).
func MarkSubmissionDeleted(submissionID string) error {
	_, err := DB.Exec("UPDATE recommendations SET is_deleted = 1 WHERE id = ?", submissionID)
	return err
}

// GetSubmissionWithDeleted retrieves a submission by its ID, including deleted ones.
func GetSubmissionWithDeleted(submissionID string) (*model.Submission, error) {
	row := DB.QueryRow(`SELECT
		id, author_id, COALESCE(author_nickname, '') as author_nickname, content, post_url, created_at,
		COALESCE(guild_id, '') as guild_id,
		COALESCE(original_title, '') as original_title,
		COALESCE(original_author, '') as original_author,
		COALESCE(recommend_title, '') as recommend_title,
		COALESCE(recommend_content, '') as recommend_content,
		COALESCE(original_post_timestamp, '') as original_post_timestamp,
		COALESCE(final_amway_message_id, '') as final_amway_message_id,
		upvotes, questions, downvotes
	FROM recommendations WHERE id = ?`, submissionID)

	return scanSubmission(row)
}

// IsSubmissionDeleted checks if a submission is marked as deleted.
func IsSubmissionDeleted(submissionID string) (bool, error) {
	var isDeleted int
	err := DB.QueryRow("SELECT is_deleted FROM recommendations WHERE id = ?", submissionID).Scan(&isDeleted)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("submission not found")
		}
		return false, err
	}
	return isDeleted == 1, nil
}
