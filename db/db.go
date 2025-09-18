package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbDriver = "sqlite3"
	dbSource = "./data/amway.db"
)

// DB 是全局数据库连接池
var DB *sql.DB

// InitDB 初始化 SQLite 数据库，如果表不存在则创建它们
func InitDB() {
	var err error
	DB, err = sql.Open(dbDriver, dbSource)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// createTables 在 migrate.go 中定义
	createTables()

	log.Println("Database connection initialized successfully.")
}
