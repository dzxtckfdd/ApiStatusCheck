package main

import (
	"apiChecker/internal/client"
	"apiChecker/internal/config"
	"apiChecker/internal/dto"
	"apiChecker/internal/service"

	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config : %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("apiChecker starting (workers: %d, http_timeout: %v)...", cfg.WorkerPool.MaxWorkers, cfg.HttpTimeout.HttpTimeout)

	jobsChan := make(chan dto.ResourceItem, cfg.WorkerPool.MaxWorkers*2)
	resultsChan := make(chan dto.CheckItem, cfg.WorkerPool.MaxWorkers*4)

	catalogClient := client.NewCatalogClient(cfg.Catalog, cfg.Internal)
	checker := service.NewChecker(cfg.HttpTimeout.HttpTimeout)

	workerPool := service.NewWorkerPool(
		cfg.WorkerPool.MaxWorkers,
		checker,
		jobsChan,
		resultsChan,
	)

	batcher := service.NewBatcher(
		catalogClient,
		resultsChan,
		50,
		2*time.Second,
	)

	scheduler := service.NewScheduler(
		catalogClient,
		jobsChan,
		cfg.Catalog.Interval,
	)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		batcher.Run(ctx)
		log.Println("batcher stopped")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		workerPool.RunWorkerPool(ctx)
		log.Println("worker pool stopped")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(jobsChan)
		scheduler.Run(ctx)
		log.Println("scheduler stopped")
	}()

	log.Println("apiChecker is running. Press Ctrl+C to stop.")

	<-ctx.Done()

	log.Println("shutdown signal received, stopping components gracefully...")

	wg.Wait()
	log.Println("apiChecker stopped successfully")
}
