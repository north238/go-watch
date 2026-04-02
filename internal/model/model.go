package model

import "time"

// Status はヘルスチェック結果のステータスを表す。
type Status string

const (
	StatusUp      Status = "up"
	StatusDown    Status = "down"
	StatusSlow    Status = "slow"
	StatusUnknown Status = "unknown"
)

// Target は監視対象 URL を表す。
type Target struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Name        string    `json:"name"`
	IntervalSec int       `json:"interval_sec"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CheckResult は 1 回のヘルスチェック結果を表す。
type CheckResult struct {
	ID             int64     `json:"id"`
	TargetID       string    `json:"target_id"`
	Status         Status    `json:"status"`
	StatusCode     int       `json:"status_code"`
	ResponseTimeMs int64     `json:"response_time_ms"`
	Error          string    `json:"error,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
}

type WSMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type CycleStart struct {
	TargetCount int       `json:"target_count"`
	StartedAt   time.Time `json:"started_at"`
}

type CycleComplete struct {
	Total       int       `json:"total"`
	Up          int       `json:"up"`
	Down        int       `json:"down"`
	Slow        int       `json:"slow"`
	DurationMs  int64     `json:"duration_ms"`
	CompletedAt time.Time `json:"completed_at"`
}
