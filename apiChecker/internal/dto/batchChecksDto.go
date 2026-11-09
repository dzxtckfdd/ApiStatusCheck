package dto

import (
	"time"
)

type CheckItem struct {
	ResourceId string    `json:"resource_id"`
	CheckedAt  time.Time `json:"checked_at"`
	StatusCode int       `json:"status_code"`
	LatencyMs  int       `json:"latency_ms"`
	IsUp       bool      `json:"is_up"`
	ErrorMsg   *string   `json:"error_msg"`
}

type SaveBatchResponce struct {
	Results []CheckItem `json:"results"`
}
