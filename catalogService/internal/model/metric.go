package model

type Metric struct {
	UptimePercentage float64 `json:"uptime_percentage"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	MinLatencyMs     int     `json:"min_latency_ms"`
	MaxLatencyMs     int     `json:"max_latency_ms"`
	TotalChecks      int64   `json:"total_checks"`
}
