package stats

import (
	"context"
	"net/http"
	"time"

	"aibertai.local/backend/internal/resource"
	"aibertai.local/backend/internal/stats/sqlc"
	"github.com/jackc/pgx/v5"
)

func (s *Service) membersOverview(w http.ResponseWriter, r *http.Request) {
	if len(r.URL.Query()) != 0 {
		resource.Fail(r.Context(), w, resource.Invalid("成员概览不支持查询参数"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		resource.Fail(r.Context(), w, err)
		return
	}
	defer rollbackStats(ctx, tx, "members_overview")
	out, err := resource.DecodeObject(sqlc.New(tx).ReportingMembersOverview(ctx, s.now().UTC()))
	if err != nil {
		resource.Fail(r.Context(), w, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	resource.JSON(r.Context(), w, http.StatusOK, out)
}
