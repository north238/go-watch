package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gowatch/internal/model"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mattn/go-sqlite3"
)

// Store は SQLite への読み書きを担う。
type Store struct {
	db *sql.DB
}

var ErrorDuplicateURL = errors.New("duplicate url")

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

// CRUDメソッド
// 作成
func (s *Store) AddTarget(ctx context.Context, url string, name string) (model.Target, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO targets (id, url, name, interval_sec, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id,
		url,
		name,
		30,
		model.StatusUnknown,
		now,
		now,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return model.Target{}, fmt.Errorf("duplicate: %w", ErrorDuplicateURL)
		}
		return model.Target{}, fmt.Errorf("failed to add target: %w", err)
	}

	return model.Target{
		ID:          id,
		URL:         url,
		Name:        name,
		IntervalSec: 30,
		Status:      model.StatusUnknown,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// 抽出
func (s *Store) ListTargets(ctx context.Context) ([]model.Target, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, url, name, interval_sec, status, created_at, updated_at FROM targets")
	if err != nil {
		return nil, fmt.Errorf("failed to list targets: %w", err)
	}
	defer rows.Close()

	targets := make([]model.Target, 0)
	for rows.Next() {
		var t model.Target
		err := rows.Scan(&t.ID, &t.URL, &t.Name, &t.IntervalSec, &t.Status, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to list targets scan: %w", err)
		}
		targets = append(targets, t)
	}

	return targets, nil
}

// 削除
func (s *Store) DeleteTarget(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM targets WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete target: %w", err)
	}

	return nil
}

// チェック結果を保存
func (s *Store) SaveCheckResult(ctx context.Context, result model.CheckResult) error {
	_, err := s.db.ExecContext(
		ctx,
		"INSERT INTO check_results (target_id, status, status_code, response_time_ms, error, checked_at) VALUES (?, ?, ?, ?, ?, ?)",
		result.TargetID,
		result.Status,
		result.StatusCode,
		result.ResponseTimeMs,
		result.Error,
		result.CheckedAt,
	)
	if err != nil {
		return fmt.Errorf("save check result: %w", err)
	}

	return nil
}

// テーブルのステータスを更新
func (s *Store) UpdateTargetStatus(ctx context.Context, targetId string, status model.Status) error {
	_, err := s.db.ExecContext(
		ctx,
		"UPDATE targets SET status = ?, updated_at = ? WHERE id = ?",
		status,
		time.Now(),
		targetId,
	)
	if err != nil {
		return fmt.Errorf("update target status: %w", err)
	}

	return nil
}

// 直近の1,000件のみ残し古いものを削除
func (s *Store) DeleteOldCheckResults(ctx context.Context, targetId string) error {
	_, err := s.db.ExecContext(
		ctx,
		"DELETE FROM check_results WHERE target_id = ? AND id NOT IN (SELECT id FROM check_results WHERE target_id = ? ORDER BY checked_at DESC LIMIT 1000)",
		targetId,
		targetId,
	)
	if err != nil {
		return fmt.Errorf("delete old check results: %w", err)
	}

	return nil
}
