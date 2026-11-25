package rule

import (
	"context"

	"aibertai.local/backend/migrations"

	"github.com/jackc/pgx/v5"
)

type beginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

// SeedPresets 在库里从未出现过系统规则时写入预置规则，归属最早的有效管理员。
// 已删除的系统规则仍会保留记录，因此管理员删掉预置规则后不会被重新补回。
func SeedPresets(ctx context.Context, db beginner) error {
	return pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext('aibertai.rule_presets'))"); err != nil {
			return err
		}
		var seeded bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM rules WHERE ownership_type = 'system')").Scan(&seeded); err != nil {
			return err
		}
		if seeded {
			return nil
		}
		_, err := tx.Exec(ctx, migrations.RulePresets)
		return err
	})
}
