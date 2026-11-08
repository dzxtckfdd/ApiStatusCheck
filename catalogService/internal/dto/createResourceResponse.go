package dto

type CreateResourceResponse struct {
	Id               string `json:"id"`
	Url              string `json:"url"`
	Name             string `json:"name"`
	CheckIntervalSec int    `json:"check_interval_sec"`
	IsActive         bool   `json:"is_active"`
	IsPublic         bool   `json:"is_public"`
}
