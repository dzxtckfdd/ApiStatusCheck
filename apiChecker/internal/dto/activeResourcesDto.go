package dto

import (
	"time"
)

type ResourcesResponce []CheckItem

type ResourceItem struct {
	Id               string     `json:"id"`
	Url              string     `json:"url"`
	Name             string     `json:"name"`
	CheckIntervalSec int        `json:"check_interval_sec"`
	IsActive         bool       `json:"is_active"`
	IsUp             bool       `json:"is_up"`
	LastCheckedAt    *time.Time `json:"last_checked_at"`
}
