package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Ordinary chat must cross the pressure threshold without /compact and reuse
// the resulting checkpoint after the SDK child is closed and the chat reopens.
func TestDSHAutoCompactionReopen(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const session = "11111111-1111-4111-8111-111111111111"
	const modelID = "22222222-2222-4222-8222-222222222222"
	const keyID = "33333333-3333-4333-8333-333333333333"
	const checkpoint = "CHECKPOINT_FROM_AUTOMATIC_SUMMARY"
	const reopen = "CONTINUE_AFTER_REOPEN"
	const modelOutput = 2048
	root := t.TempDir()
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	var summaries atomic.Int32
	var resumed atomic.Bool
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch r.URL.Path {
		case "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, modelID)
		case "/api/v1/api-keys":
			fmt.Fprintf(w, `{"id":%q,"api_key":"fixture"}`, keyID)
		case "/api/v1/api-keys/" + keyID:
		case "/v1/chat/completions":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			answer := "Recorded."
			if dshSyntheticCompactionRequest(body) {
				var caps struct {
					MaxTokens           int `json:"max_tokens"`
					MaxCompletionTokens int `json:"max_completion_tokens"`
				}
				if err := json.Unmarshal(body, &caps); err != nil || max(caps.MaxTokens, caps.MaxCompletionTokens) != modelOutput {
					t.Errorf("summarizer did not respect routed output cap: %+v (%v)", caps, err)
				}
				summaries.Add(1)
				answer = checkpoint
			} else if bytes.Contains(body, []byte(reopen)) {
				resumed.Store(bytes.Contains(body, []byte(checkpoint)) && !bytes.Contains(body, []byte("EARLY_HISTORY_0")))
				answer = "Resumed."
			}
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "data: {\"id\":\"fixture\",\"object\":\"chat.completion.chunk\",\"model\":\"test\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":%q},\"finish_reason\":null}]}\n\n", answer)
			fmt.Fprint(w, "data: {\"id\":\"fixture\",\"object\":\"chat.completion.chunk\",\"model\":\"test\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":1,\"total_tokens\":4}}\n\ndata: [DONE]\n\n")
		default:
			http.NotFound(w, r)
		}
		return w.Result(), nil
	})
	model := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", ContextWindowTokens: 32768, MaxOutputTokens: modelOutput}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(root, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(root, "journal"), &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}, func(context.Context, acpSession) (selectedModelConfig, error) { return model, nil })
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	row := acpSession{ID: session, DSHSessionID: session, Engine: "dsh", Model: modelID, CWD: root}
	statuses := map[string][]string{}
	run := func(prompt string, resume bool) acpTurn {
		t.Helper()
		op, err := newUUID()
		if err != nil {
			t.Fatal(err)
		}
		turn := acpTurn{Operation: op, Prompt: prompt}
		project := func(update map[string]any) error {
			if update["sessionUpdate"] == "compaction_update" {
				id, _ := update["compactionId"].(string)
				status, _ := update["status"].(string)
				statuses[id] = append(statuses[id], status)
				encoded, _ := json.Marshal(update)
				if bytes.Contains(encoded, []byte(checkpoint)) || update["summary"] != nil || update["error"] != nil {
					t.Fatal("private summary escaped into compaction status")
				}
			}
			return nil
		}
		if err := runDSHInteractiveTurn(ctx, cfg, row, &turn, resume, project); err != nil {
			t.Fatal(err)
		}
		row.Turns = append(row.Turns, turn)
		return turn
	}
	// The 32K window keeps 4096+2048 tokens of answer room, so the trigger is 26,624.
	for i := 0; i < 9 && summaries.Load() == 0; i++ {
		run(fmt.Sprintf("EARLY_HISTORY_%d\n", i)+strings.Repeat("historical material ", 1100), i > 0)
	}
	if summaries.Load() == 0 {
		t.Fatal("native pressure compaction did not invoke its summarizer in a 32K conversation")
	}
	// No lease pool: every run above and below restores the durable SDK session.
	turn := run(reopen, true)
	if !resumed.Load() || turn.Answer != "Resumed." || turn.DSHContext == nil {
		t.Fatalf("restored checkpoint/current prompt or context meter missing: resumed=%t answer=%q meter=%+v", resumed.Load(), turn.Answer, turn.DSHContext)
	}
	if len(statuses) != int(summaries.Load()) {
		t.Fatalf("expected status for each summary, received %v", statuses)
	}
	stored, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	var restored acpSession
	if err := json.Unmarshal(stored, &restored); err != nil {
		t.Fatal(err)
	}
	rows := map[string]int{}
	for i := range restored.Turns {
		message := restored.responseMessage(i, map[string]any{})
		for _, part := range message["parts"].([]map[string]string) {
			if part["type"] == "compaction" {
				if part["status"] != "" || len(part) != 3 {
					t.Fatalf("unexpected restored notice: %v", part)
				}
				rows[part["compactionId"]]++
			}
		}
	}
	for id, updates := range statuses {
		if !validUUID(id) || !slices.Equal(updates, []string{"in_progress", "completed"}) || rows[id] != 1 {
			t.Fatalf("compaction %s: transitions %v, restored %d rows", id, updates, rows[id])
		}
	}
}
