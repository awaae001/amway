package db

import (
	"amway/model"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// rowScanner 是一个接口，可以由 *sql.Row 或 *sql.Rows 来满足。
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanSubmission 将一行扫描到 Submission 结构体中。
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
			return nil, nil // 如果未找到投稿，则返回 nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

// AddSubmission 将新投稿添加到 recommendations 表中（旧版）。
func AddSubmission(userID, url, title, content, guildID, authorNickname string) (string, error) {
	return AddSubmissionV2(userID, url, title, content, "", "", "", guildID, authorNickname, false)
}

// AddSubmissionV2 使用原始帖子信息和推荐内容添加新投稿。
func AddSubmissionV2(userID, url, recommendTitle, recommendContent, originalTitle, originalAuthor string, originalPostTimestamp string, guildID string, authorNickname string, isAnonymous bool) (string, error) {
	tx, err := DB.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback() // 出错时回滚

	newID, err := getNextSubmissionID(tx)
	if err != nil {
		return "", err
	}

	submissionID := fmt.Sprintf("%d", newID)

	// 为投票文件生成一个随机的8个字符的十六进制ID
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

// UpdateSubmissionStatus 更新 recommendations 表中投稿的状态。
func UpdateSubmissionStatus(submissionID, status string) error {
	return UpdateSubmissionReviewer(submissionID, status, "")
}

// UpdateSubmissionReviewer 更新投稿的状态和审核员。
func UpdateSubmissionReviewer(submissionID, status, reviewerID string) error {
	_, err := DB.Exec("UPDATE recommendations SET status = ?, reviewer_id = ? WHERE id = ?", status, reviewerID, submissionID)
	return err
}

// DeleteSubmission 从 recommendations 表中删除一个投稿。
func DeleteSubmission(submissionID string) error {
	_, err := DB.Exec("DELETE FROM recommendations WHERE id = ?", submissionID)
	return err
}

// GetSubmission 从 recommendations 表中按 ID 检索投稿（不包括已删除的）。
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

// UpdateFinalAmwayMessageID 更新投稿的 final_amway_message_id。
func UpdateFinalAmwayMessageID(submissionID, messageID string) error {
	_, err := DB.Exec("UPDATE recommendations SET final_amway_message_id = ? WHERE id = ?", messageID, submissionID)
	return err
}

// UpdateThreadMessageID 更新投稿的 thread_message_id。
func UpdateThreadMessageID(submissionID, messageID string) error {
	_, err := DB.Exec("UPDATE recommendations SET thread_message_id = ? WHERE id = ?", messageID, submissionID)
	return err
}

// GetSubmissionByMessageID 按最终消息 ID 检索投稿（不包括已删除的）。
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

// UpdateReactionCount 更新投稿的反应计数。
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

// UpdateReactionCountInTx 在事务中更新投稿的反应计数。
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
		return nil // 不是可追踪的表情符号
	}

	query := fmt.Sprintf("UPDATE recommendations SET %s = %s + ? WHERE id = ?", fieldToUpdate, fieldToUpdate)
	_, err := tx.Exec(query, increment, submissionID)
	return err
}

// MarkSubmissionDeleted 将投稿标记为已删除（软删除）。
func MarkSubmissionDeleted(submissionID string) error {
	_, err := DB.Exec("UPDATE recommendations SET is_deleted = 1 WHERE id = ?", submissionID)
	return err
}

// GetSubmissionWithDeleted 按 ID 检索投稿，包括已删除的投稿。
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

// IsSubmissionDeleted 检查投稿是否被标记为已删除。
func IsSubmissionDeleted(submissionID string) (bool, error) {
	var isDeleted int
	err := DB.QueryRow("SELECT is_deleted FROM recommendations WHERE id = ?", submissionID).Scan(&isDeleted)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("未找到投稿")
		}
		return false, err
	}
	return isDeleted == 1, nil
}

// GetSubmissionsByAuthor 检索特定作者在服务器中的所有投稿（不包括已删除的）。
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

// GetAllSubmissionsByAuthor 检索特定作者的所有投稿（不包括已删除的）。
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
