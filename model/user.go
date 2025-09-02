package model

import "database/sql"

// User represents a user's stats.
type User struct {
	UserID              string
	FeaturedCount       int
	RejectedCount       int
	BanCount            int
	IsPermanentlyBanned bool
	BannedUntil         sql.NullInt64
}
