package dto

type CreateResourceRequest struct {
	Url              string `json:"url"`
	Name             string `json:"name"`
	CheckIntervalSec int    `json:"check_interval_sec"`
	IsPublic         bool   `json:"is_public"`
}
