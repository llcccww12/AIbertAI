package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"aibertai.local/backend/internal/envconfig"
	"aibertai.local/backend/internal/web"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"aibertai.local/backend/logging"

	"aibertai.local/backend/internal/audit/sqlc"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewService(pool *pgxpool.Pool, logger *slog.Logger) *Service {
	return &Service{pool: pool, logger: logger}
}

// 审计表只增不减：没有保留窗口时，分页查询和备份都会随年份线性变慢变大。
// 默认留一年，运维可以用 AIBERTAI_AUDIT_RETENTION_DAYS 调整，设 0 表示永不清理。
const defaultAuditRetention = 365 * 24 * time.Hour

func auditRetention() time.Duration {
	raw := strings.TrimSpace(envconfig.Getenv("AIBERTAI_AUDIT_RETENTION_DAYS"))
	if raw == "" {
		return defaultAuditRetention
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < 0 {
		slog.Warn("审计保留天数配置无效，沿用默认值", "module", "audit", "operation", "auditRetention", "value", raw)
		return defaultAuditRetention
	}
	return time.Duration(days) * 24 * time.Hour
}

// Purge 清理超过保留期的审计记录，返回删除行数。
func (s *Service) Purge(ctx context.Context) (int64, error) {
	retention := auditRetention()
	if retention <= 0 {
		return 0, nil
	}
	tag, err := s.pool.Exec(ctx, "DELETE FROM audits WHERE occurred_at < $1", time.Now().UTC().Add(-retention))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Run 每天清理一次过期审计记录，直到 ctx 结束。
func (s *Service) Run(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			deleted, err := s.Purge(ctx)
			cancel()
			if err != nil && ctx.Err() == nil {
				logging.Event(ctx, s.logger, slog.LevelError, "audit.operation.error", "清理过期审计记录失败", "module", "audit", "operation", "Purge", "error", err)
				continue
			}
			if deleted > 0 {
				s.logger.Info("已清理过期审计记录", "module", "audit", "operation", "Purge", "deleted", deleted)
			}
		}
	}
}

type Actor struct{ ID, Name, Email string }

type Event struct {
	ActorID, Action, Category, TargetType, TargetID string
	Params                                          any
}

// Write 与业务使用同一事务；没有 HTTP 上下文时仍可记录独立业务事件。
func Write(ctx context.Context, db sqlc.DBTX, event Event) error {
	data, err := params(event.Params)
	if err != nil {
		return err
	}
	in := sqlc.CreateParams{
		ActorUserID: optional(event.ActorID), Action: event.Action, Category: event.Category,
		TargetType: optional(event.TargetType), TargetID: uuid(event.TargetID),
		RequestParams: data, Result: "success", OccurredAt: time.Now().UTC(),
	}
	state, _ := ctx.Value(requestKey{}).(*requestState)
	if state != nil && state.actor.ID == event.ActorID {
		in.ActorName, in.ActorEmail = state.actor.Name, optional(state.actor.Email)
		in.RequestID, in.SourceIp, in.UserAgent = optional(state.id), state.ip, optional(state.agent)
		in.OccurredAt = state.at
	}
	if in.ActorName == "" {
		actor, err := sqlc.New(db).Actor(ctx, event.ActorID)
		if err != nil {
			return err
		}
		in.ActorName, in.ActorEmail = actor.Name, optional(actor.Email)
	}
	return sqlc.New(db).Create(ctx, in)
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return new(value)
}

func uuid(value string) *string {
	if value == "" {
		return nil
	}
	var id pgtype.UUID
	if err := id.Scan(value); err != nil || !id.Valid {
		if err != nil {
			slog.Warn("操作失败", "module", "audit", "operation", "uuid", "error", err)
		}
		return nil
	}
	return new(value)
}

func (s *Service) RegisterAdmin(router chi.Router) {
	router.Get("/audits", s.list)
}

func filters(r *http.Request) (sqlc.PageParams, int, error) {
	q := r.URL.Query()
	in := sqlc.PageParams{Category: q.Get("category"), Result: q.Get("result"), Actor: strings.TrimSpace(q.Get("actor")), Ip: strings.TrimSpace(q.Get("ip")), Params: strings.TrimSpace(q.Get("params"))}
	invalid := errors.New("审计筛选参数无效")
	if value := strings.TrimSpace(q.Get("actor_user_id")); value != "" {
		in.ActorUserID = uuid(value)
		if in.ActorUserID == nil {
			return in, 0, invalid
		}
	}
	if !slices.Contains([]string{"", "model", "identity", "security", "settings", "resource", "billing"}, in.Category) || !slices.Contains([]string{"", "success", "failed"}, in.Result) {
		return in, 0, invalid
	}
	for _, value := range []string{in.Actor, in.Ip, in.Params} {
		if len(value) > 512 || strings.ContainsRune(value, 0) {
			return in, 0, invalid
		}
	}
	page, size := 1, 20
	var err error
	if q.Has("page") {
		page, err = strconv.Atoi(q.Get("page"))
		if err != nil || page < 1 || page > 1000000 {
			if err != nil {
				slog.WarnContext(r.Context(), "操作失败", "module", "audit", "operation", "filters", "error", err)
			}
			return in, 0, invalid
		}
	}
	if q.Has("page_size") {
		size, err = strconv.Atoi(q.Get("page_size"))
		if err != nil || size < 1 || size > 500 {
			if err != nil {
				slog.WarnContext(r.Context(), "操作失败", "module", "audit", "operation", "filters", "error", err)
			}
			return in, 0, invalid
		}
	}
	in.PageSize, in.PageOffset = int32(size), int32((page-1)*size)
	for key, target := range map[string]**time.Time{"since": &in.Since, "until": &in.Until} {
		if q.Has(key) {
			at, err := time.Parse(time.RFC3339, q.Get(key))
			if err != nil {
				slog.WarnContext(r.Context(), "操作失败", "module", "audit", "operation", "filters", "error", err)
				return in, 0, invalid
			}
			*target = &at
		}
	}
	if in.Since != nil && in.Until != nil && !in.Since.Before(*in.Until) {
		return in, 0, invalid
	}
	return in, page, nil
}

func (s *Service) list(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	in, page, err := filters(r)
	if err != nil {
		failure(r.Context(), w, 400, "invalid_request", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	data, err := sqlc.New(s.pool).Page(ctx, in)
	if err != nil {
		logging.Event(r.Context(), s.logger, slog.LevelError, "audit.operation.error", "读取操作审计失败", "module", "audit", "operation", "list", "request_id", logging.RequestID(r.Context()), "error", err)
		failure(r.Context(), w, 500, "server_error", "读取操作审计失败")
		return
	}
	var out struct {
		Items    []map[string]any `json:"items"`
		Total    int64            `json:"total"`
		Page     int              `json:"page"`
		PageSize int32            `json:"page_size"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err = decoder.Decode(&out); err != nil {
		logging.Event(r.Context(), s.logger, slog.LevelError, "audit.operation.error", "读取操作审计失败", "module", "audit", "operation", "list", "request_id", logging.RequestID(r.Context()), "error", err)
		failure(r.Context(), w, 500, "server_error", "读取操作审计失败")
		return
	}
	for _, item := range out.Items {
		item["request_params"] = sanitize(item["request_params"], 0)
	}
	out.Page, out.PageSize = page, in.PageSize
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		logging.Event(r.Context(), s.logger, slog.LevelError, "audit.operation.error", "操作审计响应写入失败", "module", "audit", "operation", "list", "request_id", logging.RequestID(r.Context()), "error", err)
	}
}

func failure(ctx context.Context, w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{"error": web.Body(ctx, code, message)}); err != nil {
		logging.Event(ctx, slog.Default(), slog.LevelError, "audit.operation.error", "操作审计错误响应写入失败", "module", "audit", "operation", "failure", "request_id", logging.RequestID(ctx), "status", status, "code", code, "error", err)
	}
}
