package internal

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	ArchivePath string
	DBPath      string
	Port        string
	CORSOrigin  string
	StaticPath  string
}

func Load() (*Config, error) {
	cfg := &Config{
		ArchivePath: os.Getenv("CCTV_ARCHIVE_PATH"),
		DBPath:      os.Getenv("DB_PATH"),
		Port:        getenv("API_PORT", "8080"),
		CORSOrigin:  os.Getenv("CORS_ALLOWED_ORIGIN"),
		StaticPath:  getenv("STATIC_PATH", "/app/web"),
	}
	if cfg.ArchivePath == "" {
		return nil, errors.New("CCTV_ARCHIVE_PATH is required")
	}
	if cfg.DBPath == "" {
		return nil, errors.New("DB_PATH is required")
	}
	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return nil, errors.New("API_PORT must be a number")
	}
	if info, err := os.Stat(cfg.ArchivePath); err != nil {
		return nil, errors.New("CCTV_ARCHIVE_PATH does not exist or is unreadable: " + cfg.ArchivePath)
	} else if !info.IsDir() {
		return nil, errors.New("CCTV_ARCHIVE_PATH is not a directory: " + cfg.ArchivePath)
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
