package app

import (
	"context"
	"log/slog"
	"net/http"

	"aibertai.local/backend/internal/config"
	"aibertai.local/backend/internal/testmember"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newMemberTestHandler(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool, cfg config.Config) (http.Handler, error) {
	return newApplicationHandlerWithMembers(ctx, logger, pool, cfg, testmember.Users(pool), testmember.Groups(), nil)
}
