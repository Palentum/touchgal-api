COMPOSE_FILE=deploy/docker-compose.yml
FRONTEND_TMPDIR ?= /tmp

.PHONY: dev backend-dev sync sync-full migrate-up frontend-dev test

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
