package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"userManager/internal/config"
	"userManager/internal/handlers"
	"userManager/internal/postgres"
	"userManager/internal/repository"
	"userManager/internal/service"
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

	userRepository := repository.NewUserRepository(pool)
	refreshTokenRepository := repository.NewRefreshTokenRepository(pool)

	userService := service.NewUserService(userRepository, refreshTokenRepository, cfg.JWT.SecretKey, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)
	userHandler := handlers.NewUserHandler(userService)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /auth/register", userHandler.UserRegisterHandler)
	mux.HandleFunc("POST /auth/login", userHandler.UserLoginHandler)
	mux.HandleFunc("POST /auth/refresh", userHandler.UserRefreshHandler)

	server := &http.Server{
		Addr:         cfg.Server.Port,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	log.Printf("userManager started on port %s", cfg.Server.Port)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}
