package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type Config struct {
	Port         int
	GRPCPort     int
	DatabaseURL  string
	ValkeyURL    string
	JWTSecret    string
	JWTAccessTTL string
	Environment  string // "development" or "production"
	CORSOrigin   string // Allowed CORS origin(s), comma-separated
}

func Load() *Config {
	port, _ := strconv.Atoi(getEnv("PORT", "3001"))
	grpcPort, _ := strconv.Atoi(getEnv("GRPC_PORT", "50051"))
	env := strings.ToLower(getEnv("APP_ENV", "development"))

	if os.Getenv("ALLOW_LOCAL_FORWARDING") == "" {
		os.Setenv("ALLOW_LOCAL_FORWARDING", "true")
	}

	// JWT Secret: require explicit value in production
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if env == "production" {
			log.Fatal().Msg("FATAL: JWT_SECRET environment variable is required in production mode")
		}
		jwtSecret = "super_secret_jwt_key_at_least_32_characters_long_12345"
		log.Warn().Msg("Using default JWT_SECRET — this is only acceptable in development mode")
	}

	return &Config{
		Port:         port,
		GRPCPort:     grpcPort,
		DatabaseURL:  getEnv("DATABASE_URL", "postgresql://apisentinel:apisentinel_secret@localhost:5432/apisentinel_db?sslmode=disable"),
		ValkeyURL:    getEnv("VALKEY_URL", "localhost:6379"),
		JWTSecret:    jwtSecret,
		JWTAccessTTL: getEnv("JWT_ACCESS_TTL", "24h"),
		Environment:  env,
		CORSOrigin:   getEnv("CORS_ORIGIN", "*"),
	}
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
