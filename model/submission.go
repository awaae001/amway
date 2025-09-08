package model

// Submission represents a submission record from recommendations table.
type Submission struct {
	ID                    string
	UserID                string
	URL                   string
	Content               string
	Timestamp             int64
	OriginalTitle         string
	OriginalAuthor        string
	OriginalPostTimestamp string
	RecommendTitle        string
	RecommendContent      string
	GuildID               string
	AuthorNickname        string
	FinalAmwayMessageID   string
	Upvotes               int
	Questions             int
	Downvotes             int
	IsAnonymous           bool
	Status                string
	VoteFileID            string
	ThreadMessageID       string
}
