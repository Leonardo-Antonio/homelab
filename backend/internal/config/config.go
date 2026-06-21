package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr            string
	DatabasePath    string
	PhotoStorageDir string
	AllowedOrigin   string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func Load() Config {
	return Config{
		Addr:            getEnv("HTTP_ADDR", ":8080"),
		DatabasePath:    getEnv("DATABASE_PATH", "data/homelab.db"),
		PhotoStorageDir: getEnv("PHOTO_STORAGE_DIR", "data/photos"),
		AllowedOrigin:   getEnv("ALLOWED_ORIGIN", "http://localhost:5173,http://localhost:5174"),
		ReadTimeout:     getDurationSeconds("HTTP_READ_TIMEOUT_SECONDS", 5),
		WriteTimeout:    getDurationSeconds("HTTP_WRITE_TIMEOUT_SECONDS", 10),
		IdleTimeout:     getDurationSeconds("HTTP_IDLE_TIMEOUT_SECONDS", 60),
		ShutdownTimeout: getDurationSeconds("HTTP_SHUTDOWN_TIMEOUT_SECONDS", 10),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getDurationSeconds(key string, fallback int) time.Duration {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		value = fallback
	}

	return time.Duration(value) * time.Second
}
