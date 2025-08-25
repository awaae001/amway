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
