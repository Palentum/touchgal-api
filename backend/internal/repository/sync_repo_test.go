package repository

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/touchgal/developer/backend/internal/model"
)

type recordingSyncQueryer struct {
	sqls []string
	args [][]any
}

func (q *recordingSyncQueryer) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	q.sqls = append(q.sqls, sql)
	q.args = append(q.args, arguments)
	return pgconn.NewCommandTag("UPDATE 3"), nil
}

func (q *recordingSyncQueryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	panic("unexpected query")
}

func (q *recordingSyncQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	panic("unexpected query row")
}

type discardSyncQueryer struct{}

func (discardSyncQueryer) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("UPDATE 3"), nil
}

func (discardSyncQueryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	panic("unexpected query")
}

func (discardSyncQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	panic("unexpected query row")
}

func TestSyncRepoUpsertGamesUsesSingleBulkExec(t *testing.T) {
	queryer := &recordingSyncQueryer{}
	repo := NewSyncRepo(queryer)

	err := repo.UpsertGames(context.Background(), []model.CleanGame{
		{UniqueID: "g0000001", SourcePatchID: 10, Name: "Game 1"},
		{UniqueID: "g0000002", SourcePatchID: 11, Name: "Game 2"},
	})
	if err != nil {
		t.Fatalf("upsert games: %v", err)
	}
	if len(queryer.sqls) != 1 {
		t.Fatalf("expected one bulk exec, got %d", len(queryer.sqls))
	}
	if !strings.Contains(queryer.sqls[0], "jsonb_to_recordset($1::jsonb)") || !strings.Contains(queryer.sqls[0], "INSERT INTO games") {
		t.Fatalf("expected JSONB bulk game upsert, got %q", queryer.sqls[0])
	}
	var rows []model.CleanGame
	decodeJSONArg(t, queryer.args[0][0], &rows)
	if len(rows) != 2 || rows[0].UniqueID != "g0000001" || rows[1].SourcePatchID != 11 {
		t.Fatalf("expected game payload rows, got %#v", rows)
	}
}

func TestSyncRepoReplaceAliasesBatchCleansAndUsesBulkExecs(t *testing.T) {
	queryer := &recordingSyncQueryer{}
	repo := NewSyncRepo(queryer)

	err := repo.ReplaceAliasesBatch(context.Background(), map[string][]string{
		" g0000001 ": {"  Alpha  ", "Alpha", "", "Beta"},
		"g0000002":   {"   "},
		"   ":        {"ignored"},
	})
	if err != nil {
		t.Fatalf("replace aliases batch: %v", err)
	}
	if len(queryer.sqls) != 2 {
		t.Fatalf("expected delete and insert bulk execs, got %d", len(queryer.sqls))
	}
	if !strings.Contains(queryer.sqls[0], "DELETE FROM game_aliases") {
		t.Fatalf("expected bulk alias delete, got %q", queryer.sqls[0])
	}
	if !strings.Contains(queryer.sqls[1], "jsonb_to_recordset($1::jsonb)") || !strings.Contains(queryer.sqls[1], "CROSS JOIN LATERAL unnest") {
		t.Fatalf("expected JSONB bulk alias insert, got %q", queryer.sqls[1])
	}

	var rows []aliasBatchRow
	decodeJSONArg(t, queryer.args[0][0], &rows)
	if len(rows) != 2 {
		t.Fatalf("expected two affected games, got %#v", rows)
	}
	g1Aliases := aliasRowFor(rows, "g0000001")
	if len(g1Aliases.Aliases) != 2 || g1Aliases.Aliases[0] != "Alpha" || g1Aliases.Aliases[1] != "Beta" {
		t.Fatalf("expected trimmed unique aliases, got %#v", g1Aliases)
	}
	g2Aliases := aliasRowFor(rows, "g0000002")
	if g2Aliases.UniqueID != "g0000002" || len(g2Aliases.Aliases) != 0 {
		t.Fatalf("expected empty alias game to be deleted without insert values, got %#v", g2Aliases)
	}
}

func TestSyncRepoReplaceAliasesBatchSkipsEmptyInput(t *testing.T) {
	queryer := &recordingSyncQueryer{}
	repo := NewSyncRepo(queryer)

	err := repo.ReplaceAliasesBatch(context.Background(), map[string][]string{" ": {"ignored"}})
	if err != nil {
		t.Fatalf("replace aliases batch: %v", err)
	}
	if len(queryer.sqls) != 0 {
		t.Fatalf("expected no exec for empty affected games, got %d", len(queryer.sqls))
	}
}

func TestSyncRepoReplaceTagsBatchDedupeEmptyNamesAndUsesBulkSQL(t *testing.T) {
	queryer := &recordingSyncQueryer{}
	repo := NewSyncRepo(queryer)

	err := repo.ReplaceTagsBatch(context.Background(), map[string][]model.TagData{
		"g0000001": {
			{Name: "  Moe ", Aliases: []string{"萌"}, Source: "touchgal"},
			{Name: "Moe", Source: "duplicate"},
			{Name: "   ", Source: "empty"},
		},
		"g0000002": {{Name: " "}},
	})
	if err != nil {
		t.Fatalf("replace tags batch: %v", err)
	}
	if len(queryer.sqls) != 2 {
		t.Fatalf("expected delete and insert bulk execs, got %d", len(queryer.sqls))
	}
	if !strings.Contains(queryer.sqls[0], "DELETE FROM game_tags") {
		t.Fatalf("expected bulk tag relation delete, got %q", queryer.sqls[0])
	}
	if !strings.Contains(queryer.sqls[1], "INSERT INTO tags") || !strings.Contains(queryer.sqls[1], "INSERT INTO game_tags") {
		t.Fatalf("expected set-based tag upsert and relation insert, got %q", queryer.sqls[1])
	}

	var rows []tagBatchRow
	decodeJSONArg(t, queryer.args[0][0], &rows)
	if len(rows) != 2 {
		t.Fatalf("expected one tag row plus one delete sentinel, got %#v", rows)
	}
	g1Tag := tagRowFor(rows, "g0000001")
	if g1Tag.Name != "Moe" || g1Tag.Source != "touchgal" {
		t.Fatalf("expected trimmed first tag, got %#v", g1Tag)
	}
	g2Tag := tagRowFor(rows, "g0000002")
	if g2Tag.UniqueID != "g0000002" || g2Tag.Name != "" {
		t.Fatalf("expected empty-name sentinel for affected game, got %#v", g2Tag)
	}
}

func TestSyncRepoReplaceCompaniesBatchDedupeEmptyNamesAndUsesBulkSQL(t *testing.T) {
	queryer := &recordingSyncQueryer{}
	repo := NewSyncRepo(queryer)

	err := repo.ReplaceCompaniesBatch(context.Background(), map[string][]model.CompanyData{
		"g0000001": {
			{Name: "  Studio ", OfficialWebsites: []string{"https://example.test"}},
			{Name: "Studio", ParentBrands: []string{"ignored duplicate"}},
			{Name: ""},
		},
		"g0000002": nil,
	})
	if err != nil {
		t.Fatalf("replace companies batch: %v", err)
	}
	if len(queryer.sqls) != 2 {
		t.Fatalf("expected delete and insert bulk execs, got %d", len(queryer.sqls))
	}
	if !strings.Contains(queryer.sqls[0], "DELETE FROM game_companies") {
		t.Fatalf("expected bulk company relation delete, got %q", queryer.sqls[0])
	}
	if !strings.Contains(queryer.sqls[1], "INSERT INTO companies") || !strings.Contains(queryer.sqls[1], "INSERT INTO game_companies") {
		t.Fatalf("expected set-based company upsert and relation insert, got %q", queryer.sqls[1])
	}

	var rows []companyBatchRow
	decodeJSONArg(t, queryer.args[0][0], &rows)
	if len(rows) != 2 {
		t.Fatalf("expected one company row plus one delete sentinel, got %#v", rows)
	}
	g1Company := companyRowFor(rows, "g0000001")
	if g1Company.Name != "Studio" || len(g1Company.OfficialWebsites) != 1 {
		t.Fatalf("expected trimmed first company, got %#v", g1Company)
	}
	g2Company := companyRowFor(rows, "g0000002")
	if g2Company.UniqueID != "g0000002" || g2Company.Name != "" {
		t.Fatalf("expected nil-company sentinel for affected game, got %#v", g2Company)
	}
}

func TestSyncRepoUpsertRatingsBatchUsesAffectedSetForDeletesAndUpserts(t *testing.T) {
	queryer := &recordingSyncQueryer{}
	repo := NewSyncRepo(queryer)

	err := repo.UpsertRatingsBatch(context.Background(), map[string]*model.RatingData{
		"g0000001": &model.RatingData{AverageOverall: 4.5, Count: 9, Histogram: model.RatingHistogram{Score5: 3}},
	}, []string{"g0000001", "g0000002", "g0000001", " "})
	if err != nil {
		t.Fatalf("upsert ratings batch: %v", err)
	}
	if len(queryer.sqls) != 1 {
		t.Fatalf("expected one bulk exec, got %d", len(queryer.sqls))
	}
	if !strings.Contains(queryer.sqls[0], "DELETE FROM game_rating_stats") || !strings.Contains(queryer.sqls[0], "INSERT INTO game_rating_stats") {
		t.Fatalf("expected combined rating delete/upsert SQL, got %q", queryer.sqls[0])
	}

	var rows []ratingBatchRow
	decodeJSONArg(t, queryer.args[0][0], &rows)
	if len(rows) != 2 {
		t.Fatalf("expected deduped affected rows, got %#v", rows)
	}
	if !rows[0].HasRating || rows[0].UniqueID != "g0000001" || rows[0].Count != 9 || rows[0].Histogram == nil || rows[0].Histogram.Score5 != 3 {
		t.Fatalf("expected rating payload for first game, got %#v", rows[0])
	}
	var rawRows []struct {
		Histogram map[string]int `json:"histogram,omitempty"`
	}
	decodeJSONArg(t, queryer.args[0][0], &rawRows)
	if len(rawRows[0].Histogram) != 10 || rawRows[0].Histogram["5"] != 3 {
		t.Fatalf("expected fixed ten-bucket histogram JSON, got %#v", rawRows[0].Histogram)
	}
	if rows[1].HasRating || rows[1].UniqueID != "g0000002" {
		t.Fatalf("expected delete payload for missing rating, got %#v", rows[1])
	}
}

func BenchmarkSyncRepoUpsertRatingsBatch1000(b *testing.B) {
	const size = 1000
	ratings := make(map[string]*model.RatingData, size)
	affected := make([]string, 0, size)
	for i := 0; i < size; i++ {
		uniqueID := "g" + strconv.Itoa(i)
		affected = append(affected, uniqueID)
		ratings[uniqueID] = &model.RatingData{
			AverageOverall: float64(i%50) / 10,
			Count:          i,
			RecStrongNo:    i % 7,
			RecNo:          i % 11,
			RecNeutral:     i % 13,
			RecYes:         i % 17,
			RecStrongYes:   i % 19,
			Histogram: model.RatingHistogram{
				Score1:  i,
				Score2:  i + 1,
				Score3:  i + 2,
				Score4:  i + 3,
				Score5:  i + 4,
				Score6:  i + 5,
				Score7:  i + 6,
				Score8:  i + 7,
				Score9:  i + 8,
				Score10: i + 9,
			},
		}
	}

	repo := NewSyncRepo(discardSyncQueryer{})
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := repo.UpsertRatingsBatch(ctx, ratings, affected); err != nil {
			b.Fatal(err)
		}
	}
}

func TestSyncRepoRefreshSearchTextBatchAggregatesOncePerAffectedSet(t *testing.T) {
	queryer := &recordingSyncQueryer{}
	repo := NewSyncRepo(queryer)

	err := repo.RefreshSearchTextBatch(context.Background(), []string{"g0000001", "g0000002", "g0000001", ""})
	if err != nil {
		t.Fatalf("refresh search text batch: %v", err)
	}
	if len(queryer.sqls) != 1 {
		t.Fatalf("expected one bulk exec, got %d", len(queryer.sqls))
	}
	if strings.Contains(queryer.sqls[0], "(SELECT string_agg") {
		t.Fatalf("expected pre-aggregated search text, got correlated subquery SQL %q", queryer.sqls[0])
	}
	if !strings.Contains(queryer.sqls[0], "WITH affected AS") || !strings.Contains(queryer.sqls[0], "tag_text AS") || !strings.Contains(queryer.sqls[0], "company_text AS") {
		t.Fatalf("expected affected-set aggregate CTEs, got %q", queryer.sqls[0])
	}
	ids, ok := queryer.args[0][0].([]string)
	if !ok || len(ids) != 2 || ids[0] != "g0000001" || ids[1] != "g0000002" {
		t.Fatalf("expected deduped unique IDs, got %#v", queryer.args[0][0])
	}
}

func TestSyncRepoSeenSourcePatchIDsUseRunScopedBulkSQL(t *testing.T) {
	queryer := &recordingSyncQueryer{}
	repo := NewSyncRepo(queryer)
	runID := uuid.New()

	if err := repo.AddSeenSourcePatchIDs(context.Background(), runID, []int{3, 2, 3}); err != nil {
		t.Fatalf("add seen source patch IDs: %v", err)
	}
	deleted, err := repo.MarkDeletedNotSeenByRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("mark deleted not seen by run: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("expected rows affected from command tag, got %d", deleted)
	}
	if err := repo.CleanupSeenSourcePatchIDs(context.Background(), runID); err != nil {
		t.Fatalf("cleanup seen source patch IDs: %v", err)
	}
	if len(queryer.sqls) != 3 {
		t.Fatalf("expected add, mark deleted, and cleanup execs, got %d", len(queryer.sqls))
	}
	if !strings.Contains(queryer.sqls[0], "INSERT INTO sync_run_seen") || !strings.Contains(queryer.sqls[0], "unnest($2::int[])") {
		t.Fatalf("expected bulk seen insert, got %q", queryer.sqls[0])
	}
	ids, ok := queryer.args[0][1].([]int)
	if !ok || len(ids) != 2 || ids[0] != 3 || ids[1] != 2 {
		t.Fatalf("expected deduped source patch IDs, got %#v", queryer.args[0][1])
	}
	if !strings.Contains(queryer.sqls[1], "NOT EXISTS") || !strings.Contains(queryer.sqls[1], "sync_run_seen") {
		t.Fatalf("expected run-scoped unseen delete, got %q", queryer.sqls[1])
	}
	if !strings.Contains(queryer.sqls[2], "DELETE FROM sync_run_seen WHERE run_id = $1") {
		t.Fatalf("expected seen cleanup, got %q", queryer.sqls[2])
	}
}

func decodeJSONArg(t *testing.T, arg any, out any) {
	t.Helper()
	raw, ok := arg.(string)
	if !ok {
		t.Fatalf("expected JSON argument string, got %T", arg)
	}
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		t.Fatalf("decode JSON arg: %v", err)
	}
}

func aliasRowFor(rows []aliasBatchRow, uniqueID string) aliasBatchRow {
	for _, row := range rows {
		if row.UniqueID == uniqueID {
			return row
		}
	}
	return aliasBatchRow{}
}

func tagRowFor(rows []tagBatchRow, uniqueID string) tagBatchRow {
	for _, row := range rows {
		if row.UniqueID == uniqueID {
			return row
		}
	}
	return tagBatchRow{}
}

func companyRowFor(rows []companyBatchRow, uniqueID string) companyBatchRow {
	for _, row := range rows {
		if row.UniqueID == uniqueID {
			return row
		}
	}
	return companyBatchRow{}
}
