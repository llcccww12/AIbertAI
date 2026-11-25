package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// The model creates a skill by chat; the user approves the exact content; the
// skill is saved to the platform, synced, and loaded in the same conversation.
func TestDSHCreatePlatformSkillByChatPatchedSDK(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Skip("patched SDK build not configured")
	}
	const modelID = "33333333-3333-4333-8333-333333333333"
	const skillID = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	ctx, cancel := context.WithTimeout(t.Context(), 120*time.Second)
	defer cancel()
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	workspaces, err := openDesktopWorkspaces(root, filepath.Join(dir, "workspaces"), "")
	if err != nil {
		t.Fatal(err)
	}
	live := &acpLive{store: &desktopSessionStore{Root: root}, cwd: dir, workspaces: workspaces, operations: map[string]string{}, sessions: map[string]acpSession{}}
	id, err := live.createInWorkspace("11111111-1111-4111-8111-111111111111", "", "", "", expertSelection{}, nil, "", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}
	session, err := live.load(id)
	if err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var saved []byte // the uploaded package
	var calls atomic.Int32
	loadedBody := make(chan bool, 1)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/skills" && r.Method == http.MethodPost:
			file, _, err := r.FormFile("package")
			if err != nil {
				http.Error(w, "missing package", 400)
				return
			}
			data, _ := io.ReadAll(file)
			mu.Lock()
			saved = data
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"id":%q,"name":"weekly-report"}`, skillID)
		case r.URL.Path == "/api/v1/skills":
			mu.Lock()
			data := saved
			mu.Unlock()
			if data == nil {
				fmt.Fprint(w, `{"skills":[]}`)
				return
			}
			sum := sha256.Sum256(data)
			sha := hex.EncodeToString(sum[:])
			fmt.Fprintf(w, `{"skills":[{"id":%q,"name":"weekly-report","description":"Write the weekly report","package_sha256":%q,"download_path":"/api/v1/skills/%s/package?sha256=%s"}]}`, skillID, sha, skillID, sha)
		case r.URL.Path == "/api/v1/skills/"+skillID+"/package":
			mu.Lock()
			_, _ = w.Write(saved)
			mu.Unlock()
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, modelID)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == http.MethodPost:
			fmt.Fprint(w, `{"id":"44444444-4444-4444-8444-444444444444","api_key":"fixture"}`)
		case r.URL.Path == "/v1/chat/completions":
			var request any
			_ = json.NewDecoder(r.Body).Decode(&request)
			wire, _ := json.Marshal(request)
			n := calls.Add(1)
			delta, finish := map[string]any{"role": "assistant", "content": "done"}, "stop"
			call := func(name string, args any) {
				arg, _ := json.Marshal(args)
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("call-%d", n), "type": "function", "function": map[string]string{"name": name, "arguments": string(arg)}}}}
				finish = "tool_calls"
			}
			switch n {
			case 1:
				if !strings.Contains(string(wire), dshCreateSkillTool) {
					t.Error("create_platform_skill not offered to the model")
				}
				call(dshCreateSkillTool, map[string]string{"name": "weekly-report", "description": "Write the weekly report", "content": "E2E-SKILL-BODY: three bullet points."})
			case 2:
				call("skill", map[string]string{"name": "weekly-report"})
			default:
				loadedBody <- strings.Contains(string(wire), "E2E-SKILL-BODY")
			}
			w.Header().Set("Content-Type", "text/event-stream")
			emit := func(v any) { b, _ := json.Marshal(v); fmt.Fprintf(w, "data: %s\n\n", b) }
			emit(map[string]any{"id": "chat", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}})
			emit(map[string]any{"id": "chat", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": finish}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4}})
			fmt.Fprint(w, "data: [DONE]\n\n")
		default:
			fmt.Fprint(w, `{}`)
		}
	}))
	defer backend.Close()
	c := &client{backend.URL, backend.Client()}
	live.client, live.auth = c, authorization{Access: "test"}
	skills := filepath.Join(dir, "dsh", "platform-skills")
	gw := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "test"}, nil }}
	selected := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, PlatformSkillsDir: skills, Home: filepath.Join(dir, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(dir, "journal"), gw, func(context.Context, acpSession) (selectedModelConfig, error) { return selected, nil })
	if err != nil {
		t.Fatal(err)
	}
	cfg.SyncPlatformSkills = func(ctx context.Context) error { return c.syncDSHPlatformSkills(ctx, live.auth, skills) }
	if err := cfg.SyncPlatformSkills(ctx); err != nil {
		t.Fatal(err)
	}
	cfg.SessionSkills = func(session acpSession) error {
		current, err := live.load(session.ID)
		if err != nil {
			return err
		}
		prefs := current.ConnectorPreferences.normalized()
		return materializeDSHSessionSkills(skills, dshSessionSkillsDir(skills, session.ID), func(id string) bool { return prefs.Enabled[dshSkillPreferencePrefix+id] })
	}
	cfg.LeasePool = &dshLeasePool{}
	defer cfg.LeasePool.close(context.Background())
	live.dsh = cfg

	op := "22222222-2222-4222-8222-222222222222"
	turn := acpTurn{Operation: op, Prompt: "Make me a skill for weekly reports, then use it", Started: time.Now().UTC()}
	session.Model, session.Engine, session.DSHSessionID, session.Turns, session.State = modelID, "dsh", id, []acpTurn{turn}, "running"
	live.mu.Lock()
	err = live.save(session)
	live.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	local := &desktopRuntimeAccess{protocol: "monkeyai.token.fixture"}
	host := &relaySessionHost{live: live, local: local}
	done := make(chan error, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, e := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"monkeyai.v1"}})
		if e != nil {
			done <- e
			return
		}
		defer ws.CloseNow()
		permissions := &sync.Map{}
		runCtx := context.WithValue(ctx, acpPermissionContextKey{}, permissions)
		runCtx = context.WithValue(runCtx, dshSideEffectRootKey{}, root)
		bridges, instructions, e := host.dshTurnHostTools(runCtx, ws, r, session, &turn)
		if e != nil || !strings.Contains(instructions, dshCreateSkillTool) {
			done <- fmt.Errorf("create tools not discovered: %v", e)
			return
		}
		runCtx = context.WithValue(runCtx, dshExpertInstructionsKey{}, instructions)
		runCtx = context.WithValue(runCtx, dshHostToolBridgesKey{}, bridges)
		result := make(chan error, 1)
		go func() {
			result <- runDSHTurn(runCtx, cfg, session, &turn, false, func(map[string]any) error {
				live.mu.Lock()
				defer live.mu.Unlock()
				current := live.sessions[id]
				current.Turns = []acpTurn{turn}
				return live.save(current)
			})
		}()
		replies := make(chan acpPermissionReply, 1)
		go func() {
			var reply acpPermissionReply
			if wsjson.Read(runCtx, ws, &reply) == nil {
				replies <- reply
			}
		}()
		var reply acpPermissionReply
		select {
		case reply = <-replies:
		case e = <-result:
			done <- fmt.Errorf("turn ended without approval: %v", e)
			return
		}
		var key string
		_ = json.Unmarshal(reply.ID, &key)
		if waiter, ok := permissions.Load(key); ok {
			waiter.(*acpPermissionWaiter).resolve(reply)
		}
		done <- <-result
	}))
	local.address = server.Listener.Addr().String()
	server.Start()
	defer server.Close()
	ws, _, err := websocket.Dial(ctx, server.URL, &websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", local.protocol}})
	if err != nil {
		t.Fatal(err)
	}
	defer ws.CloseNow()
	var frame struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	for frame.Method != "session/request_permission" {
		readCtx, stop := context.WithTimeout(ctx, 30*time.Second)
		err = wsjson.Read(readCtx, ws, &frame)
		stop()
		if err != nil {
			select {
			case e := <-done:
				t.Fatalf("turn ended before approval: %v; model calls %d; runs %+v", e, calls.Load(), turn.ToolRuns)
			default:
			}
			t.Fatalf("waiting for approval after %d model calls: %v; runs %+v", calls.Load(), err, turn.ToolRuns)
		}
	}
	if !strings.Contains(string(frame.Params), "E2E-SKILL-BODY") {
		t.Fatalf("approval does not show the skill content: %s", frame.Params)
	}
	var rpcID any
	_ = json.Unmarshal(frame.ID, &rpcID)
	if err = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": rpcID, "result": map[string]any{"outcome": map[string]string{"outcome": "selected", "optionId": "allow"}}}); err != nil {
		t.Fatal(err)
	}
	if err = <-done; err != nil {
		t.Fatalf("turn failed: %v; runs: %+v", err, turn.ToolRuns)
	}
	mu.Lock()
	pkg := saved
	mu.Unlock()
	archive, err := zip.NewReader(bytes.NewReader(pkg), int64(len(pkg)))
	if err != nil || len(archive.File) != 1 || archive.File[0].Name != "weekly-report/SKILL.md" {
		t.Fatalf("uploaded package invalid: %v", err)
	}
	select {
	case ok := <-loadedBody:
		if !ok {
			t.Fatal("the new skill was not loadable in the same conversation")
		}
	default:
		t.Fatal("model never received the skill tool result")
	}
}
