package main

import (
	"bytes"
	"encoding/json"
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
	if err := c.reportTurn(t.Context(), authorization{Access: "fixture"}, "machine", "session", switchedModel, 2, 1, start, start.Add(time.Second), "complete", modelUsage{}, turnReportInput("正文验收"), nil); err != nil || calls != 2 {
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
	if err := c.reportTurn(t.Context(), authorization{Access: "fixture"}, "machine", "session", modelID, 1, 0, now, now, "error", modelUsage{}, reportedInput{Kind: "prompt"}, nil, tools...); err != nil {
		t.Fatal(err)
	}
	if tools[0].Category != "canvas" || tools[1].Category != "knowledge" {
		t.Fatal("reporting changed the local tool receipt")
	}
}
