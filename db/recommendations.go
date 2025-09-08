package db

import (
	"amway/model"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

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
		&sub.Upvotes, &sub.Questions, &sub.Downvotes, &sub.IsAnonymous, &sub.Status, &sub.VoteFileID, &sub.ThreadMessageID,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil, nil if no submission is found
		}
		return nil, err
	}
	return &sub, nil
}

// AddSubmission adds a new submission to the recommendations table (legacy version).
func AddSubmission(userID, url, title, content, guildID, authorNickname string) (string, error) {
	return AddSubmissionV2(userID, url, title, content, "", "", "", guildID, authorNickname, false)
}

// AddSubmissionV2 adds a new submission with original post info and recommendation content.
func AddSubmissionV2(userID, url, recommendTitle, recommendContent, originalTitle, originalAuthor string, originalPostTimestamp string, guildID string, authorNickname string, isAnonymous bool) (string, error) {
	tx, err := DB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback() // Rollback on error

	newID, err := getNextSubmissionID(tx)
	if err != nil {
		return "", err
	}

	submissionID := fmt.Sprintf("%d", newID)

	// Generate a random 8-character hex ID for the vote file
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	voteFileID := hex.EncodeToString(bytes)

	stmt, err := tx.Prepare(`INSERT INTO recommendations(
		id, author_id, author_nickname, content, post_url, created_at, guild_id,
		original_title, original_author, recommend_title, recommend_content, original_post_timestamp, is_anonymous, vote_file_id
	) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
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
		originalTitle, originalAuthor, recommendTitle, recommendContent, originalPostTimestamp, isAnonymous, voteFileID,
	)
	if err != nil {
		return "", err
	}

	return submissionID, tx.Commit()
}

// UpdateSubmissionStatus updates the status of a submission in recommendations table.
func UpdateSubmissionStatus(submissionID, status string) error {
	return UpdateSubmissionReviewer(submissionID, status, "")
}

// UpdateSubmissionReviewer updates the status and reviewer of a submission.
func UpdateSubmissionReviewer(submissionID, status, reviewerID string) error {
	_, err := DB.Exec("UPDATE recommendations SET status = ?, reviewer_id = ? WHERE id = ?", status, reviewerID, submissionID)
	return err
}

// DeleteSubmission removes a submission from the recommendations table.
func DeleteSubmission(submissionID string) error {
	_, err := DB.Exec("DELETE FROM recommendations WHERE id = ?", submissionID)
	return err
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
		upvotes, questions, downvotes, is_anonymous, status, COALESCE(vote_file_id, '') as vote_file_id,
		COALESCE(thread_message_id, '0') as thread_message_id
	FROM recommendations WHERE id = ? AND is_deleted = 0`, submissionID)

	return scanSubmission(row)
}

// UpdateFinalAmwayMessageID updates the final_amway_message_id for a submission.
func UpdateFinalAmwayMessageID(submissionID, messageID string) error {
	_, err := DB.Exec("UPDATE recommendations SET final_amway_message_id = ? WHERE id = ?", messageID, submissionID)
	return err
}

// UpdateThreadMessageID updates the thread_message_id for a submission.
func UpdateThreadMessageID(submissionID, messageID string) error {
	_, err := DB.Exec("UPDATE recommendations SET thread_message_id = ? WHERE id = ?", messageID, submissionID)
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
		upvotes, questions, downvotes, is_anonymous, status, COALESCE(vote_file_id, '') as vote_file_id,
		COALESCE(thread_message_id, '0') as thread_message_id
	FROM recommendations WHERE final_amway_message_id = ? AND is_deleted = 0`, messageID)

	return scanSubmission(row)
}

// UpdateReactionCount updates the reaction counts for a submission.
func UpdateReactionCount(submissionID string, emojiName string, increment int) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := UpdateReactionCountInTx(tx, submissionID, emojiName, increment); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateReactionCountInTx updates the reaction counts for a submission within a transaction.
func UpdateReactionCountInTx(tx *sql.Tx, submissionID string, emojiName string, increment int) error {
	var fieldToUpdate string
	switch emojiName {
	case "👍":
		fieldToUpdate = "upvotes"
	case "🤔":
		fieldToUpdate = "questions"
	case "🚫":
		fieldToUpdate = "downvotes"
	default:
		return nil // Not a trackable emoji
	}

	query := fmt.Sprintf("UPDATE recommendations SET %s = %s + ? WHERE id = ?", fieldToUpdate, fieldToUpdate)
	_, err := tx.Exec(query, increment, submissionID)
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
		upvotes, questions, downvotes, is_anonymous, status, COALESCE(vote_file_id, '') as vote_file_id,
		COALESCE(thread_message_id, '0') as thread_message_id
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

// GetSubmissionsByAuthor retrieves all submissions by a specific author in a guild (excludes deleted).
func GetSubmissionsByAuthor(authorID string, guildID string) ([]*model.Submission, error) {
	query := `SELECT
		id, author_id, COALESCE(author_nickname, '') as author_nickname, content, post_url, created_at,
		COALESCE(guild_id, '') as guild_id,
		COALESCE(original_title, '') as original_title,
		COALESCE(original_author, '') as original_author,
		COALESCE(recommend_title, '') as recommend_title,
		COALESCE(recommend_content, '') as recommend_content,
		COALESCE(original_post_timestamp, '') as original_post_timestamp,
		COALESCE(final_amway_message_id, '') as final_amway_message_id,
		upvotes, questions, downvotes, is_anonymous, status, COALESCE(vote_file_id, '') as vote_file_id,
		COALESCE(thread_message_id, '0') as thread_message_id
	FROM recommendations WHERE author_id = ? AND guild_id = ? AND is_deleted = 0 ORDER BY created_at DESC`

	rows, err := DB.Query(query, authorID, guildID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []*model.Submission
	for rows.Next() {
		submission, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		if submission != nil {
			submissions = append(submissions, submission)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return submissions, nil
}

// GetAllSubmissionsByAuthor retrieves all submissions by a specific author (excludes deleted).
func GetAllSubmissionsByAuthor(authorID string) ([]*model.Submission, error) {
	query := `SELECT
		id, author_id, COALESCE(author_nickname, '') as author_nickname, content, post_url, created_at,
		COALESCE(guild_id, '') as guild_id,
		COALESCE(original_title, '') as original_title,
		COALESCE(original_author, '') as original_author,
		COALESCE(recommend_title, '') as recommend_title,
		COALESCE(recommend_content, '') as recommend_content,
		COALESCE(original_post_timestamp, '') as original_post_timestamp,
		COALESCE(final_amway_message_id, '') as final_amway_message_id,
		upvotes, questions, downvotes, is_anonymous, status, COALESCE(vote_file_id, '') as vote_file_id,
		COALESCE(thread_message_id, '0') as thread_message_id
	FROM recommendations WHERE author_id = ? AND is_deleted = 0 ORDER BY created_at DESC`

	rows, err := DB.Query(query, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []*model.Submission
	for rows.Next() {
		submission, err := scanSubmission(rows)
		if err != nil {
			return nil, err
		}
		if submission != nil {
			submissions = append(submissions, submission)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return submissions, nil
}
