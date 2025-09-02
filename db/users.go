package db

import (
	"amway/model"
	"database/sql"
	"time"
)

// GetUserStats retrieves a user's stats from the users table.
func GetUserStats(userID string) (*model.User, error) {
	var user model.User
	err := DB.QueryRow("SELECT user_id, featured_count, rejected_count, ban_count, is_permanently_banned, banned_until FROM users WHERE user_id = ?", userID).Scan(&user.UserID, &user.FeaturedCount, &user.RejectedCount, &user.BanCount, &user.IsPermanentlyBanned, &user.BannedUntil)
	if err != nil {
		if err == sql.ErrNoRows {
			// If the user is not in the table, create a new record
			_, err = DB.Exec("INSERT INTO users(user_id) VALUES(?)", userID)
			if err != nil {
				return nil, err
			}
			// Return a new user struct with default zero values
			return &model.User{UserID: userID}, nil
		}
		return nil, err
	}
	return &user, nil
}

// IncrementFeaturedCount increments the featured_count for a user.
func IncrementFeaturedCount(userID string) error {
	_, err := DB.Exec("INSERT INTO users (user_id, featured_count) VALUES (?, 1) ON CONFLICT(user_id) DO UPDATE SET featured_count = featured_count + 1", userID)
	return err
}

// IncrementRejectedCount increments the rejected_count for a user.
func IncrementRejectedCount(userID string) error {
	_, err := DB.Exec("INSERT INTO users (user_id, rejected_count) VALUES (?, 1) ON CONFLICT(user_id) DO UPDATE SET rejected_count = rejected_count + 1", userID)
	return err
}

// CheckUserBanStatus checks if a user is currently banned.
// It returns two booleans: isBanned (true if banned temporarily or permanently)
// and isPermanent (true if the ban is permanent).
func CheckUserBanStatus(userID string) (isBanned bool, isPermanent bool, err error) {
	user, err := GetUserStats(userID)
	if err != nil {
		return false, false, err
	}

	if user.IsPermanentlyBanned {
		return true, true, nil
	}

	if user.BannedUntil.Valid && user.BannedUntil.Int64 > time.Now().Unix() {
		return true, false, nil
	}

	return false, false, nil
}

// ApplyBan applies a temporary ban to a user and increments their ban counter.
// It returns the user's updated stats.
func ApplyBan(userID string, duration time.Duration) (*model.User, error) {
	// Ensure the user exists in the database.
	_, err := GetUserStats(userID)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(duration).Unix()
	_, err = DB.Exec("UPDATE users SET ban_count = ban_count + 1, banned_until = ? WHERE user_id = ?", expiresAt, userID)
	if err != nil {
		return nil, err
	}

	// Return the updated user object.
	return GetUserStats(userID)
}

// ApplyPermanentBan permanently bans a user.
func ApplyPermanentBan(userID string) error {
	_, err := DB.Exec("UPDATE users SET is_permanently_banned = 1 WHERE user_id = ?", userID)
	return err
}

// LiftBan removes any temporary or permanent bans from a user.
func LiftBan(userID string) error {
	_, err := DB.Exec("UPDATE users SET banned_until = NULL, is_permanently_banned = 0 WHERE user_id = ?", userID)
	return err
}
