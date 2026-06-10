package main

import (
	"context"
	"flag"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/db"
	"github.com/touchgal/developer/backend/internal/logging"
	"github.com/touchgal/developer/backend/internal/repository"
	syncsvc "github.com/touchgal/developer/backend/internal/services/sync"
)

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("sync failed")
	}
}

func run() error {
	mode := flag.String("mode", syncsvc.ModeIncremental, "sync mode: incremental or full")
	flag.Parse()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger, err := logging.New(cfg.LogLevel, os.Stdout, false)
	if err != nil {
		return err
	}
	logger.Debug().Str("mode", *mode).Str("log_level", cfg.LogLevel).Msg("logger configured")
	ctx := context.Background()
	target, err := db.OpenPostgres(ctx, cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	defer target.Close()
	if err := db.ApplyMigrations(ctx, target, logger); err != nil {
		return err
	}
	source, err := db.OpenPostgres(ctx, cfg.SourceDatabaseDSN)
	if err != nil {
		return err
	}
	defer source.Close()
	repos := repository.New(target)
	service := syncsvc.NewService(cfg, source, target, repos.Sync, logger)
	run, err := service.Run(ctx, *mode)
	if err != nil {
		return err
	}
	logger.Info().Str("status", run.Status).Int("games_upserted", run.GamesUpserted).Msg("sync finished")
	return nil
}
