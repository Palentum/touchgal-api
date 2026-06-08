package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool, logger zerolog.Logger) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}

	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		version := strings.TrimSuffix(name, filepath.Ext(name))
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}

		raw, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		upSQL, err := extractGooseUp(string(raw))
		if err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, upSQL); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		logger.Info().Str("migration", version).Msg("migration applied")
	}
	return nil
}

func extractGooseUp(content string) (string, error) {
	upMarker := "-- +goose Up"
	downMarker := "-- +goose Down"
	upIndex := strings.Index(content, upMarker)
	if upIndex < 0 {
		return "", fmt.Errorf("missing %s marker", upMarker)
	}
	content = content[upIndex+len(upMarker):]
	if downIndex := strings.Index(content, downMarker); downIndex >= 0 {
		content = content[:downIndex]
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("empty up migration")
	}
	return content, nil
}
