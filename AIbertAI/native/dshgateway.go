package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

// dshGateway is a host-side transport. The runtime must persist SessionID and
// CallID before dispatch and reuse both, and the same body, on a retry. It must
// consume and close the returned response body, including on cancellation.
type dshGateway struct {
	client        *client
	authorization func(context.Context) (authorization, error)
}

type dshGatewayCall struct {
	SessionID string // platform session UUID registered through registerSession
	CallID    string // persisted UUID unique to this model invocation
	ModelID   string // ID from the authorized platform model catalog
	Body      []byte // OpenAI request, including the same model ID
}

// Match the platform model proxy's request limit so large context and image
// inputs are not rejected by the local adapter before reaching the proxy.
const dshGatewayMaxRequestBytes = 32 << 20

func (g *dshGateway) invoke(ctx context.Context, call dshGatewayCall) (*http.Response, error) {
	if !validUUID(call.SessionID) || !validUUID(call.CallID) || !validUUID(call.ModelID) || len(call.Body) > dshGatewayMaxRequestBytes {
		return nil, errors.New("invalid_dsh_gateway_call")
	}
	var request struct {
		Model string `json:"model"`
	}
	if json.Unmarshal(call.Body, &request) != nil || request.Model != call.ModelID {
		return nil, errors.New("invalid_dsh_gateway_model")
	}
	auth, err := g.authorization(ctx)
	if err != nil {
		return nil, &dshPreflightError{err}
	}
	model, err := g.client.modelConfig(ctx, auth, call.ModelID)
	if err != nil {
		return nil, &dshPreflightError{err}
	}
	path := "/v1/chat/completions"
	if model.Protocol == "openai_responses" {
		path = "/v1/responses"
	}
	key, err := g.client.createModelKey(ctx, auth)
	if err != nil {
		return nil, &dshPreflightError{err}
	}
	retire := func() error {
		cleanup, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		current, err := g.authorization(cleanup)
		if err != nil {
			return err
		}
		return g.client.json(cleanup, http.MethodDelete, "/api/v1/api-keys/"+key.ID, current.Access, "", nil, nil)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.client.server+path, bytes.NewReader(call.Body))
	if err != nil {
		return nil, &dshPreflightError{errors.Join(err, retire())}
	}
	req.Header.Set("Authorization", "Bearer "+key.Value)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MAI-Session-ID", call.SessionID)
	req.Header.Set("X-Aibertai-DSH-Stream", "1")
	req.Header.Set("Idempotency-Key", call.CallID)
	// This journal owns retries. Do not let net/http resend a body invisibly on
	// a reused connection after dispatch may already have occurred.
	req.GetBody = nil
	// The shared client has a short total deadline for ordinary API requests.
	// A DSH model stream may remain active much longer; its request context and
	// the model proxy's idle limit and request context still govern cancellation.
	streamClient := &http.Client{Transport: g.client.http.Transport, CheckRedirect: g.client.http.CheckRedirect, Jar: g.client.http.Jar}
	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, errors.Join(err, retire())
	}
	// Preserve response status, transaction headers and streaming usage exactly;
	// 409 is not replayed as success because the gateway does not replay output.
	resp.Body = &dshGatewayBody{ReadCloser: resp.Body, retire: retire}
	return resp, nil
}

type dshGatewayBody struct {
	io.ReadCloser
	retire func() error
	closed bool
}

func (b *dshGatewayBody) Close() error {
	if b.closed {
		return nil
	}
	b.closed = true
	return errors.Join(b.ReadCloser.Close(), b.retire())
}
