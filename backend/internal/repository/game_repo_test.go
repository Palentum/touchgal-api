package repository

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

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

type gameResourceRow struct {
	sourceResourceID int
	name             string
	introduction     string
	categories       []string
	sizes            []string
	publishedAt      time.Time
}

type gameResourceRows struct {
	rows   []gameResourceRow
	idx    int
	closed bool
	err    error
}

func (r *gameResourceRows) Close()                                       { r.closed = true }
func (r *gameResourceRows) Err() error                                   { return r.err }
func (r *gameResourceRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT") }
func (r *gameResourceRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *gameResourceRows) Next() bool {
	if r.idx >= len(r.rows) {
		r.closed = true
		return false
	}
	r.idx++
	return true
}
func (r *gameResourceRows) Scan(dest ...any) error {
	row := r.rows[r.idx-1]
	*(dest[0].(*int)) = row.sourceResourceID
	*(dest[1].(*string)) = row.name
	*(dest[2].(*string)) = row.introduction
	*(dest[3].(*[]string)) = row.categories
	*(dest[4].(*[]string)) = row.sizes
	*(dest[5].(*time.Time)) = row.publishedAt
	return nil
}
func (r *gameResourceRows) Values() ([]any, error) { return nil, nil }
func (r *gameResourceRows) RawValues() [][]byte    { return nil }
func (r *gameResourceRows) Conn() *pgx.Conn        { return nil }

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

func TestGameRepoSearchSQLVariantsShareRankingTemplate(t *testing.T) {
	normalizePredicate := func(sql string) string {
		return strings.Replace(sql, gameSearchAllowNsfwPredicate, gameSearchSFWPredicate, 1)
	}

	if got := normalizePredicate(gameSearchAllowNsfwSQL); got != gameSearchSFWSQL {
		t.Fatalf("search SQL variants should differ only by content_limit predicate\nsfw:\n%s\nallow nsfw normalized:\n%s", gameSearchSFWSQL, got)
	}
	if got := normalizePredicate(gameSearchCountAllowNsfwSQL); got != gameSearchCountSFWSQL {
		t.Fatalf("search count SQL variants should differ only by content_limit predicate\nsfw:\n%s\nallow nsfw normalized:\n%s", gameSearchCountSFWSQL, got)
	}
}

func TestGameRepoSearchRanksTitleBeforeAliasAndMetadata(t *testing.T) {
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
	if !strings.Contains(mainSQL, "g.content_limit = 'sfw'") {
		t.Fatalf("search SQL must default to SFW-only predicate: %q", queryer.sql)
	}
	if strings.Contains(mainSQL, " join ") || strings.Contains(mainSQL, "distinct") || strings.Contains(mainSQL, "lower(") || strings.Contains(mainSQL, "over()") {
		t.Fatalf("search SQL should avoid joins, distinct, lower(), and window count: %q", queryer.sql)
	}
	for _, want := range []string{
		"g.search_text ilike $1 escape",
		"g.name ilike $2 escape",
		"g.name ilike $3 escape",
		"g.name ilike $1 escape",
		"similarity(g.name, $4)",
		"from game_aliases a",
		"a.game_unique_id = g.unique_id",
		"a.name ilike $1 escape",
		"similarity(a.name, $4)",
		"similarity(g.search_text, $4) as metadata_rank",
		"when title_rank > 0 then 0",
		"when alias_rank > 0 then 1",
		"when title_rank > 0 then title_rank",
		"when alias_rank > 0 then alias_rank",
		"else metadata_rank",
		"name asc, unique_id asc",
	} {
		if !strings.Contains(mainSQL, want) {
			t.Fatalf("search SQL missing %q in %q", want, queryer.sql)
		}
	}
	if len(queryer.args) != 6 || queryer.args[0] != "%Summer%" || queryer.args[1] != "Summer" || queryer.args[2] != "Summer%" || queryer.args[3] != "Summer" || queryer.args[4] != 1 || queryer.args[5] != 0 {
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

func TestGameRepoSearchShortKeywordUsesNgramFilter(t *testing.T) {
	queryer := &recordingGameQueryer{
		rows:     &gameSearchRows{rows: []gameSearchRow{{uniqueID: "abcd1234", name: "魔法ゲーム"}}},
		countRow: gameCountRow{total: 3},
	}
	repo := NewGameRepo(queryer)

	result, err := repo.Search(context.Background(), "魔", 2, 10, false)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(result.Items) != 1 || result.Pagination.Page != 2 || result.Pagination.Total != 3 || result.Pagination.HasMore {
		t.Fatalf("unexpected search result: %#v", result)
	}

	mainSQL := strings.ToLower(queryer.sql)
	for _, want := range []string{
		"from game_search_ngrams n",
		"join games g on g.unique_id = n.game_unique_id",
		"n.gram = lower($4)",
		"g.content_limit = 'sfw'",
	} {
		if !strings.Contains(mainSQL, want) {
			t.Fatalf("short search SQL missing %q in %q", want, queryer.sql)
		}
	}
	if strings.Contains(mainSQL, "g.search_text ilike $1") {
		t.Fatalf("short search must not use trigram ILIKE as the row filter: %q", queryer.sql)
	}
	if len(queryer.args) != 6 || queryer.args[0] != "%魔%" || queryer.args[1] != "魔" || queryer.args[2] != "魔%" || queryer.args[3] != "魔" || queryer.args[4] != 10 || queryer.args[5] != 10 {
		t.Fatalf("unexpected search args: %#v", queryer.args)
	}

	countSQL := strings.ToLower(queryer.countSQL)
	if !strings.Contains(countSQL, "from game_search_ngrams n") || !strings.Contains(countSQL, "n.gram = lower($1)") {
		t.Fatalf("short count SQL must use ngram equality: %q", queryer.countSQL)
	}
	if strings.Contains(countSQL, "search_text ilike") {
		t.Fatalf("short count must not scan search_text ILIKE: %q", queryer.countSQL)
	}
	if len(queryer.countArgs) != 1 || queryer.countArgs[0] != "魔" {
		t.Fatalf("unexpected count args: %#v", queryer.countArgs)
	}
}

func TestGameRepoSearchShortWildcardKeywordStaysLiteral(t *testing.T) {
	queryer := &recordingGameQueryer{
		rows:     &gameSearchRows{},
		countRow: gameCountRow{total: 0},
	}
	repo := NewGameRepo(queryer)

	if _, err := repo.Search(context.Background(), "%", 1, 20, false); err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(queryer.args) != 6 || queryer.args[0] != `%\%%` || queryer.args[1] != `\%` || queryer.args[2] != `\%%` || queryer.args[3] != "%" {
		t.Fatalf("wildcard keyword should be escaped for LIKE ranking and raw for ngram equality: %#v", queryer.args)
	}
	if len(queryer.countArgs) != 1 || queryer.countArgs[0] != "%" {
		t.Fatalf("short wildcard count must use literal ngram equality arg, got %#v", queryer.countArgs)
	}
	if strings.Contains(strings.ToLower(queryer.countSQL), "search_text ilike") {
		t.Fatalf("short wildcard count must not use wildcard LIKE scan: %q", queryer.countSQL)
	}
}

func TestGameRepoSearchShortKeywordAllowNsfwUsesOptInPredicate(t *testing.T) {
	queryer := &recordingGameQueryer{
		rows:     &gameSearchRows{},
		countRow: gameCountRow{total: 0},
	}
	repo := NewGameRepo(queryer)

	if _, err := repo.Search(context.Background(), "ab", 1, 20, true); err != nil {
		t.Fatalf("search: %v", err)
	}

	mainSQL := strings.ToLower(queryer.sql)
	if !strings.Contains(mainSQL, "from game_search_ngrams n") || !strings.Contains(mainSQL, "content_limit in ('sfw', 'nsfw')") {
		t.Fatalf("short allow-NSFW search must use ngram opt-in predicate: %q", queryer.sql)
	}
	countSQL := strings.ToLower(queryer.countSQL)
	if !strings.Contains(countSQL, "from game_search_ngrams n") || !strings.Contains(countSQL, "content_limit in ('sfw', 'nsfw')") {
		t.Fatalf("short allow-NSFW count must use ngram opt-in predicate: %q", queryer.countSQL)
	}
	if len(queryer.countArgs) != 1 || queryer.countArgs[0] != "ab" {
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
	if len(queryer.args) != 6 || queryer.args[0] != "%Summer%" || queryer.args[1] != "Summer" || queryer.args[2] != "Summer%" || queryer.args[3] != "Summer" || queryer.args[4] != 20 || queryer.args[5] != 0 {
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
		t.Fatalf("LIKE contains pattern did not escape wildcards: %#v", queryer.args[0])
	}
	if queryer.args[1] != `100\%\_\\` {
		t.Fatalf("LIKE exact pattern did not escape wildcards: %#v", queryer.args[1])
	}
	if queryer.args[2] != `100\%\_\\%` {
		t.Fatalf("LIKE prefix pattern did not escape wildcards: %#v", queryer.args[2])
	}
	if queryer.args[3] != `100%_\` {
		t.Fatalf("raw relevance keyword should not be LIKE-escaped: %#v", queryer.args[3])
	}
	if queryer.countArgs[0] != queryer.args[0] {
		t.Fatalf("count query must reuse escaped contains pattern: search=%#v count=%#v", queryer.args[0], queryer.countArgs[0])
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

func TestGameRepoResourcesDefaultSFWAndDeepLink(t *testing.T) {
	queryer := &recordingGameQueryer{
		rows: &gameResourceRows{rows: []gameResourceRow{{
			sourceResourceID: 42,
			name:             "Resource",
			introduction:     "Intro",
			categories:       []string{"Galgame"},
			sizes:            []string{"4.2GB"},
			publishedAt:      time.Date(2024, 5, 30, 10, 0, 0, 0, time.UTC),
		}}},
		countRow: gameCountRow{total: 1},
	}
	repo := NewGameRepo(queryer)

	result, err := repo.Resources(context.Background(), "abcd1234", "https://www.touchgal.ink/", false)
	if err != nil {
		t.Fatalf("resources: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("unexpected resources result: %#v", result)
	}
	if result.Items[0].DeepLink != "https://www.touchgal.ink/abcd1234?tab=resources&resourceId=42&resourceSection=galgame" {
		t.Fatalf("unexpected deep link: %q", result.Items[0].DeepLink)
	}

	visibleSQL := strings.ToLower(queryer.countSQL)
	if !strings.Contains(visibleSQL, "select 1") || !strings.Contains(visibleSQL, "g.content_limit = 'sfw'") {
		t.Fatalf("visibility SQL must check SFW game visibility: %q", queryer.countSQL)
	}
	resourceSQL := strings.ToLower(queryer.sql)
	for _, want := range []string{"from game_resources", "join games g", "gr.resource_type = $2", "g.content_limit = 'sfw'"} {
		if !strings.Contains(resourceSQL, want) {
			t.Fatalf("resource SQL missing %q: %q", want, queryer.sql)
		}
	}
	if len(queryer.args) != 2 || queryer.args[0] != "abcd1234" || queryer.args[1] != model.ResourceTypeResource {
		t.Fatalf("unexpected resource args: %#v", queryer.args)
	}
}

func TestGameRepoPatchesAllowNsfwAndDeepLink(t *testing.T) {
	queryer := &recordingGameQueryer{
		rows: &gameResourceRows{rows: []gameResourceRow{{
			sourceResourceID: 43,
			name:             "Patch",
			introduction:     "Intro",
			categories:       []string{"Patch"},
			sizes:            []string{"512MB"},
			publishedAt:      time.Date(2024, 5, 31, 10, 0, 0, 0, time.UTC),
		}}},
		countRow: gameCountRow{total: 1},
	}
	repo := NewGameRepo(queryer)

	result, err := repo.Patches(context.Background(), "abcd1234", "https://www.touchgal.ink", true)
	if err != nil {
		t.Fatalf("patches: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].DeepLink != "https://www.touchgal.ink/abcd1234?tab=resources&resourceId=43&resourceSection=patch" {
		t.Fatalf("unexpected patches result: %#v", result)
	}

	visibleSQL := strings.ToLower(queryer.countSQL)
	if !strings.Contains(visibleSQL, "g.content_limit in ('sfw', 'nsfw')") {
		t.Fatalf("visibility SQL must allow SFW and NSFW rows when opted in: %q", queryer.countSQL)
	}
	resourceSQL := strings.ToLower(queryer.sql)
	if !strings.Contains(resourceSQL, "g.content_limit in ('sfw', 'nsfw')") {
		t.Fatalf("resource SQL must allow SFW and NSFW rows when opted in: %q", queryer.sql)
	}
	if len(queryer.args) != 2 || queryer.args[1] != model.ResourceTypePatch {
		t.Fatalf("unexpected patch args: %#v", queryer.args)
	}
}

func TestGameRepoResourcesMapsInvisibleGameToNotFound(t *testing.T) {
	queryer := &recordingGameQueryer{countRow: gameCountRow{err: pgx.ErrNoRows}}
	repo := NewGameRepo(queryer)

	_, err := repo.Resources(context.Background(), "abcd1234", "https://www.touchgal.ink", false)
	if err != model.ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
	if queryer.sql != "" {
		t.Fatalf("resource query should not run after invisible game check: %q", queryer.sql)
	}
}

func TestGameRepoResourcesEmptyListEncodesAsArray(t *testing.T) {
	queryer := &recordingGameQueryer{
		rows:     &gameResourceRows{},
		countRow: gameCountRow{total: 1},
	}
	repo := NewGameRepo(queryer)

	result, err := repo.Resources(context.Background(), "abcd1234", "https://www.touchgal.ink", false)
	if err != nil {
		t.Fatalf("resources: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected empty resource list, got %#v", result.Items)
	}
	if result.Items == nil {
		t.Fatal("empty resource list must encode as []")
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal resource list: %v", err)
	}
	if string(encoded) != `{"items":[]}` {
		t.Fatalf("expected empty JSON array, got %s", encoded)
	}
}
