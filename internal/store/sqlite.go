package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// Store は SQLite への読み書きを担う。
type Store struct {
	db *sql.DB
}

// New は SQLite ファイルを開き、マイグレーションを実行して Store を返す。
func New(dbPath string, migrationPath string) (*Store, error) {
	// WAL モードで開く（読み書き競合を減らすため）
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_foreign_keys=on", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(migrationPath); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

// migrate は SQL ファイルを読み込んで実行する。
// Issue #1 スコープ: 単一ファイルの単純実行（バージョン管理は MVP 対象外）。
func (s *Store) migrate(migrationPath string) error {
	query, err := os.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}

	if _, err := s.db.Exec(string(query)); err != nil {
		return fmt.Errorf("exec migration: %w", err)
	}

	return nil
}

// Close は DB 接続を閉じる。
func (s *Store) Close() error {
	return s.db.Close()
}

// DB は生の *sql.DB を返す（テスト用）。
func (s *Store) DB() *sql.DB {
	return s.db
}
