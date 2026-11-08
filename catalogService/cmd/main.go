package main

import (
	"catalogService/internal/config"
	"catalogService/internal/postgres"
	"catalogService/internal/repository"
	"catalogService/internal/service"

	"context"

	"log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config : %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := postgres.NewPool(ctx, postgres.PgxConfig{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		DBName:   cfg.Postgres.DBName,
		SSLMode:  cfg.Postgres.SSLMode,

		MaxConns:        cfg.Postgres.MaxConns,
		MinConns:        cfg.Postgres.MinConns,
		MaxConnLifetime: cfg.Postgres.MaxConnLifetime,
		MaxConnIdleTime: cfg.Postgres.MaxConnIdleTime,
	})
	if err != nil {
		log.Fatalf("failed to connect to data base: %v", err)
	}
	defer pool.Close()

	checkRepository := repository.NewCheckRepository(pool)
	incidenRepository := repository.NewIncidentRepository(pool)
	metricRepository := repository.NewMetricRepository(pool)
	resourceRepository := repository.NewResourceRepository(pool)
	stateRepository := repository.NewStateRepository(pool)
	subscriptionRepository := repository.NewSubscriptionRepository(pool)

	resourceService := service.NewResourceService(resourceRepository, subscriptionRepository, metricRepository)
	internalService := service.NewInternalService(
		checkRepository,
		stateRepository,
		incidenRepository,
		resourceRepository,
		subscriptionRepository,
		cfg.Notifier.NotifierUrl,
		int(cfg.ApiChecker.OpenThreshold),
		int(cfg.ApiChecker.CloseThreshold),
	)

	
}
