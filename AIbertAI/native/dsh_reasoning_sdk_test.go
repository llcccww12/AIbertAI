package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDSHReasoningRealSDK(t *testing.T) {
	for _, protocol := range []string{"openai_chat_completions", "openai_responses"} {
		t.Run(protocol, func(t *testing.T) { testDSHReasoningRealSDK(t, protocol) })
	}
}

func testDSHReasoningRealSDK(t *testing.T, protocol string) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const session = "11111111-1111-4111-8111-111111111111"
	const model = "22222222-2222-4222-8222-222222222222"
	const key = "33333333-3333-4333-8333-333333333333"
	const operation = "44444444-4444-4444-8444-444444444444"
	root := t.TempDir()
	c, _ := newBackendClient("https://gateway.test", false)
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":%q}]}`, model, protocol)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == http.MethodPost:
			fmt.Fprintf(w, `{"id":%q,"api_key":"fixture"}`, key)
		case r.URL.Path == "/api/v1/api-keys/"+key && r.Method == http.MethodDelete:
		case r.URL.Path == "/v1/chat/completions" || r.URL.Path == "/v1/responses":
			w.Header().Set("Content-Type", "text/event-stream")
			if protocol == "openai_responses" {
				reasoning := `{"id":"rs1","type":"reasoning","summary":[{"type":"summary_text","text":"考虑测试条件。"}]}`
				message := `{"id":"msg1","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"完成。","annotations":[]}]}`
				fmt.Fprint(w, "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp1\"}}\n\n")
				fmt.Fprint(w, "data: {\"type\":\"response.output_item.added\",\"output_index\":0,\"item\":{\"id\":\"rs1\",\"type\":\"reasoning\",\"summary\":[]}}\n\n")
				fmt.Fprint(w, "data: {\"type\":\"response.reasoning_summary_text.delta\",\"output_index\":0,\"summary_index\":0,\"delta\":\"考虑测试条件。\"}\n\n")
				fmt.Fprintf(w, "data: {\"type\":\"response.output_item.done\",\"output_index\":0,\"item\":%s}\n\n", reasoning)
				fmt.Fprint(w, "data: {\"type\":\"response.output_item.added\",\"output_index\":1,\"item\":{\"id\":\"msg1\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[]}}\n\n")
				fmt.Fprint(w, "data: {\"type\":\"response.output_text.delta\",\"output_index\":1,\"content_index\":0,\"delta\":\"完成。\"}\n\n")
				fmt.Fprintf(w, "data: {\"type\":\"response.output_item.done\",\"output_index\":1,\"item\":%s}\n\n", message)
				fmt.Fprintf(w, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp1\",\"status\":\"completed\",\"output\":[%s,%s],\"usage\":{\"input_tokens\":3,\"output_tokens\":1,\"total_tokens\":4}}}\n\n", reasoning, message)
				return w.Result(), nil
			}
			fmt.Fprintf(w, "data: {\"id\":\"fixture\",\"object\":\"chat.completion.chunk\",\"model\":%q,\"choices\":[{\"index\":0,\"delta\":{\"reasoning_content\":\"考虑测试条件。\"},\"finish_reason\":null}]}\n\n", model)
			fmt.Fprintf(w, "data: {\"id\":\"fixture\",\"object\":\"chat.completion.chunk\",\"model\":%q,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"完成。\"},\"finish_reason\":null}]}\n\n", model)
			fmt.Fprintf(w, "data: {\"id\":\"fixture\",\"object\":\"chat.completion.chunk\",\"model\":%q,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":1,\"total_tokens\":4}}\n\ndata: [DONE]\n\n", model)
		default:
			http.NotFound(w, r)
		}
		return w.Result(), nil
	})
	gw := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	selected := selectedModelConfig{ID: model, Protocol: protocol, ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(root, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(root, "journal"), gw, func(context.Context, acpSession) (selectedModelConfig, error) { return selected, nil })
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DSH_EXPECT_PERSISTENT_JOBS") == "1" {
		cfg.LeasePool = &dshLeasePool{}
		t.Cleanup(func() { _ = cfg.LeasePool.close(context.Background()) })
	}
	turn := acpTurn{Operation: operation, Prompt: "请执行模拟任务"}
	var types []string
	err = runDSHTurn(t.Context(), cfg, acpSession{ID: session, DSHSessionID: session, CWD: root, Model: model}, &turn, false, func(update map[string]any) error {
		if update != nil {
			if kind, ok := update["sessionUpdate"].(string); ok {
				types = append(types, kind)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DSH_EXPECT_CONTEXT_METER") == "1" {
		if !turn.DSHContext.valid() || turn.DSHContext.Window != 100000 || *turn.DSHContext.Used <= 0 {
			t.Fatalf("SDK context meter missing: %+v", turn.DSHContext)
		}
		for _, key := range []string{"systemTokens", "toolsTokens", "messageTokens"} {
			if value, ok := turn.DSHContext.Breakdown[key]; !ok || value < 0 {
				t.Fatalf("SDK context composition missing %s: %+v", key, turn.DSHContext.Breakdown)
			}
		}
		turn.Model = model
		meta := (acpSession{Engine: "dsh", Model: model, Turns: []acpTurn{turn}}).workspaceMetadata()
		if meta["context"] != turn.DSHContext {
			t.Fatalf("SDK context meter missing on reload: %v", meta["context"])
		}
	}
	step, thought, settled := -1, -1, -1
	for i, kind := range types {
		if kind == "dsh_step" {
			step = i
		}
		if kind == "agent_thought_chunk" {
			thought = i
		}
		if kind == "agent_message_settled" {
			settled = i
		}
	}
	if step < 0 || thought <= step || settled <= thought || turn.DSHStep != 1 {
		t.Fatalf("reasoning was not streamed before settlement: %v", types)
	}
	turn.Started = time.Unix(100, 0)
	turn.Ended = turn.Started.Add(12*time.Minute + 19*time.Second)
	history := (acpSession{Engine: "dsh", Turns: []acpTurn{turn}}).responseMessage(0, map[string]any{})
	if history["dshExecution"] != true || history["startedAt"] != turn.Started.UnixMilli() || history["endedAt"] != turn.Ended.UnixMilli() || history["dshStep"] != nil {
		t.Fatalf("completed execution history lost its timing or still reports a live step: %v", history)
	}
	parts := dshRenderedParts(turn.DSHParts, turn.Answer)
	if len(parts) != 2 || parts[0]["type"] != "thought" || parts[0]["text"] != "考虑测试条件。" || parts[1]["text"] != "完成。" {
		t.Fatalf("reasoning lost from history: %#v", parts)
	}
}
