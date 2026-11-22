package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestDSHSharedProfileTwoSessions(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("set DSH_TEST_NODE and DSH_PATCHED_TEST_ENTRY for the real SDK")
	}
	root := t.TempDir()
	home := filepath.Join(root, "home")
	const modelID = "33333333-3333-4333-8333-333333333333"
	const keyID = "44444444-4444-4444-8444-444444444444"
	type modelView struct {
		tools   []string
		history string
	}
	views := make(chan modelView, 8)
	busy, finish := make(chan struct{}), make(chan struct{})
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	client, _ := newBackendClient("https://gateway.test", false)
	client.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case req.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, modelID)
		case req.URL.Path == "/api/v1/api-keys" && req.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"fixture"}`, keyID)
		case req.URL.Path == "/api/v1/api-keys/"+keyID && req.Method == "DELETE":
		case req.URL.Path == "/v1/chat/completions":
			var in struct {
				Tools    []struct{ Function struct{ Name string } }
				Messages []json.RawMessage
			}
			if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
				return nil, err
			}
			history, _ := json.Marshal(in.Messages)
			view := modelView{history: string(history)}
			for _, tool := range in.Tools {
				view.tools = append(view.tools, tool.Function.Name)
			}
			if len(in.Messages) > 0 && strings.Contains(string(in.Messages[len(in.Messages)-1]), "wait_for_release") {
				close(busy)
				select {
				case <-finish:
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			views <- view
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {\"id\":\"r\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":%q,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"done\"},\"finish_reason\":null}]}\n\n", modelID)
			fmt.Fprintf(w, "data: {\"id\":\"r\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":%q,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":1,\"total_tokens\":4}}\n\ndata: [DONE]\n\n", modelID)
		default:
			fmt.Fprint(w, `{}`)
		}
		return w.Result(), nil
	})
	selected := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	gateway := &dshGateway{client: client, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: home, Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(root, "journal"), gateway, func(context.Context, acpSession) (selectedModelConfig, error) { return selected, nil })
	if err != nil {
		t.Fatal(err)
	}
	cfg.LeasePool = &dshLeasePool{}
	defer cfg.LeasePool.close(context.Background())
	turn := func(session, prompt string, resume bool) error {
		op, err := newUUID()
		if err != nil {
			return err
		}
		s := acpSession{ID: session, DSHSessionID: session, Engine: "dsh", CWD: root, Model: modelID}
		return runDSHTurn(ctx, cfg, s, &acpTurn{Operation: op, Prompt: prompt}, resume, nil)
	}
	check := func(expectTodo bool, prior string) {
		t.Helper()
		var v modelView
		select {
		case v = <-views:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		if slices.Contains(v.tools, "todo_write") != expectTodo || !strings.Contains(v.history, prior) {
			t.Fatalf("todo=%t (want %t), prior message %q present=%t", slices.Contains(v.tools, "todo_write"), expectTodo, prior, strings.Contains(v.history, prior))
		}
	}
	a, b := "11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"
	for _, session := range []string{a, b} {
		if err := turn(session, "seed_"+session, false); err != nil {
			t.Fatal(err)
		}
		check(true, "seed_"+session)
	}
	firstB := cfg.LeasePool.slots[b].process
	running := make(chan error, 1)
	go func() { running <- turn(b, "wait_for_release", true) }()
	// Always release the fake model before the deferred lease shutdown.
	release := func() {
		select {
		case <-finish:
		default:
			close(finish)
		}
	}
	defer release()
	select {
	case <-busy:
	case err := <-running:
		t.Fatalf("model did not block: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	// Reproduce the manager's committed atomic enablement edit, leaving session B active.
	patch := filepath.Join(home, "profiles", "sdk", "cordis.patch.yml")
	if err := os.WriteFile(patch+".new", []byte("- id: tool-todo\n  disabled: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(patch+".new", patch); err != nil {
		t.Fatal(err)
	}
	if err := turn(a, "refresh idle", true); err != nil {
		t.Fatal(err)
	}
	check(false, "seed_"+a)
	release()
	if err := <-running; err != nil {
		t.Fatal(err)
	}
	check(true, "seed_"+b)
	if cfg.LeasePool.slots[b].process != firstB {
		t.Fatal("active session was restarted")
	}
	if err := turn(b, "refresh after release", true); err != nil {
		t.Fatal(err)
	}
	check(false, "wait_for_release")
	if cfg.LeasePool.slots[b].process == firstB {
		t.Fatal("stale SDK child reused")
	}
}

func TestDSHSharedProfileWaitsForPluginWriter(t *testing.T) {
	home := t.TempDir()
	profile := filepath.Join(home, "profiles", "sdk")
	if err := os.MkdirAll(profile, 0700); err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(profile, "package.json.lock")
	if err := os.WriteFile(lock, []byte("active writer"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := readDSHSharedProfile(ctx, home); !errors.Is(err, context.Canceled) {
		t.Fatalf("writer lock ignored: %v", err)
	}
	if _, err := os.Stat(lock); err != nil {
		t.Fatalf("reader removed writer lock: %v", err)
	}
	if err := os.Remove(lock); err != nil {
		t.Fatal(err)
	}
	if _, err := readDSHSharedProfile(t.Context(), home); err != nil {
		t.Fatal(err)
	}
}
