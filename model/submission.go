package model

// Submission represents a submission record from recommendations table.
type Submission struct {
	ID        string
	UserID    string
	URL       string
	Content   string
	Timestamp int64
}
