package db

import (
	"amway/model"
	"database/sql"
	"time"
)

// GetUserStats 从 users 表中检索用户的统计数据
func GetUserStats(userID string) (*model.User, error) {
	var user model.User
	err := DB.QueryRow("SELECT user_id, featured_count, rejected_count, ban_count, is_permanently_banned, banned_until FROM users WHERE user_id = ?", userID).Scan(&user.UserID, &user.FeaturedCount, &user.RejectedCount, &user.BanCount, &user.IsPermanentlyBanned, &user.BannedUntil)
	if err != nil {
		if err == sql.ErrNoRows {
			// 如果用户不在表中，则创建新记录
			_, err = DB.Exec("INSERT INTO users(user_id) VALUES(?)", userID)
			if err != nil {
				return nil, err
			}
			// 返回具有默认零值的新用户结构
			return &model.User{UserID: userID}, nil
		}
		return nil, err
	}
	return &user, nil
}

// IncrementFeaturedCount 增加用户的 featured_count
func IncrementFeaturedCount(userID string) error {
	_, err := DB.Exec("INSERT INTO users (user_id, featured_count) VALUES (?, 1) ON CONFLICT(user_id) DO UPDATE SET featured_count = featured_count + 1", userID)
	return err
}

// IncrementRejectedCount 增加用户的 rejected_count
func IncrementRejectedCount(userID string) error {
	_, err := DB.Exec("INSERT INTO users (user_id, rejected_count) VALUES (?, 1) ON CONFLICT(user_id) DO UPDATE SET rejected_count = rejected_count + 1", userID)
	return err
}

// CheckUserBanStatus 检查用户当前是否被封禁
// 它返回两个布尔值：isBanned（如果用户被临时或永久封禁，则为 true）
// 和 isPermanent（如果封禁是永久性的，则为 true）
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

// ApplyBan 对用户应用临时封禁并增加其封禁计数器
// 它返回用户更新后的统计数据
func ApplyBan(userID string, duration time.Duration) (*model.User, error) {
	// 确保用户存在于数据库中
	_, err := GetUserStats(userID)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(duration).Unix()
	_, err = DB.Exec("UPDATE users SET ban_count = ban_count + 1, banned_until = ? WHERE user_id = ?", expiresAt, userID)
	if err != nil {
		return nil, err
	}

	// 返回更新后的用户对象
	return GetUserStats(userID)
}

// ApplyPermanentBan 永久封禁用户
func ApplyPermanentBan(userID string) error {
	_, err := DB.Exec("UPDATE users SET is_permanently_banned = 1 WHERE user_id = ?", userID)
	return err
}

// LiftBan 解除用户的任何临时或永久封禁
func LiftBan(userID string) error {
	_, err := DB.Exec("UPDATE users SET banned_until = NULL, is_permanently_banned = 0 WHERE user_id = ?", userID)
	return err
}
