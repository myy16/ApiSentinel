package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port         int
	GRPCPort     int
	DatabaseURL  string
	ValkeyURL    string
	JWTSecret    string
	JWTAccessTTL string
}

func Load() *Config {
	port, _ := strconv.Atoi(getEnv("PORT", "3001"))
	grpcPort, _ := strconv.Atoi(getEnv("GRPC_PORT", "50051"))

	return &Config{
		Port:         port,
		GRPCPort:     grpcPort,
		DatabaseURL:  getEnv("DATABASE_URL", "postgresql://apisentinel:apisentinel_secret@localhost:5432/apisentinel_db?sslmode=disable"),
		ValkeyURL:    getEnv("VALKEY_URL", "localhost:6379"),
		JWTSecret:    getEnv("JWT_SECRET", "super_secret_jwt_key_at_least_32_characters_long_12345"),
		JWTAccessTTL: getEnv("JWT_ACCESS_TTL", "15m"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
