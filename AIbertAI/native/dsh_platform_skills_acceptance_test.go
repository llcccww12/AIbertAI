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

// A synced platform skill is discoverable and loadable in a Standard session
// and stays out of a named expert's session.
func TestDSHPlatformSkillsPatchedSDK(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const id = "11111111-1111-4111-8111-111111111111"
	const model = "33333333-3333-4333-8333-333333333333"
	const expertID = "99999999-9999-4999-8999-999999999999"
	root := t.TempDir()
	home := filepath.Join(root, "home")
	const enabledSkill = "66666666-6666-4666-8666-666666666666"
	const disabledSkill = "77777777-7777-4777-8777-777777777777"
	skills := filepath.Join(root, "platform-skills")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	for _, skill := range []struct{ id, name, description, body string }{
		{enabledSkill, "weekly-report", "Write the weekly report", "PLATFORM-SKILL-BODY: use three bullet points."},
		{disabledSkill, "travel-plan", "Plan a business trip", "Book flights."},
	} {
		dir := filepath.Join(skills, skill.name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), dshSkillDocument(skill.name, skill.description, skill.body), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, dshPlatformSkillStamp), []byte(skill.id+"\nsha\n"+skill.name+"\n"+skill.description), 0600); err != nil {
			t.Fatal(err)
		}
	}
	requests := make(chan string, 8)
	var loaded atomic.Bool
	client, _ := newBackendClient("https://gateway.test", false)
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/experts/"+expertID+"/manifest":
			fmt.Fprintf(w, `{"available":true,"expert_id":%q,"prompt":"Expert role","dsh_plugins":["tool-skill","skill-filesystem"]}`, expertID)
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, model)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprint(w, `{"id":"44444444-4444-4444-8444-444444444444","api_key":"fixture"}`)
		case r.URL.Path == "/v1/chat/completions":
			var request any
			_ = json.NewDecoder(r.Body).Decode(&request)
			body, _ := json.Marshal(request)
			requests <- string(body)
			w.Header().Set("Content-Type", "text/event-stream")
			if strings.Contains(string(body), "weekly-report") && !strings.Contains(string(body), "Expert role") && loaded.CompareAndSwap(false, true) {
				calls, _ := json.Marshal([]map[string]any{{"index": 0, "id": "skill-call", "type": "function", "function": map[string]string{"name": "skill", "arguments": `{"name":"weekly-report"}`}}})
				fmt.Fprintf(w, `data: {"id":"fixture","object":"chat.completion.chunk","model":%q,"choices":[{"index":0,"delta":{"role":"assistant","tool_calls":%s},"finish_reason":"tool_calls"}]}`+"\n\ndata: [DONE]\n\n", model, calls)
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
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, PlatformSkillsDir: skills, Home: home, Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(root, "journal"), gateway, func(context.Context, acpSession) (selectedModelConfig, error) { return selected, nil })
	if err != nil {
		t.Fatal(err)
	}
	cfg.LeasePool = &dshLeasePool{}
	defer cfg.LeasePool.close(context.Background())
	host := &relaySessionHost{live: &acpLive{client: client, auth: authorization{Access: "test"}}}

	// Standard session: only the enabled skill is listed, then loaded by the skill tool.
	session := acpSession{ID: id, DSHSessionID: id, Engine: "dsh", CWD: root, Model: model}
	if err := materializeDSHSessionSkills(skills, dshSessionSkillsDir(skills, id), func(skill string) bool { return skill == enabledSkill }); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	turn := acpTurn{Operation: "22222222-2222-4222-8222-000000000001", Prompt: "Write my weekly report"}
	err = runDSHTurn(ctx, cfg, session, &turn, false, func(map[string]any) error { return nil })
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	if first := <-requests; !strings.Contains(first, "weekly-report") || !strings.Contains(first, "Write the weekly report") {
		t.Fatal("platform skill not listed to the model")
	} else if strings.Contains(first, "travel-plan") {
		t.Fatal("a skill switched off for this conversation was listed")
	}
	select {
	case after := <-requests:
		if !strings.Contains(after, "PLATFORM-SKILL-BODY") {
			t.Fatal("skill tool did not load the platform skill body")
		}
	default:
		t.Fatal("model did not receive the skill tool result")
	}

	// Named expert session: platform skills stay out.
	expertSessionID := "55555555-5555-4555-8555-555555555555"
	expertSession := acpSession{ID: expertSessionID, DSHSessionID: expertSessionID, Engine: "dsh", CWD: root, Model: model, ExpertID: expertID}
	ctx, cancel = context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	ctx, release, err := host.bindDSHExpert(ctx, expertSession)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	turn = acpTurn{Operation: "22222222-2222-4222-8222-000000000002", Prompt: "Write my weekly report"}
	if err = runDSHTurn(ctx, cfg, expertSession, &turn, false, func(map[string]any) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if request := <-requests; strings.Contains(request, "weekly-report") || !strings.Contains(request, "Expert role") {
		t.Fatal("platform skills leaked into a named expert session")
	}
}
