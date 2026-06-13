package repository

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/touchgal/developer/backend/internal/model"
)

type StatsRepo struct{ db Queryer }

func NewStatsRepo(db Queryer) *StatsRepo { return &StatsRepo{db: db} }

func (r *StatsRepo) InsertRequestLogs(ctx context.Context, logs []model.RequestLog) error {
	if len(logs) == 0 {
		return nil
	}

	args := make([]any, 0, len(logs)*12)
	var sql strings.Builder
	sql.Grow(len(logs)*160 + 5000)
	sql.WriteString(`
		WITH batch (token_id, user_id, application_id, method, path, route, status_code, latency_ms, ip, user_agent, origin, referer) AS (
			VALUES `)
	for i, log := range logs {
		if i > 0 {
			sql.WriteByte(',')
		}
		base := i*12 + 1
		writeRequestLogValues(&sql, base)
		args = append(args,
			log.TokenID,
			log.UserID,
			log.ApplicationID,
			log.Method,
			log.Path,
			log.Route,
			log.StatusCode,
			log.LatencyMS,
			log.IP,
			log.UserAgent,
			log.Origin,
			log.Referer,
		)
	}
	sql.WriteString(`
		),
		logged AS (
			INSERT INTO api_request_logs (token_id, user_id, application_id, method, path, route, status_code, latency_ms, ip, user_agent, origin, referer)
			SELECT token_id, user_id, application_id, method, path, route, status_code, latency_ms, ip, user_agent, origin, referer
			FROM batch
			RETURNING 1
		),
		daily AS (
			INSERT INTO api_usage_daily (token_id, user_id, application_id, date, total_requests, success_requests, error_requests, total_latency_ms, avg_latency_ms, unique_ips)
			SELECT token_id,
			       user_id,
			       application_id,
			       CURRENT_DATE,
			       count(*)::int,
			       count(*) FILTER (WHERE status_code >= 200 AND status_code < 400)::int,
			       count(*) FILTER (WHERE status_code >= 400)::int,
			       coalesce(sum(latency_ms), 0)::bigint,
			       CASE WHEN count(*) = 0 THEN 0 ELSE (coalesce(sum(latency_ms), 0) / count(*))::int END,
			       count(DISTINCT nullif(ip, ''))::int
			FROM batch
			WHERE token_id IS NOT NULL AND user_id IS NOT NULL AND application_id IS NOT NULL
			GROUP BY token_id, user_id, application_id
			ON CONFLICT (token_id, date) DO UPDATE SET
				user_id = EXCLUDED.user_id,
				application_id = EXCLUDED.application_id,
				total_requests = api_usage_daily.total_requests + EXCLUDED.total_requests,
				success_requests = api_usage_daily.success_requests + EXCLUDED.success_requests,
				error_requests = api_usage_daily.error_requests + EXCLUDED.error_requests,
				total_latency_ms = api_usage_daily.total_latency_ms + EXCLUDED.total_latency_ms,
				avg_latency_ms = CASE
					WHEN api_usage_daily.total_requests + EXCLUDED.total_requests = 0 THEN 0
					ELSE ((api_usage_daily.total_latency_ms + EXCLUDED.total_latency_ms) / (api_usage_daily.total_requests + EXCLUDED.total_requests))::int
				END,
				unique_ips = GREATEST(api_usage_daily.unique_ips, EXCLUDED.unique_ips),
				updated_at = now()
			RETURNING 1
		),
		origin_daily AS (
			INSERT INTO api_usage_origin_daily (token_id, user_id, application_id, date, origin, referer_host, ip_prefix, requests)
			SELECT token_id,
			       user_id,
			       application_id,
			       CURRENT_DATE,
			       origin,
			       referer_host,
			       '',
			       count(*)::int
			FROM (
				SELECT token_id,
				       user_id,
				       application_id,
				       origin,
				       coalesce(nullif(split_part(regexp_replace(referer, '^https?://', ''), '/', 1), ''), 'unknown') AS referer_host
				FROM batch
				WHERE token_id IS NOT NULL AND user_id IS NOT NULL AND application_id IS NOT NULL
			) origins
			GROUP BY token_id, user_id, application_id, origin, referer_host
			ON CONFLICT (token_id, date, origin, referer_host, ip_prefix) DO UPDATE SET
				user_id = EXCLUDED.user_id,
				application_id = EXCLUDED.application_id,
				requests = api_usage_origin_daily.requests + EXCLUDED.requests
			RETURNING 1
		),
		route_daily AS (
			INSERT INTO api_usage_route_daily (token_id, user_id, application_id, date, route, requests, error_requests, total_latency_ms)
			SELECT token_id,
			       user_id,
			       application_id,
			       CURRENT_DATE,
			       route,
			       count(*)::int,
			       count(*) FILTER (WHERE status_code >= 400)::int,
			       coalesce(sum(latency_ms), 0)::bigint
			FROM batch
			WHERE token_id IS NOT NULL AND user_id IS NOT NULL AND application_id IS NOT NULL
			GROUP BY token_id, user_id, application_id, route
			ON CONFLICT (token_id, date, route) DO UPDATE SET
				user_id = EXCLUDED.user_id,
				application_id = EXCLUDED.application_id,
				requests = api_usage_route_daily.requests + EXCLUDED.requests,
				error_requests = api_usage_route_daily.error_requests + EXCLUDED.error_requests,
				total_latency_ms = api_usage_route_daily.total_latency_ms + EXCLUDED.total_latency_ms
			RETURNING 1
		),
		ip_daily AS (
			INSERT INTO api_usage_ip_daily (token_id, user_id, application_id, date, ip)
			SELECT DISTINCT token_id, user_id, application_id, CURRENT_DATE, ip
			FROM batch
			WHERE token_id IS NOT NULL AND user_id IS NOT NULL AND application_id IS NOT NULL AND ip <> ''
			ON CONFLICT (token_id, date, ip) DO UPDATE SET
				user_id = EXCLUDED.user_id,
				application_id = EXCLUDED.application_id
			RETURNING 1
		)
		SELECT (SELECT count(*) FROM logged),
		       (SELECT count(*) FROM daily),
		       (SELECT count(*) FROM origin_daily),
		       (SELECT count(*) FROM route_daily),
		       (SELECT count(*) FROM ip_daily)`)
	_, err := r.db.Exec(ctx, sql.String(), args...)
	return err
}

func writeRequestLogValues(sql *strings.Builder, base int) {
	sql.WriteByte('(')
	writeParam(sql, base)
	sql.WriteString("::uuid, ")
	writeParam(sql, base+1)
	sql.WriteString("::uuid, ")
	writeParam(sql, base+2)
	sql.WriteString("::uuid, ")
	writeParam(sql, base+3)
	sql.WriteString("::varchar, ")
	writeParam(sql, base+4)
	sql.WriteString("::varchar, ")
	writeParam(sql, base+5)
	sql.WriteString("::varchar, ")
	writeParam(sql, base+6)
	sql.WriteString("::int, ")
	writeParam(sql, base+7)
	sql.WriteString("::int, ")
	writeParam(sql, base+8)
	sql.WriteString("::varchar, ")
	writeParam(sql, base+9)
	sql.WriteString("::varchar, ")
	writeParam(sql, base+10)
	sql.WriteString("::varchar, ")
	writeParam(sql, base+11)
	sql.WriteString("::varchar)")
}

func writeParam(sql *strings.Builder, n int) {
	sql.WriteByte('$')
	sql.WriteString(strconv.Itoa(n))
}

func (r *StatsRepo) DeleteRequestLogsBefore(ctx context.Context, before time.Time, limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	var deleted int
	err := r.db.QueryRow(ctx, `
		WITH stale AS (
			SELECT id
			FROM api_request_logs
			WHERE created_at < $1
			ORDER BY created_at
			LIMIT $2
		),
		deleted AS (
			DELETE FROM api_request_logs logs
			USING stale
			WHERE logs.id = stale.id
			RETURNING 1
		)
		SELECT count(*)::int FROM deleted`, before, limit).Scan(&deleted)
	return deleted, err
}
func (r *StatsRepo) DeleteRequestLogRollupsBefore(ctx context.Context, before time.Time, limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	var deleted int
	err := r.db.QueryRow(ctx, `
		WITH ip_stale AS (
			SELECT id FROM api_usage_ip_daily WHERE date < $1::date ORDER BY date LIMIT $2
		),
		ip_deleted AS (
			DELETE FROM api_usage_ip_daily rollups
			USING ip_stale
			WHERE rollups.id = ip_stale.id
			RETURNING 1
		),
		route_stale AS (
			SELECT id FROM api_usage_route_daily WHERE date < $1::date ORDER BY date LIMIT $2
		),
		route_deleted AS (
			DELETE FROM api_usage_route_daily rollups
			USING route_stale
			WHERE rollups.id = route_stale.id
			RETURNING 1
		),
		origin_stale AS (
			SELECT id FROM api_usage_origin_daily WHERE date < $1::date ORDER BY date LIMIT $2
		),
		origin_deleted AS (
			DELETE FROM api_usage_origin_daily rollups
			USING origin_stale
			WHERE rollups.id = origin_stale.id
			RETURNING 1
		),
		daily_stale AS (
			SELECT id FROM api_usage_daily WHERE date < $1::date ORDER BY date LIMIT $2
		),
		daily_deleted AS (
			DELETE FROM api_usage_daily rollups
			USING daily_stale
			WHERE rollups.id = daily_stale.id
			RETURNING 1
		)
		SELECT (
			(SELECT count(*) FROM ip_deleted) +
			(SELECT count(*) FROM route_deleted) +
			(SELECT count(*) FROM origin_deleted) +
			(SELECT count(*) FROM daily_deleted)
		)::int`, before, limit).Scan(&deleted)
	return deleted, err
}

const dashboardAllTokensSQL = `
	WITH daily_rollups AS (
		SELECT date, total_requests, success_requests, error_requests, total_latency_ms
		FROM api_usage_daily
		WHERE user_id = $1
		  AND date >= (CURRENT_DATE - (($2::int - 1) * INTERVAL '1 day'))::date
	),
	origin_rollups AS (
		SELECT origin, referer_host, requests
		FROM api_usage_origin_daily
		WHERE user_id = $1
		  AND date >= (CURRENT_DATE - (($2::int - 1) * INTERVAL '1 day'))::date
	),
	ip_rollups AS (
		SELECT ip
		FROM api_usage_ip_daily
		WHERE user_id = $1
		  AND date >= (CURRENT_DATE - (($2::int - 1) * INTERVAL '1 day'))::date
	),
	route_rollups AS (
		SELECT route, requests, error_requests, total_latency_ms
		FROM api_usage_route_daily
		WHERE user_id = $1
		  AND date >= (CURRENT_DATE - (($2::int - 1) * INTERVAL '1 day'))::date
	),
	summary_daily AS (
		SELECT coalesce(sum(total_requests), 0)::int AS total_requests,
		       coalesce(sum(success_requests), 0)::int AS success_requests,
		       coalesce(sum(error_requests), 0)::int AS error_requests,
		       CASE
			       WHEN coalesce(sum(total_requests), 0) = 0 THEN 0
			       ELSE (coalesce(sum(total_latency_ms), 0) / sum(total_requests))::int
		       END AS avg_latency_ms
		FROM daily_rollups
	),
	summary_origins AS (
		SELECT count(DISTINCT nullif(origin, ''))::int AS unique_origins
		FROM origin_rollups
	),
	summary_ips AS (
		SELECT count(DISTINCT ip)::int AS unique_ips
		FROM ip_rollups
	),
	trend_rows AS (
		SELECT date,
		       to_char(date, 'YYYY-MM-DD') AS day,
		       coalesce(sum(total_requests), 0)::int AS total_requests,
		       coalesce(sum(success_requests), 0)::int AS success_requests,
		       coalesce(sum(error_requests), 0)::int AS error_requests
		FROM daily_rollups
		GROUP BY date
		ORDER BY date
	),
	source_rows AS (
		SELECT coalesce(nullif(origin, ''), 'unknown') AS origin,
		       coalesce(nullif(referer_host, ''), 'unknown') AS referer_host,
		       coalesce(sum(requests), 0)::int AS requests
		FROM origin_rollups
		GROUP BY origin, referer_host
		ORDER BY requests DESC, origin, referer_host
		LIMIT 20
	),
	endpoint_rows AS (
		SELECT route,
		       coalesce(sum(requests), 0)::int AS requests,
		       CASE WHEN coalesce(sum(requests), 0) = 0 THEN 0 ELSE (coalesce(sum(total_latency_ms), 0) / sum(requests))::int END AS avg_latency_ms,
		       CASE WHEN coalesce(sum(requests), 0) = 0 THEN 0 ELSE (coalesce(sum(error_requests), 0)::float / sum(requests)::float) END AS error_rate
		FROM route_rollups
		GROUP BY route
		ORDER BY requests DESC, route
		LIMIT 50
	)
	SELECT summary_daily.total_requests,
	       summary_daily.success_requests,
	       summary_daily.error_requests,
	       summary_daily.avg_latency_ms,
	       summary_origins.unique_origins,
	       summary_ips.unique_ips,
	       coalesce((
		       SELECT jsonb_agg(jsonb_build_object(
			       'date', day,
			       'totalRequests', total_requests,
			       'successRequests', success_requests,
			       'errorRequests', error_requests
		       ) ORDER BY date)
		       FROM trend_rows
	       ), '[]'::jsonb),
	       coalesce((
		       SELECT jsonb_agg(jsonb_build_object(
			       'origin', origin,
			       'refererHost', referer_host,
			       'requests', requests
		       ) ORDER BY requests DESC, origin, referer_host)
		       FROM source_rows
	       ), '[]'::jsonb),
	       coalesce((
		       SELECT jsonb_agg(jsonb_build_object(
			       'route', route,
			       'requests', requests,
			       'avgLatencyMs', avg_latency_ms,
			       'errorRate', error_rate
		       ) ORDER BY requests DESC, route)
		       FROM endpoint_rows
	       ), '[]'::jsonb)
	FROM summary_daily, summary_origins, summary_ips`

const dashboardSingleTokenSQL = `
	WITH daily_rollups AS (
		SELECT date, total_requests, success_requests, error_requests, total_latency_ms
		FROM api_usage_daily
		WHERE user_id = $1
		  AND date >= (CURRENT_DATE - (($2::int - 1) * INTERVAL '1 day'))::date
		  AND token_id = $3
	),
	origin_rollups AS (
		SELECT origin, referer_host, requests
		FROM api_usage_origin_daily
		WHERE user_id = $1
		  AND date >= (CURRENT_DATE - (($2::int - 1) * INTERVAL '1 day'))::date
		  AND token_id = $3
	),
	ip_rollups AS (
		SELECT ip
		FROM api_usage_ip_daily
		WHERE user_id = $1
		  AND date >= (CURRENT_DATE - (($2::int - 1) * INTERVAL '1 day'))::date
		  AND token_id = $3
	),
	route_rollups AS (
		SELECT route, requests, error_requests, total_latency_ms
		FROM api_usage_route_daily
		WHERE user_id = $1
		  AND date >= (CURRENT_DATE - (($2::int - 1) * INTERVAL '1 day'))::date
		  AND token_id = $3
	),
	summary_daily AS (
		SELECT coalesce(sum(total_requests), 0)::int AS total_requests,
		       coalesce(sum(success_requests), 0)::int AS success_requests,
		       coalesce(sum(error_requests), 0)::int AS error_requests,
		       CASE
			       WHEN coalesce(sum(total_requests), 0) = 0 THEN 0
			       ELSE (coalesce(sum(total_latency_ms), 0) / sum(total_requests))::int
		       END AS avg_latency_ms
		FROM daily_rollups
	),
	summary_origins AS (
		SELECT count(DISTINCT nullif(origin, ''))::int AS unique_origins
		FROM origin_rollups
	),
	summary_ips AS (
		SELECT count(DISTINCT ip)::int AS unique_ips
		FROM ip_rollups
	),
	trend_rows AS (
		SELECT date,
		       to_char(date, 'YYYY-MM-DD') AS day,
		       coalesce(sum(total_requests), 0)::int AS total_requests,
		       coalesce(sum(success_requests), 0)::int AS success_requests,
		       coalesce(sum(error_requests), 0)::int AS error_requests
		FROM daily_rollups
		GROUP BY date
		ORDER BY date
	),
	source_rows AS (
		SELECT coalesce(nullif(origin, ''), 'unknown') AS origin,
		       coalesce(nullif(referer_host, ''), 'unknown') AS referer_host,
		       coalesce(sum(requests), 0)::int AS requests
		FROM origin_rollups
		GROUP BY origin, referer_host
		ORDER BY requests DESC, origin, referer_host
		LIMIT 20
	),
	endpoint_rows AS (
		SELECT route,
		       coalesce(sum(requests), 0)::int AS requests,
		       CASE WHEN coalesce(sum(requests), 0) = 0 THEN 0 ELSE (coalesce(sum(total_latency_ms), 0) / sum(requests))::int END AS avg_latency_ms,
		       CASE WHEN coalesce(sum(requests), 0) = 0 THEN 0 ELSE (coalesce(sum(error_requests), 0)::float / sum(requests)::float) END AS error_rate
		FROM route_rollups
		GROUP BY route
		ORDER BY requests DESC, route
		LIMIT 50
	)
	SELECT summary_daily.total_requests,
	       summary_daily.success_requests,
	       summary_daily.error_requests,
	       summary_daily.avg_latency_ms,
	       summary_origins.unique_origins,
	       summary_ips.unique_ips,
	       coalesce((
		       SELECT jsonb_agg(jsonb_build_object(
			       'date', day,
			       'totalRequests', total_requests,
			       'successRequests', success_requests,
			       'errorRequests', error_requests
		       ) ORDER BY date)
		       FROM trend_rows
	       ), '[]'::jsonb),
	       coalesce((
		       SELECT jsonb_agg(jsonb_build_object(
			       'origin', origin,
			       'refererHost', referer_host,
			       'requests', requests
		       ) ORDER BY requests DESC, origin, referer_host)
		       FROM source_rows
	       ), '[]'::jsonb),
	       coalesce((
		       SELECT jsonb_agg(jsonb_build_object(
			       'route', route,
			       'requests', requests,
			       'avgLatencyMs', avg_latency_ms,
			       'errorRate', error_rate
		       ) ORDER BY requests DESC, route)
		       FROM endpoint_rows
	       ), '[]'::jsonb)
	FROM summary_daily, summary_origins, summary_ips`

func (r *StatsRepo) Dashboard(ctx context.Context, userID uuid.UUID, days int, tokenID *uuid.UUID) (model.StatsDashboard, error) {
	if tokenID == nil {
		return scanStatsDashboardRow(r.db.QueryRow(ctx, dashboardAllTokensSQL, userID, days))
	}
	return scanStatsDashboardRow(r.db.QueryRow(ctx, dashboardSingleTokenSQL, userID, days, *tokenID))
}

func scanStatsDashboardRow(row pgx.Row) (model.StatsDashboard, error) {
	var dashboard model.StatsDashboard
	var trendJSON, sourcesJSON, endpointsJSON []byte
	err := row.Scan(
		&dashboard.Summary.TotalRequests,
		&dashboard.Summary.SuccessRequests,
		&dashboard.Summary.ErrorRequests,
		&dashboard.Summary.AvgLatencyMS,
		&dashboard.Summary.UniqueOrigins,
		&dashboard.Summary.UniqueIPs,
		&trendJSON,
		&sourcesJSON,
		&endpointsJSON,
	)
	if err != nil {
		return model.StatsDashboard{}, err
	}
	if err := json.Unmarshal(trendJSON, &dashboard.Trend); err != nil {
		return model.StatsDashboard{}, err
	}
	if err := json.Unmarshal(sourcesJSON, &dashboard.Sources); err != nil {
		return model.StatsDashboard{}, err
	}
	if err := json.Unmarshal(endpointsJSON, &dashboard.Endpoints); err != nil {
		return model.StatsDashboard{}, err
	}
	return dashboard, nil
}
