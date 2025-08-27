package db

import (
	"amway/model"
	"database/sql"
	"time"
)

// GetUserStats retrieves a user's stats from the users table.
func GetUserStats(userID string) (*model.User, error) {
	var user model.User
	err := DB.QueryRow("SELECT user_id, featured_count, rejected_count FROM users WHERE user_id = ?", userID).Scan(&user.UserID, &user.FeaturedCount, &user.RejectedCount)
	if err != nil {
		if err == sql.ErrNoRows {
			// If the user is not in the table, create a new record
			_, err = DB.Exec("INSERT INTO users(user_id, featured_count, rejected_count) VALUES(?, 0, 0)", userID)
			if err != nil {
				return nil, err
			}
			return &model.User{UserID: userID, FeaturedCount: 0, RejectedCount: 0}, nil
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

// BanUser adds a user to the banned_users table.
func BanUser(userID, reason string) error {
	_, err := DB.Exec("INSERT OR REPLACE INTO banned_users(user_id, reason, timestamp) VALUES(?, ?, ?)", userID, reason, time.Now().Unix())
	return err
}

// UnbanUser removes a user from the banned_users table.
func UnbanUser(userID string) error {
	_, err := DB.Exec("DELETE FROM banned_users WHERE user_id = ?", userID)
	return err
}

// BannedUser represents a user in the banned_users table.
type BannedUser struct {
	ID        string
	Reason    string
	Timestamp int64
}

// GetBannedUsers retrieves all users from the banned_users table.
func GetBannedUsers() ([]BannedUser, error) {
	rows, err := DB.Query("SELECT user_id, reason, timestamp FROM banned_users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []BannedUser
	for rows.Next() {
		var u BannedUser
		if err := rows.Scan(&u.ID, &u.Reason, &u.Timestamp); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}
