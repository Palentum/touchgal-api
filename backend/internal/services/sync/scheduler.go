package syncsvc

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/config"
)

type Scheduler struct {
	cron *cron.Cron
}

func StartScheduler(ctx context.Context, cfg config.Config, service *Service, log zerolog.Logger) (*Scheduler, error) {
	if service == nil || !cfg.EnableSyncWorker {
		return &Scheduler{}, nil
	}
	c := cron.New(cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)))
	if cfg.SyncIntervalMinutes > 0 {
		spec := fmt.Sprintf("@every %dm", cfg.SyncIntervalMinutes)
		if _, err := c.AddFunc(spec, func() { run(ctx, service, ModeIncremental, log) }); err != nil {
			return nil, err
		}
	}
	if cfg.SyncFullIntervalHours > 0 {
		spec := fmt.Sprintf("@every %dh", cfg.SyncFullIntervalHours)
		if _, err := c.AddFunc(spec, func() { run(ctx, service, ModeFull, log) }); err != nil {
			return nil, err
		}
	}
	c.Start()
	return &Scheduler{cron: c}, nil
}

func (s *Scheduler) Stop() {
	if s != nil && s.cron != nil {
		s.cron.Stop()
	}
}

func run(ctx context.Context, service *Service, mode string, log zerolog.Logger) {
	run, err := service.Run(ctx, mode)
	if err != nil {
		log.Error().Err(err).Str("mode", mode).Msg("sync failed")
		return
	}
	log.Info().Str("mode", mode).Str("status", run.Status).Int("games", run.GamesUpserted).Msg("sync finished")
}
