package database

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// RunMigrations applies all pending SQL migrations in order.
// It uses a `schema_migrations` table to track which migrations have been applied.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// 1. Ensure the tracking table exists
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// 2. Read all migration files
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var fileNames []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			fileNames = append(fileNames, e.Name())
		}
	}
	sort.Strings(fileNames)

	// 3. Apply each migration that hasn't been applied yet
	for _, fileName := range fileNames {
		var exists bool
		err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
			fileName,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", fileName, err)
		}

		if exists {
			continue
		}

		// Read and execute the migration
		content, err := migrationFS.ReadFile("migrations/" + fileName)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", fileName, err)
		}

		log.Info().Str("migration", fileName).Msg("Applying database migration...")

		_, err = pool.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", fileName, err)
		}

		// Record the migration
		_, err = pool.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)",
			fileName,
		)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", fileName, err)
		}

		log.Info().Str("migration", fileName).Msg("Migration applied successfully")
	}

	return nil
}
