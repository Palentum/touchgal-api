package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/touchgal/developer/backend/internal/model"
)

type recordingGameQueryer struct {
	sql       string
	args      []any
	rows      pgx.Rows
	queryErr  error
	countSQL  string
	countArgs []any
	countRow  pgx.Row
}

func (q *recordingGameQueryer) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	panic("unexpected exec")
}

func (q *recordingGameQueryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	q.sql = sql
	q.args = args
	return q.rows, q.queryErr
}

func (q *recordingGameQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	q.countSQL = sql
	q.countArgs = args
	return q.countRow
}

type gameSearchRow struct {
	uniqueID string
	name     string
}

type gameSearchRows struct {
	rows   []gameSearchRow
	idx    int
	closed bool
	err    error
}

func (r *gameSearchRows) Close()                                       { r.closed = true }
func (r *gameSearchRows) Err() error                                   { return r.err }
func (r *gameSearchRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT") }
func (r *gameSearchRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *gameSearchRows) Next() bool {
	if r.idx >= len(r.rows) {
		r.closed = true
		return false
	}
	r.idx++
	return true
}
func (r *gameSearchRows) Scan(dest ...any) error {
	row := r.rows[r.idx-1]
	*(dest[0].(*string)) = row.uniqueID
	*(dest[1].(*string)) = row.name
	return nil
}
func (r *gameSearchRows) Values() ([]any, error) { return nil, nil }
func (r *gameSearchRows) RawValues() [][]byte    { return nil }
func (r *gameSearchRows) Conn() *pgx.Conn        { return nil }

type gameCountRow struct {
	total int
	err   error
}

func (r gameCountRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*int)) = r.total
	return nil
}

func TestGameRepoSearchUsesSearchTextWithoutJoinDistinctOrWindowCount(t *testing.T) {
	queryer := &recordingGameQueryer{
		rows:     &gameSearchRows{rows: []gameSearchRow{{uniqueID: "abcd1234", name: "Summer"}}},
		countRow: gameCountRow{total: 2},
	}
	repo := NewGameRepo(queryer)

	result, err := repo.Search(context.Background(), "Summer", 1, 1, false)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].UniqueID != "abcd1234" || result.Pagination.Total != 2 || !result.Pagination.HasMore {
		t.Fatalf("unexpected search result: %#v", result)
	}

	mainSQL := strings.ToLower(queryer.sql)
	if !strings.Contains(mainSQL, "content_limit = 'sfw'") {
		t.Fatalf("search SQL must default to SFW-only predicate: %q", queryer.sql)
	}
	if strings.Contains(mainSQL, " join ") || strings.Contains(mainSQL, "distinct") || strings.Contains(mainSQL, "lower(") || strings.Contains(mainSQL, "over()") {
		t.Fatalf("search SQL should avoid joins, distinct, lower(), and window count: %q", queryer.sql)
	}
	if !strings.Contains(mainSQL, "search_text ilike $1 escape") {
		t.Fatalf("search SQL must use indexed search_text ILIKE predicate with escaping: %q", queryer.sql)
	}
	if len(queryer.args) != 3 || queryer.args[0] != "%Summer%" || queryer.args[1] != 1 || queryer.args[2] != 0 {
		t.Fatalf("unexpected search args: %#v", queryer.args)
	}

	countSQL := strings.ToLower(queryer.countSQL)
	if !strings.Contains(countSQL, "content_limit = 'sfw'") {
		t.Fatalf("count SQL must default to SFW-only predicate: %q", queryer.countSQL)
	}
	if !strings.Contains(countSQL, "select count(*)") || strings.Contains(countSQL, "over()") || strings.Contains(countSQL, " join ") {
		t.Fatalf("count SQL should be a separate single-table count: %q", queryer.countSQL)
	}
	if len(queryer.countArgs) != 1 || queryer.countArgs[0] != "%Summer%" {
		t.Fatalf("unexpected count args: %#v", queryer.countArgs)
	}
}

func TestGameRepoSearchAllowNsfwUsesOptInPredicate(t *testing.T) {
	queryer := &recordingGameQueryer{
		rows:     &gameSearchRows{rows: []gameSearchRow{{uniqueID: "abcd1234", name: "Summer"}}},
		countRow: gameCountRow{total: 1},
	}
	repo := NewGameRepo(queryer)

	if _, err := repo.Search(context.Background(), "Summer", 1, 20, true); err != nil {
		t.Fatalf("search: %v", err)
	}

	mainSQL := strings.ToLower(queryer.sql)
	if !strings.Contains(mainSQL, "content_limit in ('sfw', 'nsfw')") {
		t.Fatalf("search SQL must allow only SFW and NSFW rows when opted in: %q", queryer.sql)
	}
	if strings.Contains(mainSQL, "content_limit = 'sfw'") {
		t.Fatalf("allow-NSFW search SQL should not keep SFW-only predicate: %q", queryer.sql)
	}
	if len(queryer.args) != 3 || queryer.args[0] != "%Summer%" || queryer.args[1] != 20 || queryer.args[2] != 0 {
		t.Fatalf("unexpected search args: %#v", queryer.args)
	}

	countSQL := strings.ToLower(queryer.countSQL)
	if !strings.Contains(countSQL, "content_limit in ('sfw', 'nsfw')") {
		t.Fatalf("count SQL must allow only SFW and NSFW rows when opted in: %q", queryer.countSQL)
	}
	if strings.Contains(countSQL, "content_limit = 'sfw'") {
		t.Fatalf("allow-NSFW count SQL should not keep SFW-only predicate: %q", queryer.countSQL)
	}
	if len(queryer.countArgs) != 1 || queryer.countArgs[0] != "%Summer%" {
		t.Fatalf("unexpected count args: %#v", queryer.countArgs)
	}
}

func TestGameRepoSearchEscapesLikeWildcards(t *testing.T) {
	queryer := &recordingGameQueryer{
		rows:     &gameSearchRows{},
		countRow: gameCountRow{total: 0},
	}
	repo := NewGameRepo(queryer)

	if _, err := repo.Search(context.Background(), `100%_\`, 1, 20, false); err != nil {
		t.Fatalf("search: %v", err)
	}
	if queryer.args[0] != `%100\%\_\\%` {
		t.Fatalf("LIKE wildcards were not escaped: %#v", queryer.args[0])
	}
	if queryer.countArgs[0] != queryer.args[0] {
		t.Fatalf("count query must reuse escaped pattern: search=%#v count=%#v", queryer.args[0], queryer.countArgs[0])
	}
}

func TestGameRepoDetailAllowNsfwUsesOptInPredicate(t *testing.T) {
	queryer := &recordingGameQueryer{countRow: gameCountRow{err: pgx.ErrNoRows}}
	repo := NewGameRepo(queryer)

	_, err := repo.Detail(context.Background(), "abcd1234", "https://www.touchgal.ink", true)
	if err != model.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
	detailSQL := strings.ToLower(queryer.countSQL)
	if !strings.Contains(detailSQL, "g.content_limit in ('sfw', 'nsfw')") {
		t.Fatalf("detail SQL must allow only SFW and NSFW rows when opted in: %q", queryer.countSQL)
	}
	if strings.Contains(detailSQL, "g.content_limit = 'sfw'") {
		t.Fatalf("allow-NSFW detail SQL should not keep SFW-only predicate: %q", queryer.countSQL)
	}
}

func TestGameRepoDetailDefaultsToSFWPredicate(t *testing.T) {
	queryer := &recordingGameQueryer{countRow: gameCountRow{err: pgx.ErrNoRows}}
	repo := NewGameRepo(queryer)

	_, err := repo.Detail(context.Background(), "abcd1234", "https://www.touchgal.ink", false)
	if err != model.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
	detailSQL := strings.ToLower(queryer.countSQL)
	if !strings.Contains(detailSQL, "g.content_limit = 'sfw'") {
		t.Fatalf("detail SQL must default to SFW-only predicate: %q", queryer.countSQL)
	}
}
