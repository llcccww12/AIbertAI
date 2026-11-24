package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestTurnReportReplaysSamePayloadAtAcknowledgedCursor(t *testing.T) {
	const switchedModel = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	c, err := newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	var first []byte
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(body, []byte(`"model_id":"`+switchedModel+`"`)) {
			t.Fatal("turn report did not use the model selected for this turn")
		}
		if !bytes.Contains(body, []byte(`"input":{"kind":"prompt","text":"正文验收"}`)) {
			t.Fatal("turn report did not preserve the captured prompt")
		}
		var data string
		var status int
		if calls == 1 {
			if r.URL.Query().Get("after_turn") != "1" {
				t.Fatal("second turn did not start at the first turn's cursor")
			}
			first = body
			data, status = `{"error":{"code":"cursor_mismatch","references":{"acked_turn":2}}}`, http.StatusConflict
		} else {
			if r.URL.Query().Get("after_turn") != "2" || !bytes.Equal(first, body) {
				t.Fatal("retry must replay the same report at the server's cursor")
			}
			data, status = `{"acked_turn":2}`, http.StatusOK
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(data))}, nil
	})
	start := time.Now()
	if err := c.reportTurn(t.Context(), authorization{Access: "fixture"}, "machine", "session", switchedModel, 2, 1, start, start.Add(time.Second), "complete", modelUsage{}, turnReportInput("正文验收"), turnReportFacts{}); err != nil || calls != 2 {
		t.Fatalf("expected acknowledged replay, got %v after %d requests", err, calls)
	}
}

func TestTurnReportInputTruncatesUnicode(t *testing.T) {
	prompt := strings.Repeat("文", 8192)
	got := turnReportInput(prompt + "\x00尾")
	if got.Text != prompt || !got.TextTruncated {
		t.Fatal("expected exactly 8192 characters and a truncation marker")
	}
}

func TestTurnReportUsesBackendCategoriesForDesktopTools(t *testing.T) {
	c, err := newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	tools := []reportedTool{
		{Category: "canvas", Name: "canvas_create_html", Calls: 1},
		{Category: "knowledge", Name: "knowledge_search", Calls: 1},
		{Category: "connector", Name: "resource_connector", Calls: 1},
	}
	want := slices.Clone(tools)
	want[0].Category, want[1].Category = "builtin", "builtin"
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var payload struct {
			Turns []struct{ Tools []reportedTool }
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if len(payload.Turns) != 1 || !slices.Equal(payload.Turns[0].Tools, want) {
			t.Fatalf("unexpected reported tools: %+v", payload.Turns)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"acked_turn":1}`))}, nil
	})
	now := time.Now()
	if err := c.reportTurn(t.Context(), authorization{Access: "fixture"}, "machine", "session", modelID, 1, 0, now, now, "error", modelUsage{}, reportedInput{Kind: "prompt"}, turnReportFacts{}, tools...); err != nil {
		t.Fatal(err)
	}
	if tools[0].Category != "canvas" || tools[1].Category != "knowledge" {
		t.Fatal("reporting changed the local tool receipt")
	}
}

func TestTurnReportCountsCacheAndMemberTokens(t *testing.T) {
	c, err := newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var payload struct{ Turns []map[string]json.RawMessage }
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || len(payload.Turns) != 1 {
			t.Fatalf("decode: %v", err)
		}
		got = payload.Turns[0]
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"acked_turn":1}`))}, nil
	})
	call := func(name, args string) *modelToolUse {
		c := &modelToolUse{}
		c.Function.Name, c.Function.Arguments = name, args
		return c
	}
	// DSH counts cache outside inputTokens; Usage merges members' uncached tokens.
	input, output, used := int64(100), int64(30), int64(5000)
	turn := acpTurn{Usage: modelUsage{Input: &input, Output: &output},
		Tokens:         turnTokenFacts{CacheRead: 500, CacheWrite: 50, SubagentInput: 40, SubagentOutput: 10, SubagentCacheRead: 7},
		DSHContext:     &dshContextUsage{Used: &used, Window: 128000},
		DSHCompactions: []dshCompactionNotice{{ID: "a", Status: "completed"}, {ID: "b", Status: "failed"}},
		ToolRuns: []workspaceToolRun{
			{Call: call("write", `{"file_path":"a.txt"}`), FileOperation: "created"},
			{Call: call("edit", `{"file_path":"b.txt"}`)},
		}}
	now := time.Now()
	if err := c.reportTurn(t.Context(), authorization{Access: "fixture"}, "machine", "session", modelID, 1, 0, now, now, "complete", turn.Usage, reportedInput{Kind: "prompt"}, turn.reportFacts("")); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"usage":          `{"cache_creation_input_tokens":50,"cache_read_input_tokens":500,"input_tokens":610,"output_tokens":20}`,
		"subagent_usage": `{"cache_creation_input_tokens":0,"cache_read_input_tokens":7,"input_tokens":47,"output_tokens":10}`,
		"context_used":   `5000`, "context_window": `128000`, "compactions": `1`,
		"files_changed": `{"created":1,"updated":1,"deleted":0}`,
	} {
		if string(got[key]) != want {
			t.Fatalf("%s = %s, want %s", key, got[key], want)
		}
	}

	// Turns without DSH token facts keep their original usage shape, so an
	// unacknowledged report replays with the same hash.
	if err := c.reportTurn(t.Context(), authorization{Access: "fixture"}, "machine", "session", modelID, 1, 0, now, now, "complete", turn.Usage, reportedInput{Kind: "prompt"}, turnReportFacts{}); err != nil {
		t.Fatal(err)
	}
	if string(got["usage"]) != `{"input_tokens":100,"output_tokens":30}` || got["subagent_usage"] != nil || got["files_changed"] != nil || got["compactions"] != nil {
		t.Fatalf("legacy usage changed: %s", got["usage"])
	}
}

func TestTurnReportAggregatesToolsWithinBackendLimit(t *testing.T) {
	var tools []reportedTool
	for i := range 600 {
		tools = append(tools, reportedTool{Category: "builtin", Name: "read", ResourceID: "read", Calls: 1, Failed: i % 2, DurationMS: 10})
	}
	tools = append(tools, reportedTool{Category: "agent", Name: "subagent", ResourceID: "subagent", Calls: 1, DurationMS: 7})
	facts, truncated := turnReportTools(tools)
	want := []reportedTool{{Category: "builtin", Name: "read", ResourceID: "read", Calls: 600, Failed: 300, DurationMS: 6000}, {Category: "agent", Name: "subagent", ResourceID: "subagent", Calls: 1, DurationMS: 7}}
	if truncated || !slices.Equal(facts, want) {
		t.Fatalf("aggregated %+v truncated=%v", facts, truncated)
	}

	tools = tools[:0]
	for i := range 600 {
		tools = append(tools, reportedTool{Category: "connector", Name: fmt.Sprintf("tool-%d", i), Calls: 1})
	}
	facts, truncated = turnReportTools(tools)
	if !truncated || len(facts) != maxReportedTools || facts[0].Name != "tool-0" {
		t.Fatalf("expected %d facts and a truncation marker, got %d %v", maxReportedTools, len(facts), truncated)
	}
	c, err := newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var payload struct {
			Turns []struct {
				Tools     []reportedTool
				Truncated bool
			}
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || len(payload.Turns) != 1 || len(payload.Turns[0].Tools) != maxReportedTools || !payload.Turns[0].Truncated {
			t.Fatalf("unexpected report: %v", err)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"acked_turn":1}`))}, nil
	})
	now := time.Now()
	if err := c.reportTurn(t.Context(), authorization{Access: "fixture"}, "machine", "session", modelID, 1, 0, now, now, "complete", modelUsage{}, reportedInput{Kind: "prompt"}, turnReportFacts{}, tools...); err != nil {
		t.Fatal(err)
	}
}
