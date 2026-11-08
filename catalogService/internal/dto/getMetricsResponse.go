package dto

import "time"

type GetMetricsResponse struct {
	ResourceId       string    `json:"resource_id"`
	FromPeriod       time.Time `json:"from_period"`
	ToPeriod         time.Time `json:"to_period"`
	Frequency        Frequency `json:"frequency"`
	UptimePercentage float64   `json:"uptime_percentage"`
	AvgLatencyMs     float64   `json:"avg_latency_ms"`
	MinLatencyMs     int       `json:"min_latency_ms"`
	MaxLatencyMs     int       `json:"max_latency_ms"`
}
