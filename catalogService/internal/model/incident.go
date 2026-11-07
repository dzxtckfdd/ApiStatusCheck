package model

import "time"

type Incident struct {
	Id          string     `db:"id"`
	ResourceId  string     `db:"resource_id"`
	StartedAt   time.Time  `db:"started_at"`
	ResolvedAt  *time.Time `db:"resolved_at"`
	Description *string    `db:"description"`
}
