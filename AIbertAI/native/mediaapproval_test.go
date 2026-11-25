package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Image and video generation bill the account, so they ask even when the
// conversation runs in full-access mode; other tools keep the bypass.
func TestMediaGenerationAsksInFullAccess(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	h := &relaySessionHost{local: &desktopRuntimeAccess{}, live: &acpLive{dir: t.TempDir(), sessions: map[string]acpSession{}}}
	if err := h.live.save(acpSession{ID: id, Mode: modeFullAccess}); err != nil {
		t.Fatal(err)
	}
	ask := func(name string, bypass bool) (string, error) {
		frames := make(chan string, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ws, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			defer ws.CloseNow()
			permissions := &sync.Map{}
			ctx := context.WithValue(r.Context(), acpPermissionContextKey{}, permissions)
			go func() {
				for {
					var frame map[string]any
					if wsjson.Read(ctx, ws, &frame) != nil {
						return
					}
					if waiter, ok := permissions.Load(frame["id"]); ok {
						waiter.(*acpPermissionWaiter).reply <- acpPermissionReply{Result: json.RawMessage(`{"outcome":{"outcome":"selected","optionId":"allow"}}`)}
					}
				}
			}()
			_, err = h.requestToolApproval(ctx, ws, id, "call-1", mediaGenerationTitle(name), name, "edit", map[string]string{"prompt": "a cat"}, bypass, false)
			if err != nil {
				frames <- "error: " + err.Error()
			}
			close(frames)
		}))
		defer server.Close()
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		ws, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
		if err != nil {
			return "", err
		}
		defer ws.CloseNow()
		seen := ""
		for {
			var frame struct{ Method string }
			readCtx, stop := context.WithTimeout(ctx, 2*time.Second)
			err := wsjson.Read(readCtx, ws, &frame)
			stop()
			if err != nil {
				break
			}
			if frame.Method == "session/request_permission" {
				seen = frame.Method
				_ = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": "permission-call-1", "result": map[string]any{"outcome": map[string]string{"outcome": "selected", "optionId": "allow"}}})
			}
		}
		if e, ok := <-frames; ok {
			return seen, &testError{e}
		}
		return seen, nil
	}
	if got, err := ask("media_generate_image", false); err != nil || got != "session/request_permission" {
		t.Fatalf("generation did not ask in full access: %q %v", got, err)
	}
	if got, _ := ask("canvas_update_node", true); got != "" {
		t.Fatal("other tools lost the full-access bypass")
	}
	goal := context.WithValue(t.Context(), goalRuntimeContext{}, newGoalRuntime())
	if mediaApprovalBypass(t.Context(), "media_generate_image") || !mediaApprovalBypass(goal, "media_generate_image") || !mediaApprovalBypass(t.Context(), "canvas_update_node") {
		t.Fatal("goal runs keep the full-access bypass; interactive turns ask")
	}
	if mediaGenerationTitle("media_generate_video") != "生成视频" || mediaGenerationTitle("media_edit_image") != "编辑图片" {
		t.Fatal("titles")
	}
}

type testError struct{ s string }

func (e *testError) Error() string { return e.s }
