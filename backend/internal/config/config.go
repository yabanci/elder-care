package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	JWTTTLHours int
	ServerAddr  string
	CORSOrigin  string
}

func Load() Config {
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	ttl, err := strconv.Atoi(getenv("JWT_TTL_HOURS", "168"))
	if err != nil {
		log.Fatalf("invalid JWT_TTL_HOURS: %v", err)
	}

	return Config{
		DatabaseURL: mustEnv("DATABASE_URL"),
		JWTSecret:   mustEnv("JWT_SECRET"),
		JWTTTLHours: ttl,
		ServerAddr:  getenv("SERVER_ADDR", ":8080"),
		CORSOrigin:  getenv("CORS_ORIGIN", "http://localhost:3000"),
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("missing required env: %s", k)
	}
	return v
}
