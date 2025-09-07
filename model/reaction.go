package model

// SubmissionReaction represents a single user's reaction to a submission.
type SubmissionReaction struct {
	SubmissionID string `json:"submission_id"`
	MessageID    string `json:"message_id"`
	UserID       string `json:"user_id"`
	EmojiName    string `json:"emoji_name"`
	CreatedAt    int64  `json:"created_at"`
}
