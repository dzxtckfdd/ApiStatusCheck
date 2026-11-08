package dto

type GetResourceResponse []ResourceItem

type ResourceItem struct {
	Id               string `json:"id"`
	Url              string `json:"url"`
	Name             string `json:"name"`
	CheckIntervalSec int    `json:"check_interval_sec"`
	IsActive         bool   `json:"is_active"`
}
