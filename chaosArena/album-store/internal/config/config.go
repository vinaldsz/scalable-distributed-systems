package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppPort              string
	DatabaseURL          string
	DBMaxConns           int32
	DBMinConns           int32
	AWSRegion            string
	S3Bucket             string
	WorkerCount          int
	WorkerQueueCap       int
	MaxConcurrentUploads int
	MaxUploadMB          int64
}

func Load() *Config {
	return &Config{
		AppPort:        getEnv("APP_PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://albumuser:secret@localhost:5432/albumstore?sslmode=disable"),
		DBMaxConns:     int32(getEnvInt("DB_MAX_CONNS", 40)),
		DBMinConns:     int32(getEnvInt("DB_MIN_CONNS", 5)),
		AWSRegion:      getEnv("AWS_REGION", "us-east-1"),
		S3Bucket:       getEnv("S3_BUCKET", "album-store-photos"),
		WorkerCount:          getEnvInt("WORKER_COUNT", 100),
		WorkerQueueCap:       getEnvInt("WORKER_QUEUE_CAP", 1000),
		MaxConcurrentUploads: getEnvInt("MAX_CONCURRENT_UPLOADS", 30),
		MaxUploadMB:          int64(getEnvInt("MAX_UPLOAD_MB", 200)),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
