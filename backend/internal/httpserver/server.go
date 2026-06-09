package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/touchgal/developer/backend/internal/config"
	"github.com/touchgal/developer/backend/internal/httpserver/handlers"
	"github.com/touchgal/developer/backend/internal/httpserver/middleware"
	"github.com/touchgal/developer/backend/internal/repository"
	"github.com/touchgal/developer/backend/internal/services/application"
	"github.com/touchgal/developer/backend/internal/services/auth"
	"github.com/touchgal/developer/backend/internal/services/publicapi"
	"github.com/touchgal/developer/backend/internal/services/stats"
	syncsvc "github.com/touchgal/developer/backend/internal/services/sync"
	"github.com/touchgal/developer/backend/internal/services/token"
)

type Services struct {
	Auth         *auth.Service
	Applications *application.Service
	Tokens       *token.Service
	PublicAPI    *publicapi.Service
	Stats        *stats.Service
	Sync         *syncsvc.Service
}

func NewRouter(cfg config.Config, services Services, repos *repository.Repositories, redisClient *redis.Client, logger zerolog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.Recover(logger))
	r.Use(middleware.CORS(cfg))
	r.Use(middleware.SessionAuth(cfg, services.Auth))

	health := handlers.HealthHandler{}
	docs := handlers.DocsHandler{}
	authHandler := handlers.NewAuthHandler(cfg, services.Auth)
	applicationHandler := handlers.NewApplicationHandler(services.Applications)
	tokenHandler := handlers.NewTokenHandler(services.Tokens)
	publicHandler := handlers.NewPublicAPIHandler(services.PublicAPI)
	statsHandler := handlers.NewStatsHandler(services.Stats)
	adminHandler := handlers.NewAdminHandler(services.Applications, services.Tokens, services.Sync, repos.Sync)

	r.Get("/openapi.yaml", docs.OpenAPI)
	r.Get("/docs", docs.Swagger)
	r.Get("/v1/health", health.Health)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register/start", authHandler.RegisterStart)
		r.Post("/register/verify", authHandler.RegisterVerify)
		r.Post("/login/start", authHandler.LoginStart)
		r.Post("/login/verify", authHandler.LoginVerify)
		r.Post("/logout", authHandler.Logout)
		r.Get("/me", authHandler.Me)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireUser)
		r.Get("/applications", applicationHandler.ListMine)
		r.Post("/applications", applicationHandler.Create)
		r.Get("/tokens", tokenHandler.ListMine)
		r.Post("/tokens", tokenHandler.Create)
		r.Patch("/tokens/{id}", tokenHandler.UpdateMine)
		r.Delete("/tokens/{id}", tokenHandler.DeleteMine)
		r.Get("/dashboard/stats/summary", statsHandler.Summary)
		r.Get("/dashboard/stats/trend", statsHandler.Trend)
		r.Get("/dashboard/stats/sources", statsHandler.Sources)
		r.Get("/dashboard/stats/endpoints", statsHandler.Endpoints)
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.RequireUser)
		r.Use(middleware.RequireAdmin)
		r.Get("/applications", adminHandler.ListApplications)
		r.Post("/applications/{id}/approve", adminHandler.ApproveApplication)
		r.Post("/applications/{id}/reject", adminHandler.RejectApplication)
		r.Post("/applications/{id}/revoke", adminHandler.RevokeApplication)
		r.Get("/tokens", adminHandler.ListTokens)
		r.Delete("/tokens/{id}", adminHandler.DeleteToken)
		r.Get("/sync/runs", adminHandler.SyncRuns)
		r.Post("/sync/run", adminHandler.RunSync)
	})

	r.Route("/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middleware.APITokenAuth(services.Tokens))
			r.Use(middleware.APIRequestLog(repos.Stats, logger))
			r.Use(middleware.APIRateLimit(redisClient))
			r.Get("/me", publicHandler.Me)
			r.Get("/games/search", publicHandler.Search)
			r.Get("/games/{uniqueId}", publicHandler.Detail)
		})
	})

	return r
}
