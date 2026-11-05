package main

import (
	"context"
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

	http.HandleFunc("/auth/register", userHandler.UserRegisterHandler)
	http.HandleFunc("/auth/login", userHandler.UserLoginHandler)
	http.HandleFunc("/auth/refresh", userHandler.UserRefreshHandler)

	http.ListenAndServe(cfg.Server.Port, nil)
}
