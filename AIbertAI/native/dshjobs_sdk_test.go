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

// The model is deterministic; execution, retained output and cancellation are
// provided by the packaged SDK's bash plugin and job registry.
func TestDSHJobsLiveSDK(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const session = "11111111-1111-4111-8111-111111111111"
	const model = "22222222-2222-4222-8222-222222222222"
	const key = "33333333-3333-4333-8333-333333333333"
	work := t.TempDir()
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	client, _ := newBackendClient("https://gateway.test", false)
	var calls atomic.Int32
	continueModel := make(chan struct{})
	resumed := false
	defer func() {
		if !resumed {
			close(continueModel)
		}
	}()
	client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, model)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"fixture"}`, key)
		case r.URL.Path == "/api/v1/api-keys/"+key && r.Method == "DELETE":
		case r.URL.Path == "/v1/chat/completions":
			n := calls.Add(1)
			delta := map[string]any{"content": "finished"}
			finish := "stop"
			if n == 1 || n == 3 {
				status := "in_progress"
				if n == 3 {
					status = "completed"
				}
				args, _ := json.Marshal(map[string]any{"todos": []map[string]string{{"content": "Run the fixture job", "status": status}, {"content": "Review output", "status": "pending"}}})
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("call-todos-%d", n), "type": "function", "function": map[string]any{"name": "todo_write", "arguments": string(args)}}}}
				finish = "tool_calls"
			} else if n == 2 {
				args, _ := json.Marshal(map[string]any{"description": "Watch a fixture job", "command": "printf 'ready-from-bash\\n'; exec sleep 30", "run_in_background": true})
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "call-watch", "type": "function", "function": map[string]any{"name": "bash", "arguments": string(args)}}}}
				finish = "tool_calls"
			} else {
				select {
				case <-continueModel:
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			w.Header().Set("Content-Type", "text/event-stream")
			frame, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}})
			fmt.Fprintf(w, "data: %s\n\ndata: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":%q}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":1,\"total_tokens\":4}}\n\ndata: [DONE]\n\n", frame, finish)
		default:
			fmt.Fprint(w, `{}`)
		}
		return w.Result(), nil
	})
	gateway := &dshGateway{client: client, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(work, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(work, "journal"), gateway, func(context.Context, acpSession) (selectedModelConfig, error) {
		return selectedModelConfig{ID: model, Protocol: "openai_chat_completions", MaxOutputTokens: 4096, ContextWindowTokens: 100000}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg.LeasePool = &dshLeasePool{}
	defer func() {
		c, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		_ = cfg.LeasePool.close(c)
	}()
	row := acpSession{ID: session, DSHSessionID: session, Engine: "dsh", CWD: work, Model: model}
	host := &relaySessionHost{live: &acpLive{dsh: cfg, store: &desktopSessionStore{}, sessions: map[string]acpSession{session: row}}}
	ctx = context.WithValue(ctx, dshNativeApprovalKey{}, dshNativeApproval(func(_ json.RawMessage, _ *acpTurn) (string, error) { return "allowed-once", nil }))
	turn := acpTurn{Prompt: "Run the fixture job", Operation: "44444444-4444-4444-8444-444444444444"}
	result := make(chan error, 1)
	var plans [][]dshTodoItem
	go func() {
		result <- runDSHTurn(ctx, cfg, row, &turn, false, func(update map[string]any) error {
			if update["sessionUpdate"] == "plan" {
				plans = append(plans, update["entries"].([]dshTodoItem))
			}
			return nil
		})
	}()
	defer cancel()
	call := func(action, id string, from *int64, dest any) {
		t.Helper()
		raw, _ := json.Marshal(dshJobsRequest{SessionID: session, Action: action, JobID: id, From: from})
		v, e := host.sessionJobs(ctx, raw)
		if e != nil {
			t.Fatal(e)
		}
		b, e := json.Marshal(v)
		if e != nil || json.Unmarshal(b, dest) != nil {
			t.Fatalf("invalid %s response: %s", action, b)
		}
	}
	var id string
	var rows struct {
		Jobs []struct{ ID, Status, Owner, Kind string }
	}
	for id == "" {
		call("list", "", nil, &rows)
		for _, j := range rows.Jobs {
			if j.Owner == session && j.Kind == "bash" && j.Status == "running" {
				id = j.ID
			}
		}
		if id != "" {
			break
		}
		select {
		case e := <-result:
			t.Fatalf("turn finished before job was visible: %v, tools: %+v", e, turn.ToolRuns)
		case <-ctx.Done():
			t.Fatalf("no running bash job, model calls %d", calls.Load())
		case <-time.After(80 * time.Millisecond):
		}
	}
	var output struct {
		Chunks []struct{ Text string }
		Next   int64
	}
	for len(output.Chunks) == 0 {
		call("output", id, nil, &output)
		if ctx.Err() != nil {
			t.Fatal(ctx.Err())
		}
		if len(output.Chunks) == 0 {
			time.Sleep(60 * time.Millisecond)
		}
	}
	if !strings.Contains(output.Chunks[0].Text, "ready-from-bash") {
		t.Fatalf("unexpected real bash output: %+v", output)
	}
	var again struct{ Chunks []struct{ Text string } }
	call("output", id, nil, &again)
	if len(again.Chunks) == 0 || again.Chunks[0].Text != output.Chunks[0].Text {
		t.Fatal("reopening lost retained output")
	}
	call("output", id, &output.Next, &again)
	if len(again.Chunks) != 0 {
		t.Fatal("resumed observation duplicated output")
	}
	var killed struct{ Result string }
	call("kill", id, nil, &killed)
	if killed.Result != "requested" {
		t.Fatalf("stop result %+v", killed)
	}
	close(continueModel)
	resumed = true
	if e := <-result; e != nil {
		t.Fatal(e)
	}
	if len(plans) != 3 || len(plans[0]) != 0 || len(plans[1]) != 2 || plans[1][0].Status != "in_progress" || len(plans[2]) != 2 || plans[2][0].Status != "completed" || plans[2][1].Status != "pending" {
		t.Fatalf("native todo updates: %+v", plans)
	}
	call("list", "", nil, &rows)
	if len(rows.Jobs) != 1 || rows.Jobs[0].ID != id || rows.Jobs[0].Status != "killed" {
		t.Fatalf("closed turn lost final job state: %+v", rows)
	}
	// A pre-upgrade desktop saved tool calls only. Reopening recovers the
	// plugin's last actual write even after the entire SDK has shut down.
	if err := cfg.LeasePool.evictIfIdle(row.ID); err != nil {
		t.Fatal(err)
	}
	turn.DSHTodos = nil
	row.Turns = []acpTurn{turn}
	old := &relaySessionHost{live: &acpLive{dsh: cfg, cwd: row.CWD, dir: t.TempDir(), sessions: map[string]acpSession{}}}
	if err := old.live.save(row); err != nil {
		t.Fatal(err)
	}
	old.restoreLegacyDSHTodos(ctx, row.ID)
	restored, e := old.live.load(row.ID)
	if e != nil || restored.Turns[0].DSHTodos == nil || restored.Turns[0].DSHTodos.Items[0].Status != "completed" {
		t.Fatalf("old SDK history did not restore todos: %v", e)
	}
	meta := restored.workspaceMetadata()
	if meta["dshTodoSeq"] != restored.Turns[0].DSHTodos.Seq || len(meta["plan"].([]dshTodoItem)) != 2 {
		t.Fatalf("historic session metadata missing plan")
	}
}
