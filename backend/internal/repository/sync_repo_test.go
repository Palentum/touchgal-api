package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type recordingSyncQueryer struct {
	sqls []string
	args [][]any
}

func (q *recordingSyncQueryer) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	q.sqls = append(q.sqls, sql)
	q.args = append(q.args, arguments)
	return pgconn.NewCommandTag("OK"), nil
}

func (q *recordingSyncQueryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	panic("unexpected query")
}

func (q *recordingSyncQueryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	panic("unexpected query row")
}

func TestSyncRepoReplaceAliasesBatchesCleanUniqueAliases(t *testing.T) {
	queryer := &recordingSyncQueryer{}
	repo := NewSyncRepo(queryer)

	err := repo.ReplaceAliases(context.Background(), "g0000001", []string{"  Alpha  ", "Alpha", "", "Beta"})
	if err != nil {
		t.Fatalf("replace aliases: %v", err)
	}
	if len(queryer.sqls) != 2 {
		t.Fatalf("expected delete plus one bulk insert, got %d execs", len(queryer.sqls))
	}
	if !strings.Contains(queryer.sqls[0], "DELETE FROM game_aliases") {
		t.Fatalf("expected first exec to delete aliases, got %q", queryer.sqls[0])
	}
	if !strings.Contains(queryer.sqls[1], "unnest($2::text[])") {
		t.Fatalf("expected bulk alias insert via unnest, got %q", queryer.sqls[1])
	}
	if len(queryer.args[1]) != 2 || queryer.args[1][0] != "g0000001" {
		t.Fatalf("unexpected bulk insert args: %#v", queryer.args[1])
	}
	aliases, ok := queryer.args[1][1].([]string)
	if !ok {
		t.Fatalf("expected alias arg to be []string, got %T", queryer.args[1][1])
	}
	if len(aliases) != 2 || aliases[0] != "Alpha" || aliases[1] != "Beta" {
		t.Fatalf("expected trimmed unique aliases, got %#v", aliases)
	}
}

func TestSyncRepoReplaceAliasesSkipsEmptyBulkInsert(t *testing.T) {
	queryer := &recordingSyncQueryer{}
	repo := NewSyncRepo(queryer)

	err := repo.ReplaceAliases(context.Background(), "g0000001", []string{"", "   "})
	if err != nil {
		t.Fatalf("replace aliases: %v", err)
	}
	if len(queryer.sqls) != 1 {
		t.Fatalf("expected only delete for empty aliases, got %d execs", len(queryer.sqls))
	}
	if !strings.Contains(queryer.sqls[0], "DELETE FROM game_aliases") {
		t.Fatalf("expected delete aliases SQL, got %q", queryer.sqls[0])
	}
}
