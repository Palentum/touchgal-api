package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/touchgal/developer/backend/internal/model"
)

type StatsRepo struct{ db Queryer }

func NewStatsRepo(db Queryer) *StatsRepo { return &StatsRepo{db: db} }

func (r *StatsRepo) InsertRequestLog(ctx context.Context, log model.RequestLog) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO api_request_logs (token_id, user_id, application_id, method, path, route, status_code, latency_ms, ip, user_agent, origin, referer)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		log.TokenID, log.UserID, log.ApplicationID, log.Method, log.Path, log.Route, log.StatusCode, log.LatencyMS, log.IP, log.UserAgent, log.Origin, log.Referer)
	return err
}

func (r *StatsRepo) Summary(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) (model.StatsSummary, error) {
	where, args := statsWhere(userID, days, tokenID)
	var summary model.StatsSummary
	err := r.db.QueryRow(ctx, fmt.Sprintf(`
		SELECT count(*)::int,
		       count(*) FILTER (WHERE status_code >= 200 AND status_code < 400)::int,
		       count(*) FILTER (WHERE status_code >= 400)::int,
		       coalesce(avg(latency_ms), 0)::int,
		       count(DISTINCT nullif(origin, ''))::int,
		       count(DISTINCT nullif(ip, ''))::int
		FROM api_request_logs WHERE %s`, where), args...).Scan(&summary.TotalRequests, &summary.SuccessRequests, &summary.ErrorRequests, &summary.AvgLatencyMS, &summary.UniqueOrigins, &summary.UniqueIPs)
	return summary, err
}

func (r *StatsRepo) Trend(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) ([]model.StatsTrend, error) {
	where, args := statsWhere(userID, days, tokenID)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD') AS day,
		       count(*)::int,
		       count(*) FILTER (WHERE status_code >= 200 AND status_code < 400)::int,
		       count(*) FILTER (WHERE status_code >= 400)::int
		FROM api_request_logs WHERE %s
		GROUP BY day ORDER BY day`, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.StatsTrend{}
	for rows.Next() {
		var item model.StatsTrend
		if err := rows.Scan(&item.Date, &item.TotalRequests, &item.SuccessRequests, &item.ErrorRequests); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *StatsRepo) Sources(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) ([]model.StatsSource, error) {
	where, args := statsWhere(userID, days, tokenID)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT coalesce(nullif(origin, ''), 'unknown') AS origin,
		       coalesce(nullif(split_part(regexp_replace(referer, '^https?://', ''), '/', 1), ''), 'unknown') AS referer_host,
		       count(*)::int
		FROM api_request_logs WHERE %s
		GROUP BY origin, referer_host ORDER BY count(*) DESC LIMIT 20`, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.StatsSource{}
	for rows.Next() {
		var item model.StatsSource
		if err := rows.Scan(&item.Origin, &item.RefererHost, &item.Requests); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *StatsRepo) Endpoints(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) ([]model.StatsEndpoint, error) {
	where, args := statsWhere(userID, days, tokenID)
	rows, err := r.db.Query(ctx, fmt.Sprintf(`
		SELECT route,
		       count(*)::int,
		       coalesce(avg(latency_ms), 0)::int,
		       CASE WHEN count(*) = 0 THEN 0 ELSE (count(*) FILTER (WHERE status_code >= 400)::float / count(*)::float) END
		FROM api_request_logs WHERE %s
		GROUP BY route ORDER BY count(*) DESC LIMIT 50`, where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.StatsEndpoint{}
	for rows.Next() {
		var item model.StatsEndpoint
		if err := rows.Scan(&item.Route, &item.Requests, &item.AvgLatencyMS, &item.ErrorRate); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func statsWhere(userID uuid.UUID, days int, tokenID *uuid.UUID) (string, []any) {
	if tokenID == nil {
		return "user_id = $1 AND created_at >= now() - ($2::int * interval '1 day')", []any{userID, days}
	}
	return "user_id = $1 AND created_at >= now() - ($2::int * interval '1 day') AND token_id = $3", []any{userID, days, *tokenID}
}
