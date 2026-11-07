package model

import "time"

type Check struct {
	ResourceId string    `db:"resource_id"`
	CheckedAt  time.Time `db:"checked_at"`
	StatusCode int       `db:"status_code"`
	LatencyMs  int       `db:"latency_ms"`
	IsUp       bool      `db:"is_up"`
	ErrorMsg   *string   `db:"error_msg"`
}
