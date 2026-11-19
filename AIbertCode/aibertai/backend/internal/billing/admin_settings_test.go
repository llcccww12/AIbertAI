package billing

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	auditlog "aibertai.local/backend/internal/audit"
	"aibertai.local/backend/internal/identity"
	"aibertai.local/backend/internal/setting"
	"aibertai.local/backend/logging"
	"github.com/go-chi/chi/v5"
)

func billingAdmin(t *testing.T, s *Service, user string) func(string, string, any, int) []byte {
	t.Helper()
	token := "billing-settings-test"
	hash := sha256.Sum256([]byte(token))
	_, err := s.pool.Exec(t.Context(), `INSERT INTO browser_sessions(token_hash,user_id,authentication_method,expires_at) VALUES($1,$2,'password',now()+interval '1 hour')`, hex.EncodeToString(hash[:]), user)
	if err != nil {
		t.Fatal(err)
	}
	settings := setting.NewService(setting.NewPostgres(s.pool))
	identities := identity.NewService(s.pool, settings, "http://localhost")
	router := chi.NewRouter()
	router.Use(logging.HTTP(slog.New(slog.NewTextHandler(io.Discard, nil))))
	router.Use(identities.RequireAdmin)
	router.Use(auditlog.NewService(s.pool, slog.New(slog.NewTextHandler(io.Discard, nil))).Middleware(func(r *http.Request) auditlog.Actor {
		u, _ := identity.UserFromContext(r.Context())
		return auditlog.Actor{ID: u.ID, Name: u.Name, Email: u.Email}
	}))
	s.RegisterAdmin(router)
	settings.RegisterAdmin(router)
	return func(method, path string, input any, status int) []byte {
		t.Helper()
		raw, _ := json.Marshal(input)
		r := httptest.NewRequest(method, path, bytes.NewReader(raw))
		r.Header.Set("Content-Type", "application/json")
		if status != 401 {
			r.AddCookie(&http.Cookie{Name: "monkeyai_session", Value: token})
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("%s %s: 状态 %d，期望 %d，响应 %s", method, path, w.Code, status, w.Body.String())
		}
		if strings.Contains(w.Body.String(), "BEGIN ") {
			t.Fatal("响应泄漏证书内容")
		}
		return w.Body.Bytes()
	}
}

func TestBillingSettingsRetireExternalMode(t *testing.T) {
	s, user, _ := fixture(t)
	call := billingAdmin(t, s, user)
	path := "/billing/settings"
	response := call(http.MethodGet, path, nil, http.StatusOK)
	if strings.Contains(string(response), `"wallet"`) {
		t.Fatal("external wallet fields are still exposed")
	}
	p, err := s.Policy(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	call(http.MethodPatch, path+"/mode", map[string]any{"revision": p.Revision, "charging_mode": "remote", "enabled": true}, http.StatusBadRequest)
	call(http.MethodPatch, path+"/wallet", map[string]any{"revision": p.Revision}, http.StatusNotFound)
	call(http.MethodPut, "/billing/accounts/"+user+"/wallet", nil, http.StatusNotFound)
	for index, enabled := range []bool{false, true} {
		call(http.MethodPatch, path+"/mode", map[string]any{"revision": p.Revision + int64(index), "charging_mode": "local", "enabled": enabled}, http.StatusOK)
		current, err := s.Policy(t.Context())
		if err != nil || current.Enabled != enabled || current.Mode != "local" {
			t.Fatalf("unexpected billing policy: %+v, %v", current, err)
		}
	}
}

func TestRetireExternalBillingMigrationPreservesHistory(t *testing.T) {
	s, user, model := fixture(t)
	ctx := t.Context()
	reservation, err := s.Begin(ctx, Request{UserID: user, ResourceID: model, Category: "model"})
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000051_billing_remove_external_wallet.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../migrations/000051_billing_remove_external_wallet.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	ledgerExec(t, s, string(down))
	ledgerExec(t, s, `UPDATE settings SET value = value || '{"charging_mode":"remote","enabled":true,"wallet":{"app_id":4}}' WHERE key='billing'`)
	ledgerExec(t, s, `UPDATE billing_transactions SET mode='remote' WHERE id=$1`, reservation.ID)
	if _, err := s.pool.Exec(ctx, string(up)); err == nil || !strings.Contains(err.Error(), "Unsettled external billing transactions") {
		t.Fatalf("migration should refuse an unsettled external transaction: %v", err)
	}
	var mode string
	if err := s.pool.QueryRow(ctx, `SELECT value->>'charging_mode' FROM settings WHERE key='billing'`).Scan(&mode); err != nil || mode != "remote" {
		t.Fatalf("blocked migration changed policy: %q, %v", mode, err)
	}

	ledgerExec(t, s, `UPDATE billing_transactions SET status='rejected' WHERE id=$1`, reservation.ID)
	ledgerExec(t, s, `INSERT INTO wallet_billing_records(biz_id, transaction_id, external_user_id, base_url, app_id, status, frozen_amount_quota)
		VALUES('historical-record', $1, 'synthetic-user', 'https://example.test', 4, 'rejected', 100)`, reservation.ID)
	ledgerExec(t, s, string(up))
	policy, err := s.Policy(ctx)
	if err != nil || policy.Mode != "local" || policy.Enabled {
		t.Fatalf("old external mode should become disabled: %+v, %v", policy, err)
	}
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM billing_transactions t JOIN wallet_billing_records w ON w.transaction_id=t.id
		WHERE t.id=$1 AND t.mode='remote' AND t.status='rejected' AND w.biz_id='historical-record'`, reservation.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("historical transaction changed: %d, %v", count, err)
	}
	call := billingAdmin(t, s, user)
	response := call(http.MethodGet, "/billing/transactions/"+reservation.ID, nil, http.StatusOK)
	var history struct {
		Mode    string `json:"mode"`
		Records []struct {
			ID string `json:"biz_id"`
		} `json:"wallet_records"`
	}
	if err := json.Unmarshal(response, &history); err != nil || history.Mode != "remote" || len(history.Records) != 1 || history.Records[0].ID != "historical-record" {
		t.Fatalf("historical transaction is not available through the admin API: %s, %v", response, err)
	}
}
