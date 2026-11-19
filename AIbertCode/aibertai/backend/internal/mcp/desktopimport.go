package mcp

import (
	"context"
	"encoding/json"
	"net/http"

	"aibertai.local/backend/internal/identity"
	"aibertai.local/backend/internal/mcp/sqlc"
	"aibertai.local/backend/internal/resource"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type desktopImportHeadersKey struct{}

func (s *Service) importDesktopConnector(w http.ResponseWriter, r *http.Request) {
	var input resource.Object
	if err := resource.Decode(w, r, &input); err != nil {
		resource.Fail(r.Context(), w, err)
		return
	}
	ctx := r.Context()
	if value, present := input["http_headers"]; present {
		data, err := json.Marshal(value)
		if err != nil {
			resource.Fail(ctx, w, resource.Invalid("认证 Header 无效"))
			return
		}
		headers, err := decodeHeaders(data)
		if err != nil {
			resource.Fail(ctx, w, err)
			return
		}
		input["authorization_mode"], input["authorization_method"] = "independent", "http_header"
		ctx = context.WithValue(ctx, desktopImportHeadersKey{}, headers)
	}
	user, _ := identity.UserFromContext(ctx)
	item, err := s.Connectors.ImportUser(ctx, user.ID, chi.URLParam(r, "key"), input)
	if err != nil {
		resource.Fail(ctx, w, err)
		return
	}
	resource.ETag(w, item)
	resource.JSON(ctx, w, http.StatusOK, item)
}

const desktopImportedCredentialName = "旧本机配置"

func persistDesktopImportHeaders(ctx context.Context, tx pgx.Tx, connector resource.Object, headers map[string]string) error {
	id := resource.ID()
	data, err := json.Marshal(resource.Object{
		"id": id, "connector_id": connector.String("id"), "user_id": connector.String("actor_id"),
		"name": desktopImportedCredentialName, "http_headers": headers, "config_revision": connector["config_revision"],
	})
	if err != nil {
		return err
	}
	if _, err := sqlc.New(tx).CreateCredential(ctx, data); err != nil {
		return err
	}
	return resource.Audit(ctx, tx, connector.String("actor_id"), "connector_credential", id, "save")
}
