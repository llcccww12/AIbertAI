package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Some self-hosted Chat Completions servers keep the connection open long
// after [DONE]. The gateway must record delivery at [DONE] so the SDK's Ack
// succeeds, and the next call must not wait for the old connection to close.
func TestDSHGatewayDeliversAtDoneWhileUpstreamStaysOpen(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const modelID = "22222222-2222-4222-8222-222222222222"
	const first = "33333333-3333-4333-8333-333333333333"
	const second = "55555555-5555-4555-8555-555555555555"
	const keyID = "44444444-4444-4444-8444-444444444444"
	release := make(chan struct{})
	var drained atomic.Bool
	var calls atomic.Int32
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" {
			w := httptest.NewRecorder()
			switch {
			case r.URL.Path == "/api/v1/models":
				fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, modelID)
			case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
				fmt.Fprintf(w, `{"id":%q,"api_key":"key"}`, keyID)
			}
			return w.Result(), nil
		}
		hold := calls.Add(1) == 1
		body, w := io.Pipe()
		go func() {
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
			if hold {
				<-release
				drained.Store(true)
			}
			w.Close()
		}()
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: body, Request: r}, nil
	})
	gateway := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	g, err := newDSHLocalGateway(gateway, session, selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions"}, filepath.Join(t.TempDir(), "journal"))
	if err != nil {
		t.Fatal(err)
	}
	g.requireInvocationID = true
	server := httptest.NewServer(g)
	defer server.Close()
	defer close(release)

	call := func(invocation string) {
		t.Helper()
		r, _ := http.NewRequest("POST", server.URL+"/v1/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":%q,"stream":true}`, modelID)))
		r.Header.Set("Authorization", "Bearer "+g.token)
		r.Header.Set("X-AIBERTAI-Invocation-ID", invocation)
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: status %d", invocation, resp.StatusCode)
		}
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if scanner.Text() == "data: [DONE]" {
				return
			}
		}
		t.Fatalf("%s: stream ended without [DONE]", invocation)
	}

	call(first)
	started := time.Now()
	if err := g.Ack(first); err != nil {
		t.Fatalf("ack while upstream still open: %v", err)
	}
	if time.Since(started) > 2*time.Second || drained.Load() {
		t.Fatalf("ack waited for the upstream to close")
	}
	call(second)
	if err := g.Ack(second); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if drained.Load() {
		t.Fatal("second call waited for the first upstream to close")
	}
}
