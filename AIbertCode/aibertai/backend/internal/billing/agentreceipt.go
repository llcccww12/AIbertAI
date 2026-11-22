package billing

import (
	"errors"
	"net/http"
	"regexp"

	"aibertai.local/backend/internal/billing/sqlc"
	"aibertai.local/backend/internal/identity"
	"aibertai.local/backend/internal/resource"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var modelReceiptHash = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// agentModelReceipt only confirms that the exact user's model invocation has
// a durable terminal accounting record. It exposes no input or account data.
func (s *Service) agentModelReceipt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	user, ok := identity.UserFromContext(r.Context())
	principal, hasPrincipal := identity.PrincipalFromContext(r.Context())
	if !ok || !hasPrincipal || principal.UserID != user.ID || principal.Purpose == "runtime" {
		resource.Fail(r.Context(), w, resource.NotFound)
		return
	}
	session, invocation := chi.URLParam(r, "id"), chi.URLParam(r, "invocation")
	var in struct {
		ModelID       string `json:"model_id"`
		RequestHash   string `json:"request_hash"`
		TransactionID string `json:"transaction_id"`
	}
	if _, err := uuid.Parse(session); err != nil {
		resource.Fail(r.Context(), w, resource.NotFound)
		return
	}
	if _, err := uuid.Parse(invocation); err != nil {
		resource.Fail(r.Context(), w, resource.NotFound)
		return
	}
	if err := resource.Decode(w, r, &in); err != nil {
		resource.Fail(r.Context(), w, err)
		return
	}
	if _, err := uuid.Parse(in.ModelID); err != nil || !modelReceiptHash.MatchString(in.RequestHash) {
		resource.Fail(r.Context(), w, resource.NotFound)
		return
	}
	q := sqlc.New(s.pool)
	owned, err := q.SessionOwned(r.Context(), sqlc.SessionOwnedParams{ID: session, OwnerUserID: user.ID})
	if err != nil {
		resource.Fail(r.Context(), w, err)
		return
	}
	if !owned {
		resource.Fail(r.Context(), w, resource.NotFound)
		return
	}
	row, err := q.IdempotentTransaction(r.Context(), sqlc.IdempotentTransactionParams{UserID: user.ID, Category: "model", IdempotencyKey: &invocation})
	if errors.Is(err, pgx.ErrNoRows) || err == nil && (row.RequestHash != in.RequestHash || in.TransactionID != "" && row.ID != in.TransactionID) {
		resource.Fail(r.Context(), w, resource.NotFound)
		return
	}
	if err != nil {
		resource.Fail(r.Context(), w, err)
		return
	}
	record, err := resource.DecodeObject(q.GetTransaction(r.Context(), row.ID))
	if err != nil {
		resource.Fail(r.Context(), w, err)
		return
	}
	if record.String("user_id") != user.ID || record.String("session_id") != session || record.String("resource_id") != in.ModelID || record.String("category") != "model" || record.String("idempotency_key") != invocation || record.String("request_hash") != in.RequestHash {
		resource.Fail(r.Context(), w, resource.NotFound)
		return
	}
	status, result := record.String("status"), record.String("result")
	terminal := (status == "unknown" || status == "settling" || status == "settled" || status == "released" || status == "rejected") && record.String("completed_at") != "" && (result == "cancelled" || result == "failed" || result == "succeeded")
	out := map[string]any{"recorded": terminal, "transaction_id": row.ID}
	if terminal {
		out["status"] = status
		out["result"] = result
		out["error_code"] = record.String("error_code")
	}
	resource.JSON(r.Context(), w, http.StatusOK, out)
}
