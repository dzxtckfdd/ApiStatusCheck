package service

import (
	"bytes"
	"catalogService/internal/dto"
	"catalogService/internal/model"
	"catalogService/internal/repository"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"context"
)

type InternalService struct {
	httpClient              *http.Client
	checkRepository         *repository.CheckRepository
	stateRepository         *repository.StateRepository
	incidentRepository      *repository.IncidentRepository
	resourceRepository      *repository.ResourceRepository
	subscritprionRepository *repository.SubscriptionRepository
	notifierUrl             string
	openThreshold           int
	closeThreshold          int
}

func NewInternalService(
	checkRepository *repository.CheckRepository,
	stateRepository *repository.StateRepository,
	incidentRepository *repository.IncidentRepository,
	resourceRepository *repository.ResourceRepository,
	subscritprionRepository *repository.SubscriptionRepository,
	notifierUrl string,
	openThreshold int,
	closeThreshold int,
) *InternalService {
	return &InternalService{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		checkRepository:         checkRepository,
		stateRepository:         stateRepository,
		incidentRepository:      incidentRepository,
		resourceRepository:      resourceRepository,
		subscritprionRepository: subscritprionRepository,
		notifierUrl:             notifierUrl,
		openThreshold:           openThreshold,
		closeThreshold:          closeThreshold,
	}
}

func (is *InternalService) GetActiveResources(ctx context.Context) (*dto.GetResourceResponse, error) {
	res, err := is.resourceRepository.GetActiveResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get active targets: %w", err)
	}

	response := make(dto.GetResourceResponse, 0, len(res))

	for _, t := range res {
		response = append(response, dto.ResourceItem{
			Id:               t.Id,
			Url:              t.Url,
			Name:             t.Name,
			CheckIntervalSec: t.CheckIntervalSec,
			IsActive:         t.IsActive,
			IsUp:             t.IsUp,
			LastCheckedAt:    t.LastCheckedAt,
		})
	}

	return &response, nil
}

func (is *InternalService) ProcessBatch(ctx context.Context, req dto.SaveBatchRequest) error {
	if len(req.Results) == 0 {
		return nil
	}

	models := make([]*model.Check, 0, len(req.Results))

	for _, r := range req.Results {
		models = append(models, &model.Check{
			ResourceId: r.ResourceId,
			CheckedAt:  r.CheckedAt,
			StatusCode: r.StatusCode,
			LatencyMs:  r.LatencyMs,
			IsUp:       r.IsUp,
			ErrorMsg:   r.ErrorMsg,
		})
	}

	if err := is.checkRepository.SaveBatch(ctx, models); err != nil {
		return fmt.Errorf("unable to save checks batch: %w", err)
	}

	for _, c := range req.Results {
		if err := is.processCheck(ctx, c); err != nil {
			fmt.Printf("failed to process check for resource %s: %v\n", c.ResourceId, err)
		}
	}

	return nil
}

func (is *InternalService) processCheck(ctx context.Context, c dto.CheckItem) error {
	state, err := is.stateRepository.GetById(ctx, c.ResourceId)
	if err != nil {
		if errors.Is(err, repository.ErrStateNotFound) {
			state = &model.State{
				ResourceId: c.ResourceId,
				IsUp:       true,
			}
		} else {
			return err
		}
	}

	state.LastCheckedAt = &c.CheckedAt
	if !c.IsUp {
		state.ConsecutiveFailures++
		state.ConsecutiveSuccesses = 0

		if state.ConsecutiveFailures >= is.openThreshold && state.IsUp {
			state.IsUp = false

			incId, err := is.incidentRepository.Open(ctx, c.ResourceId, "Service unreachable")
			if err == nil {
				state.CurrentIncidentId = &incId
			}

			go is.sendNotification(c.ResourceId, "DOWN")
		}
	} else {
		state.ConsecutiveSuccesses++
		state.ConsecutiveFailures = 0
		if state.ConsecutiveSuccesses >= is.closeThreshold && !state.IsUp {
			state.IsUp = true
			if state.CurrentIncidentId != nil {
				_ = is.incidentRepository.Resolve(ctx, *state.CurrentIncidentId)
				state.CurrentIncidentId = nil
			}
			// Уведомляем NotifierService
			go is.sendNotification(c.ResourceId, "UP")
		}
	}
	return is.stateRepository.UpsertById(ctx, state)
}

func (is *InternalService) sendNotification(resourceId, status string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	subscribers, err := is.subscritprionRepository.GetSubscribers(ctx, resourceId)
	if err != nil || len(subscribers) == 0 {
		return
	}

	subscriberIds := make([]string, 0, len(subscribers))

	for _, s := range subscribers {
		subscriberIds = append(subscriberIds, s.Id)
	}

	res, err := is.resourceRepository.GetById(ctx, resourceId)
	if err != nil {
		return
	}

	payload := map[string]any{
		"resource_id":    resourceId,
		"resource_name":  res.Name,
		"resource_url":   res.Url,
		"status":         status,
		"subscriber_ids": subscriberIds,
		"occurred_at":    time.Now().UTC(),
	}

	data, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		is.notifierUrl+"/internal/notify",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := is.httpClient.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
}
