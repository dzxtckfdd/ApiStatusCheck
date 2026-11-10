package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server      ServerConfig
	Internal    InternalConfig
	Catalog     CatalogConfig
	WorkerPool  WorkerPoolConfig
	HttpTimeout HttpTimeoutConfig
}

type ServerConfig struct {
	Port string
}

type InternalConfig struct {
	SecretKey string
}

type CatalogConfig struct {
	CatalogUrl string
	Interval   time.Duration
}

type WorkerPoolConfig struct {
	MaxWorkers int
}

type HttpTimeoutConfig struct {
	HttpTimeout time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load("../.env")

	cfg := &Config{}

	cfg.Server.Port = getEnv("SERVER_PORT", ":8003")

	cfg.Internal.SecretKey = getEnv("SECRET_KEY", "internal-secret")

	cfg.Catalog.CatalogUrl = getEnv("SERVICE_URL", "http://localhost:8002")
	cfg.Catalog.Interval = getEnvDuration("INTERVAL", 5)

	cfg.WorkerPool.MaxWorkers = getEnvInt("MAX_WORKERS", 20)

	cfg.HttpTimeout.HttpTimeout = getEnvDuration("HTTP_TIMEOUT", 10)

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	val, err := time.ParseDuration(valStr)
	if err != nil {
		return defaultVal
	}
	return val
}
