package service

import (
	"catalogService/internal/dto"
	"catalogService/internal/repository"
	"errors"
	"fmt"
	"time"

	"context"
	"net/url"
)

var ErrInvalidURL error = errors.New("invalid target url")
var ErrAccessDenied = errors.New("access denied to target resource")

type ResourceService struct {
	resourceRepository     *repository.ResourceRepository
	subscriptionRepository *repository.SubscriptionRepository
	metricRepository       *repository.MetricRepository
	stateRepository        *repository.StateRepository
}

func NewResourceService(
	resourceRepository *repository.ResourceRepository,
	subscriptionRepository *repository.SubscriptionRepository,
	metricRepository *repository.MetricRepository,
) *ResourceService {
	return &ResourceService{
		resourceRepository:     resourceRepository,
		subscriptionRepository: subscriptionRepository,
		metricRepository:       metricRepository,
	}
}

func ValidateCreateResourceRequest(req *dto.CreateResourceRequest) error {
	if len(req.Url) == 0 {
		return ErrInvalidURL
	}
	if _, err := url.Parse(req.Url); err != nil {
		return ErrInvalidURL
	}
	if req.CheckIntervalSec <= 0 {
		req.CheckIntervalSec = 60
	}
	if req.CheckIntervalSec < 10 {
		req.CheckIntervalSec = 10
	}

	return nil
}

func (rs *ResourceService) CreateResource(ctx context.Context, req dto.CreateResourceRequest, userId string) (*dto.CreateResourceResponse, error) {
	if err := ValidateCreateResourceRequest(&req); err != nil {
		return nil, err
	}

	existing, err := rs.resourceRepository.GetByUrl(ctx, req.Url)
	if err == nil && existing != nil {
		if err := rs.subscriptionRepository.LinkUserById(ctx, userId, existing.Id); err != nil {
			return nil, fmt.Errorf("unable to link user with existing resource: %w", err)
		}
		return &dto.CreateResourceResponse{
			Id:               existing.Id,
			Url:              existing.Url,
			Name:             existing.Name,
			CheckIntervalSec: existing.CheckIntervalSec,
			IsActive:         existing.IsActive,
		}, nil
	}

	resourceId, err := rs.resourceRepository.Create(ctx, req.Url, req.Name, req.CheckIntervalSec, req.IsPublic)
	if err != nil {
		return nil, fmt.Errorf("unable to create resource: %w", err)
	}

	err = rs.subscriptionRepository.LinkUserById(ctx, userId, resourceId)
	if err != nil {
		return nil, fmt.Errorf("unable to link user with resource: %w", err)
	}

	return &dto.CreateResourceResponse{
		Id:               resourceId,
		Url:              req.Url,
		Name:             req.Name,
		CheckIntervalSec: req.CheckIntervalSec,
		IsActive:         true,
	}, nil
}

func (rs *ResourceService) GetResourcesByUserId(ctx context.Context, userId string) (*dto.GetResourceResponse, error) {
	items, err := rs.resourceRepository.GetByUserId(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("unable to get user resources: %w", err)
	}

	response := make(dto.GetResourceResponse, 0, len(items))

	for _, item := range items {
		response = append(response, dto.ResourceItem{
			Id:               item.Id,
			Url:              item.Url,
			Name:             item.Name,
			CheckIntervalSec: item.CheckIntervalSec,
			IsActive:         item.IsActive,
		})
	}

	return &response, nil
}

func (rs *ResourceService) GetPublicResources(ctx context.Context) (*dto.GetResourceResponse, error) {
	items, err := rs.resourceRepository.GetPublic(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get public resources: %w", err)
	}

	response := make(dto.GetResourceResponse, 0, len(items))

	for _, item := range items {
		response = append(response, dto.ResourceItem{
			Id:               item.Id,
			Url:              item.Url,
			Name:             item.Name,
			CheckIntervalSec: item.CheckIntervalSec,
			IsActive:         item.IsActive,
		})
	}

	return &response, nil
}

func (rs *ResourceService) GetMetricsById(ctx context.Context, req dto.GetMetricsRequest, userId string) (*dto.GetMetricsResponse, error) {
	resource, err := rs.resourceRepository.GetById(ctx, req.ResourceId)
	if err != nil {
		return nil, err
	}

	if !resource.IsPublic {
		hasAccess, err := rs.subscriptionRepository.HasAccess(ctx, userId, req.ResourceId)
		if err != nil {
			return nil, fmt.Errorf("unable to check access: %w", err)
		}
		if !hasAccess {
			return nil, ErrAccessDenied
		}
	}

	if req.ToPeriod.IsZero() {
		req.ToPeriod = time.Now()
	}
	if req.FromPeriod.IsZero() {
		req.FromPeriod = req.ToPeriod.Add(-24 * time.Hour)
	}

	var resolution repository.AggregateResolution

	switch req.Frequency {
	case dto.FrequencyDaily:
		resolution = repository.ResolutionDaily
	default:
		req.Frequency = dto.FrequencyHourly
		resolution = repository.ResolutionHourly
	}

	metrics, err := rs.metricRepository.GetSummary(ctx, req.ResourceId, resolution, req.FromPeriod, req.ToPeriod)
	if err != nil {
		return nil, fmt.Errorf("unable to get metrics: %w", err)
	}

	return &dto.GetMetricsResponse{
		ResourceId:       req.ResourceId,
		FromPeriod:       req.FromPeriod,
		ToPeriod:         req.ToPeriod,
		Frequency:        req.Frequency,
		UptimePercentage: metrics.UptimePercentage,
		AvgLatencyMs:     metrics.AvgLatencyMs,
		MinLatencyMs:     metrics.MinLatencyMs,
		MaxLatencyMs:     metrics.MaxLatencyMs,
	}, nil
}
