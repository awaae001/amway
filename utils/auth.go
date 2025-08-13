package utils

import (
	"amway/config"
	"slices"
)

// CheckAuth 检查用户是否有权限
func CheckAuth(userID string, roles []string) bool {
	authConfig := config.Cfg.Commands.Auth

	// 检查是否为开发者
	if slices.Contains(authConfig.Developers, userID) {
		return true
	}

	// 检查是否拥有管理员角色
	for _, role := range roles {
		if slices.Contains(authConfig.AdminsRoles, role) {
			return true
		}
	}

	return false
}
