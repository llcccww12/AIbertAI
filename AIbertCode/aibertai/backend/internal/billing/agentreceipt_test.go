package billing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aibertai.local/backend/internal/identity"
	"aibertai.local/backend/internal/resource"
	"github.com/go-chi/chi/v5"
)

func TestAgentModelReceipt(t *testing.T) {
	s, user, model := fixture(t)
	ctx := t.Context()
	other, session, invocation := resource.ID(), resource.ID(), resource.ID()
	if _, err := s.pool.Exec(ctx, `INSERT INTO users(id,name,email,role) VALUES($1,'Other','other-receipt@example.com','user')`, other); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `INSERT INTO sessions(id,owner_user_id,title,session_type,client_type,client_name,started_at,last_active_at) VALUES($1,$2,'fixture','conversation','desktop','fixture',now(),now())`, session, user); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{user, other} {
		h := sha256.Sum256([]byte(id))
		if _, err := s.pool.Exec(ctx, `INSERT INTO oauth_tokens(user_id,client_id,access_token_hash,refresh_token_hash,access_expires_at,refresh_expires_at) VALUES($1,'test',$2,$3,now()+interval '1 hour',now()+interval '2 hours')`, id, hex.EncodeToString(h[:]), "refresh-"+id); err != nil {
			t.Fatal(err)
		}
		if _, err := s.pool.Exec(ctx, `UPDATE oauth_grants SET scopes=ARRAY['session:report'] WHERE user_id=$1`, id); err != nil {
			t.Fatal(err)
		}
	}
	hash := "sha256:" + strings.Repeat("a", 64)
	reservation, err := s.Begin(ctx, Request{UserID: user, ResourceID: model, SessionID: session, Category: "model", IdempotencyKey: invocation, RequestHash: hash})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Start(ctx, reservation.ID); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	router.Use(identity.NewService(s.pool, nil, "").RequireAgent)
	s.RegisterAgent(router)
	type receipt struct {
		Recorded      bool   `json:"recorded"`
		Status        string `json:"status"`
		TransactionID string `json:"transaction_id"`
		Result        string `json:"result"`
		ErrorCode     string `json:"error_code"`
	}
	var last receipt
	call := func(token, sessionID, callID, modelID, requestHash, transaction string, status int) bool {
		t.Helper()
		body := fmt.Sprintf(`{"model_id":%q,"request_hash":%q,"transaction_id":%q}`, modelID, requestHash, transaction)
		r := httptest.NewRequest(http.MethodPost, "/sessions/"+sessionID+"/model-invocations/"+callID+"/receipt", strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != status {
			t.Fatalf("status=%d want=%d body=%s", w.Code, status, w.Body.String())
		}
		last = receipt{}
		if status == http.StatusOK && json.Unmarshal(w.Body.Bytes(), &last) != nil {
			t.Fatal("invalid receipt")
		}
		if status == http.StatusOK && (last.TransactionID != reservation.ID || last.Recorded && last.Status == "" || !last.Recorded && (last.Result != "" || last.ErrorCode != "")) {
			t.Fatalf("invalid receipt fields: %+v", last)
		}
		return last.Recorded
	}
	if call(user, session, invocation, model, hash, reservation.ID, http.StatusOK) {
		t.Fatal("running request treated as terminal")
	}
	if _, err := s.pool.Exec(ctx, `UPDATE billing_transactions SET status='unknown', error_code='execution_interrupted' WHERE id=$1`, reservation.ID); err != nil {
		t.Fatal(err)
	}
	if call(user, session, invocation, model, hash, reservation.ID, http.StatusOK) {
		t.Fatal("recovery scan without a terminal report treated as terminal")
	}
	if err := s.Finish(ctx, reservation.ID, Usage{Known: false, Result: "failed", ErrorCode: "upstream_connection_failed"}); err != nil {
		t.Fatal(err)
	}
	if !call(user, session, invocation, model, hash, reservation.ID, http.StatusOK) {
		t.Fatal("durable unknown usage not reported")
	}
	if last.Result != "failed" || last.ErrorCode != "upstream_connection_failed" {
		t.Fatalf("terminal evidence was lost: %+v", last)
	}
	call(other, session, invocation, model, hash, reservation.ID, http.StatusNotFound)
	call(user, resource.ID(), invocation, model, hash, reservation.ID, http.StatusNotFound)
	call(user, session, invocation, resource.ID(), hash, reservation.ID, http.StatusNotFound)
	call(user, session, invocation, model, "sha256:"+strings.Repeat("b", 64), reservation.ID, http.StatusNotFound)
	call(user, session, invocation, model, hash, resource.ID(), http.StatusNotFound)
	call(user, session, resource.ID(), model, hash, "", http.StatusNotFound)
	if !call(user, session, invocation, model, hash, "", http.StatusOK) {
		t.Fatal("receipt cannot be recovered without an earlier response header")
	}
	call("invalid", session, invocation, model, hash, reservation.ID, http.StatusUnauthorized)
	if _, err := s.pool.Exec(ctx, `UPDATE oauth_grants SET scopes=ARRAY['session:read'] WHERE user_id=$1`, user); err != nil {
		t.Fatal(err)
	}
	call(user, session, invocation, model, hash, reservation.ID, http.StatusForbidden)
	if _, err := s.pool.Exec(ctx, `UPDATE oauth_grants SET purpose='runtime', scopes=ARRAY['session:report'] WHERE user_id=$1`, user); err != nil {
		t.Fatal(err)
	}
	call(user, session, invocation, model, hash, reservation.ID, http.StatusNotFound)
}

func TestModelReceiptHashContract(t *testing.T) {
	body := json.RawMessage(`{"model":"fixture","messages":[{"role":"user","content":"中文 <&>"}]}`)
	value := resource.Hash(map[string]any{"model_id": "22222222-2222-4222-8222-222222222222", "path": "/v1/chat/completions", "body": body, "session_id": "11111111-1111-4111-8111-111111111111"})
	if value != "sha256:5b8a76de595ac296a0cd16374bef6fafa13503f0a2a13adb3750d8d6eff5b3ea" {
		t.Fatalf("request hash changed: %s", value)
	}
	value = resource.Hash(map[string]any{"model_id": "22222222-2222-4222-8222-222222222222", "path": "/v1/chat/completions", "body": json.RawMessage(`{"model":"real-model","stream":true}`), "session_id": "11111111-1111-4111-8111-111111111111"})
	if value != "sha256:7d9aa8100a0b94e9e7d64312b7b093d9f8266b2509e8f07da5c73762055a7233" {
		t.Fatalf("rewritten request hash changed: %s", value)
	}
}
