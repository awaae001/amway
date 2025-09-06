package model

import (
	"time"
)

// SubmissionData holds the temporary data for a submission.
type SubmissionData struct {
	ChannelID        string
	MessageID        string
	OriginalAuthor   string
	RecommendTitle   string
	RecommendContent string
	ReplyToOriginal  bool
	SubmissionID     string // Added to track the database submission ID
	CreatedAt        time.Time
}
