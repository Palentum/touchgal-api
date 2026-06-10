package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/touchgal/developer/backend/internal/model"
)

type recordingStatsQueryer struct {
	execSQL  string
	execArgs []any
	rowSQL   string
	rowArgs  []any
	row      pgx.Row
}

func (q *recordingStatsQueryer) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	q.execSQL = sql
	q.execArgs = arguments
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (q *recordingStatsQueryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	panic("unexpected query")
}

func (q *recordingStatsQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	q.rowSQL = sql
	q.rowArgs = args
	return q.row
}

type statsDashboardRow struct{}

func (statsDashboardRow) Scan(dest ...any) error {
	for _, d := range dest {
		switch target := d.(type) {
		case *int:
			*target = 0
		case *float64:
			*target = 0
		case *[]byte:
			*target = []byte("[]")
		}
	}
	return nil
}

func TestStatsRepoInsertRequestLogsWritesRawAndAggregatesInOneBatch(t *testing.T) {
	queryer := &recordingStatsQueryer{}
	repo := NewStatsRepo(queryer)
	tokenID := uuid.New()
	userID := uuid.New()
	applicationID := uuid.New()

	err := repo.InsertRequestLogs(context.Background(), []model.RequestLog{
		{TokenID: &tokenID, UserID: &userID, ApplicationID: &applicationID, Method: httpMethodGet, Path: "/v1/me", Route: "/v1/me", StatusCode: 200, LatencyMS: 12, IP: "203.0.113.10"},
		{TokenID: &tokenID, UserID: &userID, ApplicationID: &applicationID, Method: httpMethodGet, Path: "/v1/games/search", Route: "/v1/games/search", StatusCode: 500, LatencyMS: 34, IP: "203.0.113.10"},
	})
	if err != nil {
		t.Fatalf("insert request logs: %v", err)
	}
	if len(queryer.execArgs) != 24 {
		t.Fatalf("args got %d", len(queryer.execArgs))
	}
	sql := strings.ToLower(queryer.execSQL)
	for _, want := range []string{
		"with batch",
		"insert into api_request_logs",
		"insert into api_usage_daily",
		"insert into api_usage_origin_daily",
		"insert into api_usage_route_daily",
		"insert into api_usage_ip_daily",
		"on conflict (token_id, date) do update",
		"$24::varchar",
	} {
		if !strings.Contains(sql, want) {
			t.Fatalf("batch SQL missing %q: %s", want, queryer.execSQL)
		}
	}
}

func TestStatsRepoDashboardReadsAggregatesNotRawLogs(t *testing.T) {
	queryer := &recordingStatsQueryer{row: statsDashboardRow{}}
	repo := NewStatsRepo(queryer)
	userID := uuid.New()

	if _, err := repo.Dashboard(context.Background(), userID, 30, nil); err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	sql := strings.ToLower(queryer.rowSQL)
	if strings.Contains(sql, "api_request_logs") {
		t.Fatalf("dashboard must not read raw request logs: %s", queryer.rowSQL)
	}
	for _, want := range []string{"api_usage_daily", "api_usage_origin_daily", "api_usage_ip_daily", "api_usage_route_daily", "jsonb_agg"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("dashboard SQL missing %q: %s", want, queryer.rowSQL)
		}
	}
	if len(queryer.rowArgs) != 2 || queryer.rowArgs[0] != userID || queryer.rowArgs[1] != 30 {
		t.Fatalf("dashboard args got %#v", queryer.rowArgs)
	}
}

func TestStatsRepoDeletesRollupRowsByDate(t *testing.T) {
	queryer := &recordingStatsQueryer{row: statsDashboardRow{}}
	repo := NewStatsRepo(queryer)
	before := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	if _, err := repo.DeleteRequestLogRollupsBefore(context.Background(), before, 100); err != nil {
		t.Fatalf("delete rollups: %v", err)
	}
	sql := strings.ToLower(queryer.rowSQL)
	for _, want := range []string{"api_usage_ip_daily", "api_usage_route_daily", "api_usage_origin_daily", "api_usage_daily"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("rollup cleanup SQL missing %q: %s", want, queryer.rowSQL)
		}
	}
	if strings.Contains(sql, "api_request_logs") {
		t.Fatalf("rollup cleanup must not delete raw request logs: %s", queryer.rowSQL)
	}
	if len(queryer.rowArgs) != 2 || queryer.rowArgs[0] != before || queryer.rowArgs[1] != 100 {
		t.Fatalf("cleanup args got %#v", queryer.rowArgs)
	}
}

const httpMethodGet = "GET"
