package model

import "time"

type State struct {
	ResourceId           string    `db:"resource_id"`
	IsUp                 bool      `db:"is_up"`
	ConsecutiveFailures  int       `db:"consecutive_failures"`
	ConsecutiveSuccesses int       `db:"consecutive_successes"`
	CurrentIncidentId    *string    `db:"current_incident_id"`
	LastCheckedAt        *time.Time `db:"last_checked_at"`
}
