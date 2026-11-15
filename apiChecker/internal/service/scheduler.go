package service

import (
	"apiChecker/internal/client"
	"apiChecker/internal/dto"

	"context"
	"log"
	"sync"
	"time"
)

type trackedResource struct {
	item          dto.ResourceItem
	lastCheckedAt time.Time
}

type Scheduler struct {
	catalogClient   *client.CatalogClient
	outChan         chan<- dto.ResourceItem
	refreshInterval time.Duration
	tickInterval    time.Duration
	mu              sync.RWMutex
	targets         map[string]*trackedResource
}

func NewScheduler(
	catalogClient *client.CatalogClient,
	outChan chan<- dto.ResourceItem,
	refreshInterval time.Duration,
) *Scheduler {
	if refreshInterval <= 0 {
		refreshInterval = 5 * time.Minute
	}

	return &Scheduler{
		catalogClient:   catalogClient,
		outChan:         outChan,
		refreshInterval: refreshInterval,
		tickInterval:    1 * time.Second,
		targets:         make(map[string]*trackedResource),
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.refreshTargets(ctx)

	refreshTicker := time.NewTicker(s.refreshInterval)
	defer refreshTicker.Stop()

	tickTicker := time.NewTicker(s.tickInterval)
	defer tickTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-refreshTicker.C:
			s.refreshTargets(ctx)

		case <-tickTicker.C:
			s.dispatch(ctx)
		}
	}
}

func (s *Scheduler) refreshTargets(ctx context.Context) {
	resp, err := s.catalogClient.GetActiveResources(ctx)
	if err != nil {
		log.Printf("scheduler: failed to refresh active targets: %v", err)
		return
	}

	if resp == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	newTargets := make(map[string]*trackedResource, len(*resp))

	for _, item := range *resp {
		existing, found := s.targets[item.Id]
		if found {
			existing.item = item
			newTargets[item.Id] = existing
		} else {
			lastChecked := time.Now()
			if item.LastCheckedAt != nil {
				lastChecked = *item.LastCheckedAt
			}

			newTargets[item.Id] = &trackedResource{
				item:          item,
				lastCheckedAt: lastChecked,
			}
		}
	}

	s.targets = newTargets
	log.Printf("scheduler: loaded %d active targets", len(s.targets))
}

func (s *Scheduler) dispatch(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for _, target := range s.targets {
		interval := time.Duration(target.item.CheckIntervalSec) * time.Second
		if interval <= 0 {
			interval = 60 * time.Second
		}

		if now.Sub(target.lastCheckedAt) >= interval {
			select {
			case <-ctx.Done():
				return

			case s.outChan <- target.item:
				target.lastCheckedAt = now

			default:
				log.Printf("scheduler: worker queue full, skipping task for %s", target.item.Url)
			}
		}
	}
}
