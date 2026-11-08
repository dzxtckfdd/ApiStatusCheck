package main

import (
	"catalogService/internal/config"
	"catalogService/internal/handlers"
	"catalogService/internal/middleware"
	"catalogService/internal/postgres"
	"catalogService/internal/repository"
	"catalogService/internal/service"
	"errors"
	"net/http"

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

	clientHandler := handlers.NewClientHandler(resourceService)
	internalHandler := handlers.NewInternalHandler(internalService, cfg.Internal.SecretKey)

	authMw := middleware.AuthMiddleware(cfg.Auth.JWTSecret)
	optionalAuthMw := middleware.OptionalAuthMiddleware(cfg.Auth.JWTSecret)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/resources/public", clientHandler.GetPublicResources)
	mux.Handle("POST /api/v1/resources/metrics", optionalAuthMw(http.HandlerFunc(clientHandler.GetMetrics)))

	mux.Handle("POST /api/v1/resources", authMw(http.HandlerFunc(clientHandler.CreateResource)))
	mux.Handle("GET /api/v1/resources", authMw(http.HandlerFunc(clientHandler.GetUserResources)))

	mux.HandleFunc("GET /internal/resources", internalHandler.GetActiveTargets)
	mux.HandleFunc("POST /internal/checks", internalHandler.ProcessBatch)

	server := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	log.Printf("catalogService started on port %s", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server listen error: %v", err)
	}
}
