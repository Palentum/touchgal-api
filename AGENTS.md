# Repository Guidelines

## Project Overview

TouchGal API is an independent developer API and portal for sanitized Galgame metadata. It intentionally does not serve directly from the TouchGal main schema (`kun-touchgal-next`). The repo contains a Go backend, a standalone sync worker, a clean PostgreSQL schema, Redis-backed auth/rate-limit support, a Nuxt 3 developer portal, OpenAPI docs, and deployment examples.

Primary contract: sync only public metadata into the clean DB, then expose it through reviewed API tokens and the Nuxt portal. Do not copy main-site users, sessions, comments, resource links, or internal IDs into this system.

## Architecture & Data Flow

```text
TouchGal main PostgreSQL (read-only)
  -> backend/cmd/sync full|incremental
  -> TouchGal API clean PostgreSQL
  -> backend/cmd/api chi router + services
  -> /v1 API + Nuxt Developer Portal
```

- Backend entrypoints:
  - `backend/cmd/api/main.go`: loads config, opens DB/Redis, applies embedded migrations, builds repositories/services, starts the HTTP server.
  - `backend/cmd/sync/main.go`: runs one `incremental` or `full` sync against `SOURCE_DATABASE_DSN`.
- Backend layering: handlers (`backend/internal/httpserver/handlers`) -> services (`backend/internal/services`) -> repositories (`backend/internal/repository`) -> PostgreSQL/Redis. Keep validation/business rules in services, SQL in repositories, HTTP concerns in handlers/middleware.
- Router: `backend/internal/httpserver/server.go` uses `chi`. Route groups are `/auth`, authenticated app/token/dashboard routes, `/admin`, and `/v1` token-authenticated public API routes.
- Auth flow: email OTP -> server-side HttpOnly session cookie. Frontend must not store session/token secrets in `localStorage`.
- API token flow: approved application -> token created with `tgal_live` prefix -> SHA-256 hash with `API_TOKEN_PEPPER` stored -> plaintext returned once. `/v1` accepts bearer auth or `X-API-Token` and applies Redis rate limits.
- Sync flow: reads source `patch`, aliases, tag/company relations, and rating stats by keyset/page; writes clean tables (`games`, `game_aliases`, `tags`, `companies`, `game_rating_stats`) with short batch transactions. Full sync records seen source patch IDs in `sync_run_seen` and marks unseen rows deleted only after all batches succeed.
- Frontend flow: Nuxt pages call the backend through `useApi()` with `credentials: 'include'`; Pinia auth state gates dashboard/admin pages via route middleware.

## Key Directories

- `backend/cmd/api`, `backend/cmd/sync`: runnable Go entrypoints.
- `backend/internal/config`: `.env` loading and typed config validation.
- `backend/internal/db`: PostgreSQL/Redis setup and embedded SQL migration runner.
- `backend/internal/db/migrations`: goose-formatted schema files for clean DB tables.
- `backend/internal/model`: shared domain types, status constants, and sentinel errors.
- `backend/internal/repository`: pgx repositories; `Queryer` enables transaction/test injection.
- `backend/internal/services`: business logic for auth, tokens, applications, public API, stats, sync, email.
- `backend/internal/httpserver`: chi router, handlers, middleware, request/response helpers.
- `backend/internal/openapi`: OpenAPI file served by `/openapi.yaml`.
- `frontend/pages`: Nuxt routes for landing, auth, apply, dashboard, admin, docs.
- `frontend/components`: feature-grouped Vue components (`auth`, `application`, `dashboard`, `admin`, `landing`).
- `frontend/composables`: API and dashboard client helpers.
- `frontend/stores`: Pinia state, currently auth-focused.
- `frontend/middleware`: Nuxt route guards for auth/admin.
- `deploy`: Docker Compose, nginx, k8s CronJob, systemd examples.
- `docs`: architecture/deployment notes and public OpenAPI copy.

## Development Commands

Root `Makefile` is the main command surface:

```bash
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env

make dev           # docker compose -f deploy/docker-compose.yml up --build (backend/frontend only; requires host PostgreSQL/Redis)
make backend-dev   # cd backend && go run ./cmd/api
make frontend-dev  # cd frontend && pnpm dev
make migrate-up    # cd backend && goose -dir internal/db/migrations postgres "$DATABASE_DSN" up
make sync          # cd backend && go run ./cmd/sync --mode=incremental
make sync-full     # cd backend && go run ./cmd/sync --mode=full
make test          # cd backend && go test ./...; cd frontend && pnpm typecheck
```

Useful direct commands:

```bash
cd backend && go test ./...
cd frontend && pnpm typecheck
cd frontend && pnpm build
cd frontend && pnpm preview
```

`make migrate-up` needs the goose CLI and `DATABASE_DSN`; the API and sync binaries also call the embedded migration runner on startup.

## Code Conventions & Common Patterns

- Go code is standard `gofmt` Go. Use `context.Context` for DB/service calls and keep dependencies explicit through constructors.
- Services define narrow store interfaces (example: `publicapi.GameStore`) and contain input normalization/validation. Repositories implement SQL with pgx and expose typed methods.
- Shared repository injection goes through `repository.Queryer` and `repository.NewWithQueryer`; use this for transactions and focused tests rather than global state.
- Error handling uses sentinel errors from `backend/internal/model/errors.go`. HTTP mapping lives in `handlers/respond.go`; reuse `Success`, `Error`, `ErrorCode`, and `DecodeJSON` rather than hand-rolling response shapes.
- API responses are consistently `{ success: true, data }` or `{ success: false, error: { code, message } }`. Keep frontend TypeScript aligned with `ApiResponse<T>` in `frontend/composables/useApi.ts`.
- `DecodeJSON` rejects unknown fields. Preserve this behavior when adding handlers.
- Middleware owns cross-cutting HTTP concerns: request ID, recovery, CORS, session auth, admin checks, API token auth, rate limiting, and request logging.
- Do not log plaintext API tokens, OTP codes in production, session tokens, DSNs, or peppers. Token plaintext is only for the create-token response.
- Frontend uses Vue/Nuxt Composition API, Pinia setup stores, strict TypeScript, Nuxt UI, and feature-scoped PascalCase components.
- Frontend API calls should go through `useApi()` or typed composables such as `useDashboard()`. Keep backend base URLs in `runtimeConfig.public`, not hardcoded in components.
- Route protection belongs in `frontend/middleware/auth.ts` and `frontend/middleware/admin.ts`.
- Keep `docs/openapi.yaml` and `backend/internal/openapi/openapi.yaml` synchronized when API schemas or routes change.
- Schema changes belong in a new numbered file under `backend/internal/db/migrations` with goose `Up`/`Down` markers; do not mutate applied migrations casually.

## Important Files

- `README.md`: authoritative project purpose, security rules, env setup, local run, sync, and deployment notes.
- `Makefile`: canonical local commands.
- `backend/go.mod`: Go module (`github.com/touchgal/developer/backend`) and backend dependencies (`chi`, `pgx`, `go-redis`, `zerolog`, `cron`, `uuid`).
- `backend/.env.example`: backend runtime variables, including DB DSNs, Redis, session, SMTP, token, and sync settings.
- `frontend/package.json`: Nuxt scripts and package manager declaration.
- `frontend/.env.example`: public Nuxt runtime config variables.
- `frontend/nuxt.config.ts`: modules, runtime config, strict TypeScript/typecheck settings.
- `backend/internal/httpserver/server.go`: route topology and middleware order.
- `backend/internal/httpserver/handlers/respond.go`: response/error contract.
- `backend/internal/repository/common.go`: repository wiring and `Queryer` contract.
- `backend/internal/services/sync`: source queries, mapping, scheduler, and clean DB upsert flow.
- `docs/openapi.yaml`, `backend/internal/openapi/openapi.yaml`: OpenAPI 3.1 contract.
- `deploy/docker-compose.yml`: local/small deployment stack for backend/frontend; PostgreSQL and Redis are expected on the host.
- `deploy/k8s-cronjob-sync.yaml`: production-style detached incremental sync worker.

## Runtime/Tooling Preferences

- Backend runtime/toolchain: Go `1.26` (`backend/go.mod`). Docker builds with `golang:1.26-alpine` and runs on Alpine.
- Frontend package manager: `pnpm@11.5.0` (`frontend/package.json`). Use pnpm, not npm/yarn. Docker uses Node 22 with Corepack.
- Frontend framework: Nuxt 3, Vue 3, Pinia, Nuxt UI, ECharts/vue-echarts.
- Local services: PostgreSQL and Redis run on the host, not in `deploy/docker-compose.yml`; containers should use `host.docker.internal` when connecting back to host services.
- Configuration is env-driven. Copy `.env.example` files before running local services; `SOURCE_DATABASE_DSN` must be read-only. `ENABLE_SYNC_WORKER` defaults to `false`; production should run `backend/cmd/sync` separately unless intentionally enabling in-process sync for small/local deployments.
- OpenAPI is static YAML served by the backend docs handler and duplicated under `docs/` for documentation.
- No repository-wide lint config or CI workflow is present. Do not invent lint commands; use the configured tests/typecheck unless adding tooling intentionally.

## Testing & QA

- Canonical check: `make test` runs all Go tests and Nuxt typecheck.
- Backend tests use the standard Go `testing` package and are co-located with packages as `_test.go` files.
- Existing test coverage is concentrated in:
  - `backend/internal/services/application/service_test.go`
  - `backend/internal/services/auth/otp_test.go`
  - `backend/internal/services/auth/session_test.go`
  - `backend/internal/services/publicapi/service_test.go`
  - `backend/internal/services/token/service_test.go`
  - `backend/internal/httpserver/middleware/rate_limit_test.go`
- Existing tests use small package-local fakes and sentinel error assertions; no testify/mock framework is used.
- Frontend has no unit/component/E2E test framework configured. The available QA gate is `pnpm typecheck` / `nuxt typecheck`.
- No coverage threshold, OpenAPI validator, E2E suite, or CI config is configured in the repo.
- For backend behavior changes, add or update focused Go tests near the changed service/middleware package and run the package test or `cd backend && go test ./...`.
- For frontend changes, run `cd frontend && pnpm typecheck`; run `pnpm build` when changing Nuxt config, route structure, or SSR-sensitive code.
