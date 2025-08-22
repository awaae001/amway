package db

import (
	"database/sql"
	"time"
)

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
