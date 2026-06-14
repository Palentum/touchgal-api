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
	"github.com/touchgal/developer/backend/internal/services/requestlog"
	"github.com/touchgal/developer/backend/internal/services/stats"
	syncsvc "github.com/touchgal/developer/backend/internal/services/sync"
	"github.com/touchgal/developer/backend/internal/services/token"
	usersvc "github.com/touchgal/developer/backend/internal/services/users"
)

type Services struct {
	Auth         *auth.Service
	Applications *application.Service
	Tokens       *token.Service
	Users        *usersvc.Service
	PublicAPI    *publicapi.Service
	Stats        *stats.Service
	RequestLogs  *requestlog.Writer
	Sync         *syncsvc.Service
	ReadinessDB  repository.Queryer
}

func NewRouter(cfg config.Config, services Services, repos *repository.Repositories, redisClient *redis.Client, logger zerolog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.Recover(logger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg))
	sessionAuth := middleware.SessionAuth(cfg, services.Auth)
	authCodeIPRateLimit := middleware.AuthCodeIPRateLimit(redisClient, cfg.AuthCodeIPMinuteLimit, cfg.AuthCodeIPDailyLimit)

	health := handlers.NewHealthHandler(services.ReadinessDB, redisClient)
	docs := handlers.DocsHandler{}
	authHandler := handlers.NewAuthHandler(cfg, services.Auth)
	applicationHandler := handlers.NewApplicationHandler(services.Applications)
	tokenHandler := handlers.NewTokenHandler(services.Tokens)
	publicHandler := handlers.NewPublicAPIHandler(services.PublicAPI)
	statsHandler := handlers.NewStatsHandler(services.Stats)
	adminHandler := handlers.NewAdminHandler(services.Applications, services.Tokens, services.Users, services.Auth, services.Sync, repos.Sync, cfg.EnableSyncWorker)

	r.Get("/openapi.yaml", docs.OpenAPI)
	r.Get("/docs", docs.Swagger)
	r.Get("/v1/health", health.Health)
	r.Get("/v1/ready", health.Ready)

	r.Route("/auth", func(r chi.Router) {
		r.Use(middleware.NoStore)
		r.With(authCodeIPRateLimit).Post("/register/start", authHandler.RegisterStart)
		r.Post("/register/verify", authHandler.RegisterVerify)
		r.With(authCodeIPRateLimit).Post("/login/start", authHandler.LoginStart)
		r.Post("/login/verify", authHandler.LoginVerify)
		r.Post("/logout", authHandler.Logout)
		r.With(sessionAuth).Get("/me", authHandler.Me)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.NoStore)
		r.Use(sessionAuth)
		r.Use(middleware.RequireUser)
		r.Get("/applications", applicationHandler.ListMine)
		r.Post("/applications", applicationHandler.Create)
		r.Get("/tokens", tokenHandler.ListMine)
		r.Post("/tokens", tokenHandler.Create)
		r.Patch("/tokens/{id}", tokenHandler.UpdateMine)
		r.Delete("/tokens/{id}", tokenHandler.DeleteMine)
		r.Get("/dashboard/stats", statsHandler.Dashboard)
	})

	r.Route("/admin", func(r chi.Router) {
		r.Use(middleware.NoStore)
		r.Use(sessionAuth)
		r.Use(middleware.RequireUser)
		r.Use(middleware.RequireAdmin)
		r.Get("/users", adminHandler.ListUsers)
		r.Patch("/users/{id}", adminHandler.UpdateUser)
		r.Delete("/users/{id}", adminHandler.DeleteUser)
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
			r.Use(middleware.APIPreAuthRateLimit(redisClient, cfg.APIPreAuthIPMinuteLimit, cfg.APIPreAuthIPDailyLimit))
			r.Use(middleware.APITokenAuth(services.Tokens))
			r.Use(middleware.APIRateLimit(redisClient))
			r.Use(middleware.APILastUsed(services.Tokens, redisClient, logger, cfg.APILastUsedUpdateInterval()))
			r.Use(middleware.APIRequestLog(services.RequestLogs))
			r.Get("/me", publicHandler.Me)
			r.Get("/games/search", publicHandler.Search)
			r.Get("/games/{uniqueId}", publicHandler.Detail)
			r.Get("/games/{uniqueId}/resources", publicHandler.Resources)
			r.Get("/games/{uniqueId}/patches", publicHandler.Patches)
		})
	})

	return r
}
