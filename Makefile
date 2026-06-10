COMPOSE_FILE=deploy/docker-compose.yml
FRONTEND_TMPDIR ?= /tmp

.PHONY: dev backend-dev sync sync-full migrate-up frontend-dev test bench perf-explain perf-explain-source frontend-analyze

dev:
	docker compose -f $(COMPOSE_FILE) up --build

backend-dev:
	cd backend && go run ./cmd/api

sync:
	cd backend && go run ./cmd/sync --mode=incremental

sync-full:
	cd backend && go run ./cmd/sync --mode=full

migrate-up:
	cd backend && goose -dir internal/db/migrations postgres "$$DATABASE_DSN" up

frontend-dev:
	cd frontend && TMPDIR="$(FRONTEND_TMPDIR)" pnpm dev

test:
	cd backend && go test ./...
	cd frontend && pnpm typecheck

bench:
	cd backend && go test -run='^$$' -bench=. -benchmem ./...

perf-explain:
	@test -n "$$DATABASE_DSN" || (printf 'DATABASE_DSN is required\n' && exit 1)
	psql "$$DATABASE_DSN" -v ON_ERROR_STOP=1 -v keyword="$${keyword:-summer}" -v page="$${page:-1}" -v limit="$${limit:-20}" -v days="$${days:-30}" -v unique_id="$${unique_id:-}" -v user_id="$${user_id:-}" -f scripts/perf-explain.sql

perf-explain-source:
	@test -n "$$SOURCE_DATABASE_DSN" || (printf 'SOURCE_DATABASE_DSN is required\n' && exit 1)
	psql "$$SOURCE_DATABASE_DSN" -v ON_ERROR_STOP=1 -v source_last_id="$${source_last_id:-0}" -v source_limit="$${source_limit:-1000}" -v source_window="$${source_window:-1 day}" -f scripts/perf-explain-source.sql

frontend-analyze:
	cd frontend && pnpm analyze
