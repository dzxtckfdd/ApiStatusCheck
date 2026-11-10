package service

import (
	"apiChecker/internal/dto"
	"io"

	"context"
	"net/http"
	"time"
)

type Checker struct {
	client *http.Client
}

func NewChecker(timeout time.Duration) *Checker {
	return &Checker{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

func (c *Checker) Check(ctx context.Context, res dto.ResourceItem) *dto.CheckItem {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, res.Url, nil)
	if err != nil {
		errMsg := err.Error()
		return &dto.CheckItem{
			ResourceId: res.Id,
			CheckedAt:  start,
			StatusCode: 0,
			LatencyMs:  0,
			IsUp:       false,
			ErrorMsg:   &errMsg,
		}
	}

	req.Header.Set("User-Agent", "ApiStatusCheck-Bot/1.0")

	resp, err := c.client.Do(req)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		errMsg := err.Error()
		return &dto.CheckItem{
			ResourceId: res.Id,
			CheckedAt:  start,
			StatusCode: 0,
			LatencyMs:  latency,
			IsUp:       false,
			ErrorMsg:   &errMsg,
		}
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	isUp := resp.StatusCode >= 200 && resp.StatusCode < 400

	var errorMsg *string

	if !isUp {
		msg := resp.Status
		errorMsg = &msg
	}

	return &dto.CheckItem{
		ResourceId: res.Id,
		CheckedAt:  start,
		StatusCode: resp.StatusCode,
		LatencyMs:  latency,
		IsUp:       isUp,
		ErrorMsg:   errorMsg,
	}
}
