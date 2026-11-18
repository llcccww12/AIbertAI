package stats

import (
	"context"
	"strconv"

	"aibertai.local/backend/internal/resource"
	"aibertai.local/backend/internal/stats/sqlc"
	"github.com/jackc/pgx/v5"
)

func realtime(ctx context.Context, tx pgx.Tx, w window) (resource.Object, error) {
	return resource.DecodeObject(sqlc.New(tx).ReportingRealtime(ctx, sqlc.ReportingRealtimeParams{
		FromTime: w.from, UntilTime: w.until,
		WindowMinutes: strconv.FormatFloat(w.until.Sub(w.from).Minutes(), 'f', -1, 64),
	}))
}
