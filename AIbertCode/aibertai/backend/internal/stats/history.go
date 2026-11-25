package stats

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"aibertai.local/backend/internal/resource"
	"aibertai.local/backend/internal/stats/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

const historySummaryDefaultWindow = 30 * 24 * time.Hour

type historyFilter struct {
	title, user      string
	userID, expertID *string
	failedOnly       bool
	from, until      *time.Time
}

// parseHistoryFilter 解析会话历史列表与汇总共用的筛选条件；失败时已写入错误响应。
func parseHistoryFilter(w http.ResponseWriter, r *http.Request) (historyFilter, bool) {
	q := r.URL.Query()
	f := historyFilter{title: strings.TrimSpace(q.Get("task")), user: strings.TrimSpace(q.Get("user"))}
	if len(f.title) > 200 || len(f.user) > 200 {
		resource.Fail(r.Context(), w, resource.Invalid("搜索条件过长"))
		return f, false
	}
	for key, target := range map[string]**string{"user_id": &f.userID, "expert_id": &f.expertID} {
		if value := strings.TrimSpace(q.Get(key)); value != "" {
			var parsed pgtype.UUID
			if err := parsed.Scan(value); err != nil || !parsed.Valid {
				message := "用户 ID 无效"
				if key == "expert_id" {
					message = "专家 ID 无效"
				}
				resource.Fail(r.Context(), w, resource.Invalid(message))
				return f, false
			}
			*target = &value
		}
	}
	switch q.Get("status") {
	case "", "all":
	case "failed":
		f.failedOnly = true
	default:
		resource.Fail(r.Context(), w, resource.Invalid("状态筛选无效"))
		return f, false
	}
	for key, target := range map[string]**time.Time{"from": &f.from, "until": &f.until} {
		if value := q.Get(key); value != "" {
			at, err := time.Parse(time.RFC3339, value)
			if err != nil {
				slog.WarnContext(r.Context(), "操作失败", "module", "stats", "operation", "history", "error", err)
				resource.Fail(r.Context(), w, resource.Invalid("筛选时间需使用带时区的 ISO 格式"))
				return f, false
			}
			*target = &at
		}
	}
	if f.from != nil && f.until != nil && !f.from.Before(*f.until) {
		resource.Fail(r.Context(), w, resource.Invalid("结束时间必须晚于开始时间"))
		return f, false
	}
	return f, true
}

func (s *Service) history(w http.ResponseWriter, r *http.Request) {
	f, ok := parseHistoryFilter(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, size := 1, 20
	for key, target := range map[string]*int{"page": &page, "page_size": &size} {
		if value := q.Get(key); value != "" {
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > 1_000_000 || (key == "page_size" && n > 500) {
				if err != nil {
					slog.WarnContext(r.Context(), "操作失败", "module", "stats", "operation", "history", "error", err)
				}
				resource.Fail(r.Context(), w, resource.Invalid("分页参数无效"))
				return
			}
			*target = n
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	items, err := resource.DecodeObjects(sqlc.New(s.pool).ListHistory(ctx, sqlc.ListHistoryParams{
		TitleQuery: f.title,
		UserQuery:  f.user,
		UserID:     f.userID,
		ExpertID:   f.expertID,
		FailedOnly: f.failedOnly,
		FromTime:   f.from,
		UntilTime:  f.until,
		Limit:      int32(size + 1),
		Offset:     int32((page - 1) * size),
	}))
	if err != nil {
		resource.Fail(r.Context(), w, err)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	hasNext := len(items) > size
	if hasNext {
		items = items[:size]
	}
	resource.JSON(r.Context(), w, http.StatusOK, resource.Object{"items": items, "has_next": hasNext, "page": page, "page_size": size})
}

func (s *Service) historySummary(w http.ResponseWriter, r *http.Request) {
	f, ok := parseHistoryFilter(w, r)
	if !ok {
		return
	}
	// 未指定开始时间时只汇总最近 30 天，避免整库聚合。
	summaryFrom := time.Now().Add(-historySummaryDefaultWindow)
	if f.from != nil {
		summaryFrom = *f.from
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	summary, err := resource.DecodeObject(sqlc.New(s.pool).HistorySummary(ctx, sqlc.HistorySummaryParams{
		SummaryFrom: summaryFrom,
		TitleQuery:  f.title,
		UserQuery:   f.user,
		UserID:      f.userID,
		ExpertID:    f.expertID,
		FailedOnly:  f.failedOnly,
		FromTime:    f.from,
		UntilTime:   f.until,
	}))
	if err != nil {
		resource.Fail(r.Context(), w, err)
		return
	}
	summary["window_from"] = summaryFrom.UTC()
	w.Header().Set("Cache-Control", "private, no-store")
	resource.JSON(r.Context(), w, http.StatusOK, summary)
}
