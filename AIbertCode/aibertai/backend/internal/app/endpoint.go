package app

import (
	"context"
	"net/http"

	"aibertai.local/backend/internal/endpoint"
	"aibertai.local/backend/internal/identity"
)

type endpointAuth struct{ service *identity.Service }

func (a endpointAuth) Credential(r *http.Request) (endpoint.Credential, bool) {
	c, ok := identity.CredentialFromContext(r.Context())
	return endpoint.Credential{UserID: c.UserID, Reference: c.Reference, ExpiresAt: c.ExpiresAt}, ok
}

func (a endpointAuth) Valid(ctx context.Context, c endpoint.Credential) (bool, error) {
	return a.service.ValidCredential(ctx, identity.AccessCredential{UserID: c.UserID, Reference: c.Reference, ExpiresAt: c.ExpiresAt})
}
