package teamcapacity

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"aibertai.local/backend/internal/teamcapacity/sqlc"
	"aibertai.local/backend/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type status struct {
	MaxMembers    *int32 `json:"max_members"`
	ActiveMembers int32  `json:"active_members"`
}

type Service struct {
	queries *sqlc.Queries
}

func New(db *pgxpool.Pool) *Service { return &Service{queries: sqlc.New(db)} }

func (s *Service) RegisterAdmin(router chi.Router) {
	router.Get("/team-capacity", s.read)
	router.Put("/team-capacity", s.update)
}

func (s *Service) read(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	value, err := s.queries.GetCapacity(r.Context())
	if err != nil {
		respondError(w, r, http.StatusInternalServerError, "server_error", "读取团队人数上限失败")
		return
	}
	web.JSON(r.Context(), w, http.StatusOK, status(value))
}

func (s *Service) update(w http.ResponseWriter, r *http.Request) {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaType != "application/json" {
		respondError(w, r, http.StatusUnsupportedMediaType, "invalid_request", "请求必须使用 JSON")
		return
	}
	var input struct {
		MaxMembers json.RawMessage `json:"max_members"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	decoder.DisallowUnknownFields()
	var limit *int32
	if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF ||
		len(input.MaxMembers) == 0 || json.Unmarshal(input.MaxMembers, &limit) != nil ||
		(limit != nil && *limit < 1) {
		respondError(w, r, http.StatusBadRequest, "invalid_request", "人数上限必须为正整数或不限人数")
		return
	}
	value, err := s.queries.SetCapacity(r.Context(), limit)
	if errors.Is(err, pgx.ErrNoRows) {
		respondError(w, r, http.StatusConflict, "team_capacity_below_usage", "人数上限不能低于当前启用人数")
		return
	}
	if err != nil {
		respondError(w, r, http.StatusInternalServerError, "server_error", "保存团队人数上限失败")
		return
	}
	web.JSON(r.Context(), w, http.StatusOK, status(value))
}

func respondError(w http.ResponseWriter, r *http.Request, code int, kind, message string) {
	web.JSON(r.Context(), w, code, map[string]any{"error": web.Body(r.Context(), kind, message)})
}
