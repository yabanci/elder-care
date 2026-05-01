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
	// Web Push (VAPID). Empty values disable push entirely; the API
	// then returns 503 on /api/push/subscribe and never tries to send.
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string // mailto: or app URL
}

func Load() Config {
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	ttl, err := strconv.Atoi(getenv("JWT_TTL_HOURS", "168"))
	if err != nil {
		log.Fatalf("invalid JWT_TTL_HOURS: %v", err)
	}

	return Config{
		DatabaseURL:     mustEnv("DATABASE_URL"),
		JWTSecret:       mustEnv("JWT_SECRET"),
		JWTTTLHours:     ttl,
		ServerAddr:      getenv("SERVER_ADDR", ":8080"),
		CORSOrigin:      getenv("CORS_ORIGIN", "http://localhost:3000"),
		VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		VAPIDSubject:    getenv("VAPID_SUBJECT", "mailto:admin@eldercare.local"),
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
