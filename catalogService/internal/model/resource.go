package model

import "time"

type Resource struct {
	Id               string     `db:"id"`
	Url              string     `db:"url"`
	Name             string     `db:"name"`
	CheckIntervalSec int        `db:"check_interval_sec"`
	IsActive         bool       `db:"is_active"`
	IsPublic         bool       `db:"is_public"`
	CreatedAt        *time.Time `db:"created_at"`
	IsUp             bool       `db:"is_up"`
	LastCheckedAt    *time.Time `db:"last_checked_at"`
}
