package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	Auth     AuthConfig
	Notifier NotifierConfig
	Internal InternalConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type PostgresConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

type AuthConfig struct {
	JWTSecret string
}

type NotifierConfig struct {
	NotifierUrl string
}

type InternalConfig struct {
	SecretKey string
}

func Load() (*Config, error) {
	_ = godotenv.Load("../.env")

	cfg := &Config{}

	cfg.Server.Port = getEnv("SERVER_PORT", ":8002")
	cfg.Server.ReadTimeout = getEnvDuration("SERVER_READ_TIMEOUT", 5*time.Second)
	cfg.Server.WriteTimeout = getEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second)

	cfg.Postgres.Host = getEnv("DB_HOST", "localhost")
	cfg.Postgres.Port = getEnvInt("DB_PORT", 5432)
	cfg.Postgres.User = getEnv("DB_USER", "postgres")
	cfg.Postgres.Password = getEnv("DB_PASSWORD", "postgres")
	cfg.Postgres.DBName = getEnv("DB_NAME", "catalog_db")
	cfg.Postgres.SSLMode = getEnv("DB_SSLMODE", "disable")
	cfg.Postgres.MaxConns = int32(getEnvInt("DB_MAX_CONNS", 10))
	cfg.Postgres.MinConns = int32(getEnvInt("DB_MIN_CONNS", 2))
	cfg.Postgres.MaxConnLifetime = getEnvDuration("DB_MAX_CONN_LIFETIME", 1*time.Hour)
	cfg.Postgres.MaxConnIdleTime = getEnvDuration("DB_MAX_CONN_IDLE_TIME", 15*time.Minute)

	cfg.Auth.JWTSecret = getEnv("JWT_SECRET", "super-secret-key-change-me-in-production")

	cfg.Notifier.NotifierUrl = getEnv("SERVICE_URL", "http://localhost:8004")

	cfg.Internal.SecretKey = getEnv("SECRET_KEY", "internal-secret")

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
