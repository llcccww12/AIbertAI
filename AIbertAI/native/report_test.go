package main

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTurnReportReplaysSamePayloadAtAcknowledgedCursor(t *testing.T) {
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
	if err := c.reportTurn(t.Context(), authorization{Access: "fixture"}, "machine", "session", 2, 1, start, start.Add(time.Second), "complete", modelUsage{}, nil); err != nil || calls != 2 {
		t.Fatalf("expected acknowledged replay, got %v after %d requests", err, calls)
	}
}
