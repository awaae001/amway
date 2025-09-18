package db

import (
	"log"
)

// createTables 如果数据库中不存在必要的表，则创建它们
func createTables() {
	// 用于创建 'recommendations' 表的 SQL 语句
	createRecommendationsTableSQL := `
	CREATE TABLE IF NOT EXISTS recommendations (
		id TEXT PRIMARY KEY,
		author_id TEXT NOT NULL,
		author_nickname TEXT,
		content TEXT NOT NULL,
		post_url TEXT,
		upvotes INTEGER NOT NULL DEFAULT 0,
		questions INTEGER NOT NULL DEFAULT 0,
		downvotes INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		reviewer_id TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		guild_id TEXT,
		original_title TEXT,
		original_author TEXT,
		recommend_title TEXT,
		recommend_content TEXT,
		original_post_timestamp TEXT,
		final_amway_message_id TEXT,
		is_deleted INTEGER NOT NULL DEFAULT 0,
		is_anonymous INTEGER NOT NULL DEFAULT 0,
		vote_file_id TEXT,
		thread_message_id TEXT NOT NULL DEFAULT '0'
	);`

	_, err := DB.Exec(createRecommendationsTableSQL)
	if err != nil {
		log.Fatalf("Failed to create recommendations table: %v", err)
	}

	// 用于创建 'users' 表的 SQL 语句
	createUsersTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		user_id TEXT PRIMARY KEY,
		featured_count INTEGER NOT NULL DEFAULT 0,
		rejected_count INTEGER NOT NULL DEFAULT 0
	);`

	_, err = DB.Exec(createUsersTableSQL)
	if err != nil {
		log.Fatalf("Failed to create users table: %v", err)
	}

	// 用于顺序 ID 生成的 'id_counter' 表的 SQL 语句
	createIdCounterTableSQL := `
	CREATE TABLE IF NOT EXISTS id_counter (
		counter_name TEXT PRIMARY KEY,
		current_value INTEGER NOT NULL DEFAULT 0
	);`

	_, err = DB.Exec(createIdCounterTableSQL)
	if err != nil {
		log.Fatalf("Failed to create id_counter table: %v", err)
	}

	// 用于创建 'submission_reactions' 表的 SQL 语句
	createSubmissionReactionsTableSQL := `
	CREATE TABLE IF NOT EXISTS submission_reactions (
		submission_id TEXT NOT NULL,
		message_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		emoji_name TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		PRIMARY KEY (submission_id, user_id)
	);`

	_, err = DB.Exec(createSubmissionReactionsTableSQL)
	if err != nil {
		log.Fatalf("Failed to create submission_reactions table: %v", err)
	}

	log.Println("Database tables initialized successfully.")
}
