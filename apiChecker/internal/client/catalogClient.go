package client

import (
	"apiChecker/internal/config"
	"apiChecker/internal/dto"
	"context"
	"strings"

	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type CatalogClient struct {
	catalogBaseURL string
	secretKey      string
	httpClient     *http.Client
}

func NewCatalogClient(catalog config.CatalogConfig, internal config.InternalConfig) *CatalogClient {
	return &CatalogClient{
		catalogBaseURL: strings.TrimRight(catalog.CatalogUrl, "/"),
		secretKey:      internal.SecretKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (cc *CatalogClient) GetActiveResources(ctx context.Context) (*dto.ResourcesResponce, error) {
	reqURL := cc.catalogBaseURL + "/internal/resources"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Internal-Secret", cc.secretKey)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request catalog service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog service returned unexpected status: %s", resp.Status)
	}

	var resources dto.ResourcesResponce

	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &resources, nil
}

func (cc *CatalogClient) SendBatchChecks(ctx context.Context, batch *dto.SaveBatchRequest) error {
	if batch == nil || len(batch.Results) == 0 {
		return nil
	}

	reqURL := cc.catalogBaseURL + "/internal/checks"

	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", cc.secretKey)

	resp, err := cc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send batch to catalog service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("catalog service rejected batch with status: %s", resp.Status)
	}
	return nil
}
