package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Queryer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repositories struct {
	Users        *UserRepo
	Auth         *AuthRepo
	Applications *ApplicationRepo
	Tokens       *TokenRepo
	Games        *GameRepo
	Stats        *StatsRepo
	Sync         *SyncRepo
}

func New(pool *pgxpool.Pool) *Repositories {
	return NewWithQueryer(pool)
}

func NewWithQueryer(db Queryer) *Repositories {
	return &Repositories{
		Users:        NewUserRepo(db),
		Auth:         NewAuthRepo(db),
		Applications: NewApplicationRepo(db),
		Tokens:       NewTokenRepo(db),
		Games:        NewGameRepo(db),
		Stats:        NewStatsRepo(db),
		Sync:         NewSyncRepo(db),
	}
}

func positiveCapHint(limit int) int {
	if limit > 0 {
		return limit
	}
	return 0
}
