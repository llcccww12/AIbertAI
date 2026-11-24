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
	"testing"
	"time"
)

// The SDK may settle as soon as it reads [DONE]; the upstream connection can
// close slightly later. Ack must not race the gateway's delivered record.
func TestDSHGatewayAckAfterDoneWaitsForDelivery(t *testing.T) {
	const session = "11111111-1111-4111-8111-111111111111"
	const modelID = "22222222-2222-4222-8222-222222222222"
	const invocation = "33333333-3333-4333-8333-333333333333"
	const keyID = "44444444-4444-4444-8444-444444444444"
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
		body, w := io.Pipe()
		go func() {
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n")
			fmt.Fprint(w, "data: [DONE]\n\n")
			time.Sleep(300 * time.Millisecond)
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

	r, _ := http.NewRequest("POST", server.URL+"/v1/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":%q,"stream":true}`, modelID)))
	r.Header.Set("Authorization", "Bearer "+g.token)
	r.Header.Set("X-AIBERTAI-Invocation-ID", invocation)
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	scanner := bufio.NewScanner(resp.Body)
	sawDone := false
	for scanner.Scan() {
		if scanner.Text() == "data: [DONE]" {
			sawDone = true
			break
		}
	}
	if !sawDone {
		t.Fatal("stream ended without [DONE]")
	}
	if err := g.Ack(invocation); err != nil {
		t.Fatalf("ack after [DONE]: %v", err)
	}
}
