package rule

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aibertai.local/backend/internal/resource"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSeedPresetsAfterInitialAdmin(t *testing.T) {
	dsn := os.Getenv("MONKEYAI_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("设置 MONKEYAI_TEST_DATABASE_URL 运行预置规则集成测试")
	}
	ctx := t.Context()
	root, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(root.Close)
	schema := "test_rule_presets_" + strings.ReplaceAll(resource.ID(), "-", "")
	if _, err := root.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := root.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	paths, err := filepath.Glob("../../migrations/*.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(data)); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
	active := func() int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM rules WHERE ownership_type='system' AND deleted_at IS NULL").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	seed := func() {
		t.Helper()
		if err := SeedPresets(ctx, pool); err != nil {
			t.Fatal(err)
		}
	}
	if seed(); active() != 0 {
		t.Fatal("没有管理员时不应写入预置规则")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO users(name,email,role) VALUES('管理员','admin@example.com','admin')`); err != nil {
		t.Fatal(err)
	}
	if seed(); active() != 18 {
		t.Fatalf("首个管理员创建后应写入 18 条预置规则，实际 %d", active())
	}
	if _, err := pool.Exec(ctx, `UPDATE rules SET deleted_at=now() WHERE name='简洁回复'`); err != nil {
		t.Fatal(err)
	}
	if seed(); active() != 17 {
		t.Fatalf("管理员删除的预置规则不应被补回，实际 %d", active())
	}
}
