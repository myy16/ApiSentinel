package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// LoadDotEnv loads a local dotenv file without overwriting variables already
// supplied by the process environment. It is intended for development only;
// production should use the platform/container secret store.
func LoadDotEnv(paths ...string) error {
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			line = strings.TrimPrefix(line, "export ")
			key, value, ok := strings.Cut(line, "=")
			if !ok || strings.TrimSpace(key) == "" {
				return fmt.Errorf("invalid dotenv entry in %s at line %d", path, lineNo)
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			value = strings.Trim(value, "\"'")
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, value)
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		return nil
	}
	return nil
}

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

	// Environment detection: prefer APP_ENV, fall back to NODE_ENV for backward compat
	env := strings.ToLower(getEnv("APP_ENV", ""))
	if env == "" {
		env = strings.ToLower(getEnv("NODE_ENV", "development"))
	}
	isProduction := env == "production"

	// SSRF Guard: local forwarding MUST be off in production.
	// Only allow in development when explicitly set or by default.
	if isProduction {
		os.Setenv("ALLOW_LOCAL_FORWARDING", "false")
	} else if os.Getenv("ALLOW_LOCAL_FORWARDING") == "" {
		os.Setenv("ALLOW_LOCAL_FORWARDING", "true")
	}

	// JWT Secret: require explicit value in production
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		if isProduction {
			log.Fatal().Msg("FATAL: JWT_SECRET environment variable is required in production mode")
		}
		jwtSecret = "super_secret_jwt_key_at_least_32_characters_long_12345"
		log.Warn().Msg("Using default JWT_SECRET — this is only acceptable in development mode")
	}

	// CORS: production MUST specify explicit origins (wildcard + credentials is unsafe)
	corsOrigin := getEnv("CORS_ORIGIN", "")
	if corsOrigin == "" || corsOrigin == "*" {
		if isProduction {
			log.Fatal().Msg("FATAL: CORS_ORIGIN must be set to explicit domain(s) in production (wildcard '*' with credentials is unsafe)")
		}
		corsOrigin = "*"
	}

	return &Config{
		Port:         port,
		GRPCPort:     grpcPort,
		DatabaseURL:  getEnv("DATABASE_URL", "postgresql://apisentinel:apisentinel_secret@localhost:5432/apisentinel_db?sslmode=disable"),
		ValkeyURL:    getEnv("VALKEY_URL", "localhost:6379"),
		JWTSecret:    jwtSecret,
		JWTAccessTTL: getEnv("JWT_ACCESS_TTL", "24h"),
		Environment:  env,
		CORSOrigin:   corsOrigin,
	}
}

func (c *Config) IsDevelopment() bool {
	return c.Environment != "production"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
