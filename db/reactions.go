package db

import (
	"amway/model"
	"database/sql"
)

// GetReaction retrieves a user's reaction for a specific submission.
func GetReaction(submissionID, userID string) (*model.SubmissionReaction, error) {
	row := DB.QueryRow(`
		SELECT submission_id, message_id, user_id, emoji_name, created_at
		FROM submission_reactions
		WHERE submission_id = ? AND user_id = ?
	`, submissionID, userID)

	var reaction model.SubmissionReaction
	err := row.Scan(&reaction.SubmissionID, &reaction.MessageID, &reaction.UserID, &reaction.EmojiName, &reaction.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No reaction found is not an error
		}
		return nil, err
	}
	return &reaction, nil
}

// UpsertReaction inserts a new reaction or updates an existing one.
func UpsertReaction(reaction *model.SubmissionReaction) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := UpsertReactionInTx(tx, reaction); err != nil {
		return err
	}

	return tx.Commit()
}

// UpsertReactionInTx inserts a new reaction or updates an existing one within a transaction.
func UpsertReactionInTx(tx *sql.Tx, reaction *model.SubmissionReaction) error {
	query := `
		INSERT INTO submission_reactions (submission_id, message_id, user_id, emoji_name, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(submission_id, user_id) DO UPDATE SET
		emoji_name = excluded.emoji_name,
		created_at = excluded.created_at;
	`
	_, err := tx.Exec(query, reaction.SubmissionID, reaction.MessageID, reaction.UserID, reaction.EmojiName, reaction.CreatedAt)
	return err
}

// DeleteReaction removes a user's reaction from a submission.
func DeleteReaction(submissionID, userID string) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := DeleteReactionInTx(tx, submissionID, userID); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteReactionInTx removes a user's reaction from a submission within a transaction.
func DeleteReactionInTx(tx *sql.Tx, submissionID, userID string) error {
	_, err := tx.Exec(`
		DELETE FROM submission_reactions
		WHERE submission_id = ? AND user_id = ?
	`, submissionID, userID)
	return err
}
