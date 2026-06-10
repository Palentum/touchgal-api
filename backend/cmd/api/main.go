package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/db"
	"github.com/touchgal/developer/backend/internal/httpserver"
	"github.com/touchgal/developer/backend/internal/logging"
	"github.com/touchgal/developer/backend/internal/repository"
	"github.com/touchgal/developer/backend/internal/services/application"
	"github.com/touchgal/developer/backend/internal/services/auth"
	"github.com/touchgal/developer/backend/internal/services/email"
	"github.com/touchgal/developer/backend/internal/services/publicapi"
	"github.com/touchgal/developer/backend/internal/services/stats"
	syncsvc "github.com/touchgal/developer/backend/internal/services/sync"
	"github.com/touchgal/developer/backend/internal/services/token"
	usersvc "github.com/touchgal/developer/backend/internal/services/users"
)

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("api stopped")
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger, err := logging.New(cfg.LogLevel, os.Stdout, !cfg.IsProduction())
	if err != nil {
		return err
	}
	logger.Debug().Str("app_env", cfg.AppEnv).Str("log_level", cfg.LogLevel).Msg("logger configured")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	target, err := db.OpenPostgres(ctx, cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	defer target.Close()
	if err := db.ApplyMigrations(ctx, target, logger); err != nil {
		return err
	}
	source, err := db.OpenOptionalPostgres(ctx, cfg.SourceDatabaseDSN)
	if err != nil {
		logger.Warn().Err(err).Msg("source database unavailable; API will continue and sync runs will fail until SOURCE_DATABASE_DSN is fixed")
		source = nil
	}
	if source != nil {
		defer source.Close()
	}
	redisClient, err := db.OpenRedis(ctx, cfg)
	if err != nil {
		return err
	}
	defer redisClient.Close()

	repos := repository.New(target)
	mailer := email.NewMailer(cfg, logger)

	authService := auth.NewService(cfg, repos.Users, repos.Auth, redisClient, mailer)
	applicationService := application.NewService(cfg, repos.Applications)
	tokenService := token.NewService(cfg, repos.Tokens, repos.Applications)
	userService := usersvc.NewService(repos.Users)
	publicService := publicapi.NewService(cfg, repos.Games)
	statsService := stats.NewService(repos.Stats)
	syncService := syncsvc.NewService(cfg, source, target, repos.Sync, logger)

	scheduler, err := syncsvc.StartScheduler(ctx, cfg, syncService, logger)
	if err != nil {
		return err
	}
	defer scheduler.Stop()

	server := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: httpserver.NewRouter(cfg, httpserver.Services{
			Auth:         authService,
			Applications: applicationService,
			Tokens:       tokenService,
			Users:        userService,
			PublicAPI:    publicService,
			Stats:        statsService,
			Sync:         syncService,
		}, repos, redisClient, logger),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	logger.Info().Str("addr", cfg.HTTPAddr).Msg("api listening")
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
