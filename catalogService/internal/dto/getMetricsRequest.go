package dto

import "time"

type Frequency string

const (
	FrequencyMinutely Frequency = "minutely"
	FrequencyHourly   Frequency = "hourly"
	FrequencyDaily    Frequency = "daily"
)

type GetMetricsRequest struct {
	ResourceId string    `json:"resource_id"`
	FromPeriod time.Time `json:"from_period"`
	ToPeriod   time.Time `json:"to_period"`
	Frequency  Frequency `json:"frequency"`
}
