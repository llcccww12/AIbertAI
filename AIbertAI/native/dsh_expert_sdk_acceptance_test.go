package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDSHExpertInstructionsPatchedSDK(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const id = "11111111-1111-4111-8111-111111111111"
	const model = "33333333-3333-4333-8333-333333333333"
	const firstID = "99999999-9999-4999-8999-999999999999"
	const secondID = "88888888-8888-4888-8888-888888888888"
	const key = "44444444-4444-4444-8444-444444444444"
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	// An employee may previously have enabled the native scheduler in the shared profile.
	// The host overlay must still take precedence, including when SDK defaults are selected.
	if err := os.WriteFile(filepath.Join(home, "cordis.patch.yml"), []byte("- id: schedule\n  disabled: false\n- id: tool-schedule\n  disabled: false\n- id: hmr\n  disabled: false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	requests := make(chan string, 4)
	var requestedDoc atomic.Bool
	const docPrompt = "Load the bundled Standard documentation."
	client, _ := newBackendClient("https://gateway.test", false)
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/experts/"+firstID+"/manifest":
			fmt.Fprintf(w, `{"available":true,"expert_id":%q,"prompt":"First expert {{literal}}","dsh_plugins":["tool-fs-search","tool-skill","tool-schedule"]}`, firstID)
		case r.URL.Path == "/api/v1/experts/"+secondID+"/manifest":
			fmt.Fprintf(w, `{"available":true,"expert_id":%q,"prompt":"Second expert {{literal}}","dsh_plugins":[]}`, secondID)
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, model)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"fixture"}`, key)
		case r.URL.Path == "/api/v1/api-keys/"+key && r.Method == "DELETE":
		case r.URL.Path == "/v1/chat/completions":
			var request any
			_ = json.NewDecoder(r.Body).Decode(&request)
			body, _ := json.Marshal(request)
			requests <- string(body)
			w.Header().Set("Content-Type", "text/event-stream")
			if strings.Contains(string(body), docPrompt) && requestedDoc.CompareAndSwap(false, true) {
				toolCalls := []map[string]any{
					{"index": 0, "id": "skill-call", "type": "function", "function": map[string]string{"name": "skill", "arguments": `{"name":"dsh-doc"}`}},
					{"index": 1, "id": "models-call", "type": "function", "function": map[string]string{"name": "list_subagent_models", "arguments": `{"provider":"aibertai"}`}},
				}
				wireCalls, _ := json.Marshal(toolCalls)
				fmt.Fprintf(w, `data: {"id":"fixture","object":"chat.completion.chunk","model":%q,"choices":[{"index":0,"delta":{"role":"assistant","tool_calls":%s},"finish_reason":"tool_calls"}]}`+"\n\ndata: [DONE]\n\n", model, wireCalls)
				break
			}
			fmt.Fprintf(w, "data: {\"id\":\"fixture\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":%q,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"done\"},\"finish_reason\":null}]}\n\n", model)
			fmt.Fprintf(w, "data: {\"id\":\"fixture\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":%q,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":1,\"total_tokens\":4}}\n\ndata: [DONE]\n\n", model)
		default:
			fmt.Fprint(w, `{}`)
		}
		return w.Result(), nil
	})
	selected := selectedModelConfig{ID: model, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	gateway := &dshGateway{client: client, authorization: func(context.Context) (authorization, error) { return authorization{Access: "test"}, nil }}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, StandardSkillsDir: filepath.Join(filepath.Dir(entry), "..", "..", "..", "packages", "preset", "agent-preset", "skills"), Home: home, Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(root, "journal"), gateway, func(context.Context, acpSession) (selectedModelConfig, error) { return selected, nil })
	if err != nil {
		t.Fatal(err)
	}
	session := acpSession{ID: id, DSHSessionID: id, Engine: "dsh", CWD: root, Model: model}
	cfg.LeasePool = &dshLeasePool{}
	defer cfg.LeasePool.close(context.Background())
	host := &relaySessionHost{live: &acpLive{client: client, auth: authorization{Access: "test"}}}
	for i, expert := range []string{"First expert {{literal}}", "Second expert {{literal}}", ""} {
		ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
		session.ExpertID = []string{firstID, secondID, ""}[i]
		if expert != "" {
			var release func()
			ctx, release, err = host.bindDSHExpert(ctx, session)
			if err != nil {
				cancel()
				t.Fatal(err)
			}
			defer release()
		}
		turn := acpTurn{Operation: fmt.Sprintf("22222222-2222-4222-8222-%012d", i+1), Prompt: "work"}
		if expert == "" {
			turn.Prompt = docPrompt
		}
		err = runDSHTurn(ctx, cfg, session, &turn, i > 0, func(map[string]any) error { return nil })
		cancel()
		if err != nil {
			t.Fatalf("turn %d: %v", i, err)
		}
		request := <-requests
		var wire struct {
			Tools []struct{ Function struct{ Name string } }
		}
		if err := json.Unmarshal([]byte(request), &wire); err != nil {
			t.Fatal(err)
		}
		advertised := map[string]bool{}
		for _, tool := range wire.Tools {
			advertised[tool.Function.Name] = true
		}
		for _, name := range []string{"schedule_create", "schedule_list", "schedule_view", "schedule_edit"} {
			if advertised[name] {
				t.Fatalf("turn %d: session child advertised host-wide scheduler tool %s", i, name)
			}
		}
		if advertised["glob"] != (i != 1) || advertised["grep"] != (i != 1) || advertised["read"] != (i == 2) || advertised["skill"] != (i != 1) {
			t.Fatalf("turn %d, plugin selection failed: %v", i, advertised)
		}
		if i == 2 {
			select {
			case afterLoad := <-requests:
				if !strings.Contains(afterLoad, "aibertai/"+model+" — ") {
					t.Fatal("The native model discovery tool did not return the authorized parent route")
				}
				if !strings.Contains(afterLoad, "# DeepSeek Harness documentation") {
					t.Fatal("The native skill tool did not load the bundled documentation")
				}
			default:
				t.Fatal("The model did not receive the skill tool result")
			}
			for _, name := range []string{"edit", "todo_write", "subagent", "list_subagent_models"} {
				if !advertised[name] {
					t.Errorf("Standard mode did not advertise %q", name)
				}
			}
			if !strings.Contains(request, "You are a coding agent powered by") || !strings.Contains(request, "dsh-doc") {
				t.Error("Standard mode is missing its native role or bundled DSH documentation skill")
			}
		} else if advertised["list_subagent_models"] || strings.Contains(request, "dsh-doc") {
			t.Fatal("Standard mode configuration escaped into a named expert")
		}
		if expert != "" && !strings.Contains(request, expert) {
			t.Fatalf("turn %d missing expert: %s", i, request)
		}
		if i > 0 {
			old := "First expert"
			if i == 2 {
				old = "Second expert"
			}
			if strings.Contains(request, old) {
				t.Fatalf("turn %d leaked prior expert: %s", i, request)
			}
		}
	}
}
