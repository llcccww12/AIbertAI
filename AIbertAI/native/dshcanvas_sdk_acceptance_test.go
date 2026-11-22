package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Opt-in full SDK -> Go approval -> real Electron worker test. The model and
// backend are synthetic; it does not exercise the employee workbench UI.
func TestDSHCanvasSDKWorkerAcceptance(t *testing.T) {
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	electron, worker, adapter := os.Getenv("DSH_TEST_ELECTRON"), os.Getenv("DSH_TEST_CANVAS_WORKER"), os.Getenv("DSH_TEST_CANVAS_ADAPTER")
	if node == "" || entry == "" || electron == "" || worker == "" || adapter == "" {
		t.Skip("patched SDK and isolated Electron canvas worker not configured")
	}
	scenario := os.Getenv("DSH_TEST_CANVAS_SCENARIO")
	customGoal := scenario == "expert_goal"
	localGoal := scenario == "local_goal" || customGoal
	const customExpert = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, electron, adapter, worker)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { stdin.Close(); _ = cmd.Wait() }()
	designBridge := &desktopDesignBridge{pending: map[string]chan designToolResult{}, send: func(v any) error { return json.NewEncoder(stdin).Encode(v) }}
	go designBridge.read(stdout)
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
	discovery := localGoal || os.Getenv("DSH_TEST_CANVAS_DISCOVERY") == "1"
	id, err := live.createInWorkspace("11111111-1111-4111-8111-111111111111", "", "", "", expertSelection{}, nil, "", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}
	session, err := live.load(id)
	if err != nil {
		t.Fatal(err)
	}
	if customGoal {
		session.ExpertID = customExpert
	} else if discovery {
		session.ExpertID = designExpertID
	}
	selection := map[string]any{"designType": "slide_deck", "designTypeSource": "user", "templateRef": nil, "designSystemRef": nil, "mediaModelId": nil}
	useTemplate := os.Getenv("DSH_TEST_CANVAS_CATALOG_FILE")
	if useTemplate != "" {
		var catalog struct {
			Revision string
			Details  map[string]map[string]struct {
				Digest string `json:"content_digest"`
			}
		}
		data, err := os.ReadFile(useTemplate)
		if err != nil {
			t.Fatal(err)
		}
		templateID := os.Getenv("DSH_TEST_CANVAS_TEMPLATE_ID")
		if templateID == "" {
			templateID = "rd-proposal-editable-template"
		}
		if err := json.Unmarshal(data, &catalog); err != nil || catalog.Revision == "" || catalog.Details["templates"][templateID].Digest == "" {
			t.Fatalf("missing catalog fixture: %v", err)
		}
		selection["templateRef"] = map[string]string{"id": templateID, "kind": "template", "version": catalog.Revision, "digest": "sha256:" + strings.TrimPrefix(catalog.Details["templates"][templateID].Digest, "sha256:")}
	}
	selectionJSON, err := json.Marshal(selection)
	if err != nil {
		t.Fatal(err)
	}
	// A new custom expert starts in an ordinary workspace. Tool discovery must open its first canvas.
	if !customGoal {
		if _, err := designBridge.call(ctx, session, nil, "design_session_open", selectionJSON); err != nil {
			t.Fatal(err)
		}
	}
	rootSession, err := workspaces.sessionRoot(session)
	if err != nil {
		t.Fatal(err)
	}
	defer rootSession.Close()
	design := workspaceDesign(rootSession, id)
	if !customGoal && len(design) == 0 {
		t.Fatal("missing canvas marker")
	}
	if !customGoal && useTemplate != "" {
		var marker struct{ LocalDesignPath string }
		var details struct{ TemplateResources struct{ Path string } }
		if err := json.Unmarshal(design, &marker); err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(session.CWD, marker.LocalDesignPath, "design-context.json"))
		if err != nil || json.Unmarshal(raw, &details) != nil || !filepath.IsLocal(details.TemplateResources.Path) {
			t.Fatalf("selected template was not prepared: %v", err)
		}
		if _, err := os.Stat(filepath.Join(session.CWD, details.TemplateResources.Path, "SKILL.md")); err != nil {
			t.Fatal(err)
		}
	}
	if strings.HasPrefix(scenario, "pptx") && useTemplate == "" {
		if _, err := designBridge.call(ctx, session, design, "design_test_style", json.RawMessage(`{}`)); err != nil {
			t.Fatal(err)
		}
	}
	catalog, err := designBridge.tools(ctx, session, design)
	if err != nil {
		t.Fatal(err)
	}
	caps, err := designBridge.call(ctx, session, design, "design_guard_capabilities", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	catalog = dshCanvasRestrictedCatalog(catalog, true, caps)
	var definition desktopDesignTool
	for _, item := range catalog {
		if item.Name == designTextTool {
			definition = item
			break
		}
	}
	if !customGoal && definition.Name == "" {
		t.Fatal("guarded text tool unavailable")
	}
	var tools []dshHostTool
	for _, item := range catalog {
		tools = append(tools, dshHostTool{Name: item.Name, Description: item.Description, InputSchema: item.Schema})
	}
	var calls atomic.Int32
	const modelID = "77777777-7777-4777-8777-777777777777"
	const keyID = "88888888-8888-4888-8888-888888888888"
	const skillID = "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	const skillText = "SKILL_VERIFIED_CANVAS_TEXT"
	const skillHint = "Canvas lettering guide"
	var skillPackage []byte
	var skillDigest string
	if customGoal {
		skillPackage = officeTestArchive(t, map[string]string{"lettering/SKILL.md": skillText})
		skillDigest = fmt.Sprintf("%x", sha256.Sum256(skillPackage))
	}
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/experts/") && strings.HasSuffix(r.URL.Path, "/manifest"):
			expertID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/experts/"), "/manifest")
			if expertID == customExpert {
				fmt.Fprintf(w, `{"available":true,"expert_id":%q,"name":"Custom canvas expert","prompt":"AUTHORIZED_CUSTOM_CANVAS_ROLE","rules":[{"content":"CUSTOM_CANVAS_RULE"}],"dsh_plugins":["tool-skill"],"skills":[{"id":%q,"name":%q,"package_sha256":%q}],"canvas_tools":[%q]}`, expertID, skillID, skillHint, skillDigest, designTextTool)
			} else {
				fmt.Fprintf(w, `{"available":true,"expert_id":%q,"name":"Platform expert"}`, expertID)
			}
			return
		case r.URL.Path == "/api/v1/experts/"+customExpert+"/skills/"+skillID+"/package":
			if r.Header.Get("Authorization") != "Bearer test" || r.URL.Query().Get("sha256") != skillDigest {
				t.Error("skill package was not requested for the authorized expert and digest")
			}
			_, _ = w.Write(skillPackage)
			return
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}],"total_count":1}`, modelID)
			return
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"test"}`, keyID)
			return
		case r.URL.Path == "/api/v1/api-keys/"+keyID && r.Method == "DELETE":
			return
		case r.URL.Path != "/v1/chat/completions":
			t.Errorf("unexpected route %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		var request any
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Error("bad model request")
			w.WriteHeader(400)
			return
		}
		wire, _ := json.Marshal(request)
		n := calls.Add(1)
		if n == 1 && !strings.Contains(string(wire), designTextTool) {
			t.Error("SDK did not expose host tool")
		}
		if n == 1 && customGoal {
			for _, want := range []string{"AUTHORIZED_CUSTOM_CANVAS_ROLE", "CUSTOM_CANVAS_RULE", skillHint, `"name":"skill"`} {
				if !strings.Contains(string(wire), want) {
					t.Errorf("custom expert missing %q", want)
				}
			}
			for _, unexpected := range []string{designCommitHTMLTool, `"name":"glob"`, skillText} {
				if strings.Contains(string(wire), unexpected) {
					t.Errorf("custom expert exposed %q before loading the skill", unexpected)
				}
			}
		}
		if n == 1 && discovery && !customGoal && (!strings.Contains(string(wire), "design-context.json") || !strings.Contains(string(wire), "templateResources.path")) {
			t.Error("SDK model request missing the selected design workspace instructions")
		}
		delta := map[string]any{"role": "assistant", "content": "finished"}
		finish := "stop"
		if (n == 1 && !customGoal) || (n == 2 && (scenario == "" || customGoal)) {
			content := fmt.Sprintf("SDK canvas acceptance %d", n)
			if customGoal {
				if !strings.Contains(string(wire), skillText) {
					t.Error("the native skill tool did not return the configured skill body")
				}
				content = skillText
			}
			args, _ := json.Marshal(map[string]any{"content": content, "x": 32 * n, "y": 32})
			delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("canvas-call-%d", n), "type": "function", "function": map[string]string{"name": designTextTool, "arguments": string(args)}}}}
			finish = "tool_calls"
		}
		if customGoal && n == 1 {
			args, _ := json.Marshal(map[string]string{"name": "aibert-" + skillID})
			delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "load-lettering", "type": "function", "function": map[string]string{"name": "skill", "arguments": string(args)}}}}
			finish = "tool_calls"
		}
		if localGoal && ((!customGoal && n == 2) || (customGoal && n == 3)) {
			saved, err := live.load(id)
			if err != nil || saved.Goal == nil || saved.Goal.DSHGoalID == "" {
				t.Errorf("missing live goal: %v", err)
				http.Error(w, "goal missing", 500)
				return
			}
			args, _ := json.Marshal(map[string]any{"goal_id": saved.Goal.DSHGoalID, "revision": saved.Goal.DSHRevision, "action": "complete"})
			delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "complete-canvas", "type": "function", "function": map[string]string{"name": "update_goal", "arguments": string(args)}}}}
			finish = "tool_calls"
		}
		limit := int32(4)
		if scenario == "pptx_two" {
			limit = 7
		}
		if strings.HasPrefix(scenario, "pptx") && n <= limit && (scenario != "pptx_workspace_recover" || n <= 2) {
			name := designCreateHTMLTool
			args := map[string]any{"title": "Slide one", "slide_index": 1}
			if scenario == "pptx_two" && n == 4 {
				args = map[string]any{"title": "Slide two", "slide_index": 2}
			}
			if n == 2 || n == 3 || scenario == "pptx_two" && (n == 5 || n == 6) {
				name = designCommitHTMLTool
				var findNode func(any) string
				findNode = func(value any) string {
					switch value := value.(type) {
					case string:
						var decoded any
						if json.Unmarshal([]byte(value), &decoded) == nil {
							return findNode(decoded)
						}
					case []any:
						for _, child := range value {
							if id := findNode(child); id != "" {
								return id
							}
						}
					case map[string]any:
						if node, ok := value["node"].(map[string]any); ok {
							if id, ok := node["id"].(string); ok && id != "" {
								return id
							}
						}
						for _, child := range value {
							if id := findNode(child); id != "" {
								return id
							}
						}
					}
					return ""
				}
				input := request
				if scenario == "pptx_two" && n >= 5 {
					if object, ok := request.(map[string]any); ok {
						if messages, ok := object["messages"].([]any); ok {
							for i := len(messages) - 1; i >= 0; i-- {
								if findNode(messages[i]) != "" {
									input = messages[i]
									break
								}
							}
						}
					}
				}
				id := findNode(input)
				if id == "" {
					t.Errorf("slide id not found in model context: %s", wire)
					http.Error(w, "missing slide", 500)
					return
				}
				var findArtifact func(any) string
				findArtifact = func(value any) string {
					switch value := value.(type) {
					case string:
						var decoded any
						if json.Unmarshal([]byte(value), &decoded) == nil {
							return findArtifact(decoded)
						}
					case []any:
						for _, child := range value {
							if path := findArtifact(child); path != "" {
								return path
							}
						}
					case map[string]any:
						if path, ok := value["artifact_path"].(string); ok && path != "" {
							return path
						}
						for _, child := range value {
							if path := findArtifact(child); path != "" {
								return path
							}
						}
					}
					return ""
				}
				artifact := findArtifact(input)
				if !filepath.IsLocal(artifact) {
					t.Errorf("invalid fixture artifact path: %q", artifact)
					http.Error(w, "invalid artifact", 500)
					return
				}
				if n == 2 || n == 5 {
					name = "write"
					content := "<html><body>Slide one</body></html>"
					if n == 5 {
						content = "<html><body>Slide two</body></html>"
					}
					args = map[string]any{"file_path": artifact, "content": content}
				} else {
					args = map[string]any{"node_id": id}
				}
			} else if n == limit {
				name = designPreparePPTXTool
				args = map[string]any{}
			}
			arg, _ := json.Marshal(args)
			delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("canvas-call-%d", n), "type": "function", "function": map[string]string{"name": name, "arguments": string(arg)}}}}
			finish = "tool_calls"
		}
		w.Header().Set("Content-Type", "text/event-stream")
		emit := func(v any) { b, _ := json.Marshal(v); fmt.Fprintf(w, "data: %s\n\n", b) }
		emit(map[string]any{"id": "chat", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}})
		emit(map[string]any{"id": "chat", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": finish}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4}})
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer model.Close()
	client := &client{model.URL, model.Client()}
	gw := &dshGateway{client: client, authorization: func(context.Context) (authorization, error) { return authorization{Access: "test"}, nil }}
	selected := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(dir, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(dir, "journal"), gw, func(context.Context, acpSession) (selectedModelConfig, error) { return selected, nil })
	if err != nil {
		t.Fatal(err)
	}
	session.Model = modelID

	op := "22222222-2222-4222-8222-222222222222"
	turn := acpTurn{Operation: op, Prompt: "Create a canvas text node", Started: time.Now().UTC()}
	session.Engine, session.DSHSessionID, session.Turns, session.State = "dsh", id, []acpTurn{turn}, "running"
	live.mu.Lock()
	err = live.save(session)
	live.mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if scenario == "pending" {
		if e := dshConnectorDispatch(root, id, op, "prior-connector"); e != nil {
			t.Fatal(e)
		}
	}
	local := &desktopRuntimeAccess{protocol: "monkeyai.token.fixture"}
	host := &relaySessionHost{live: live, local: local, designBridge: designBridge}
	if scenario == "expert_switch" {
		local.address = "127.0.0.1:5173"
		request := &http.Request{Host: local.address, Header: http.Header{}}
		request.Header.Set("Sec-WebSocket-Protocol", "monkeyai.v1, "+local.protocol)
		live.client, live.auth = client, authorization{Access: "test"}
		const plainID = "99999999-9999-4999-8999-999999999999"
		const configuredID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
		host.importedResources = &importedDesktopResources{Experts: map[string]importedExpertRuntime{configuredID: {DesignTools: []string{designTextTool}}}}
		for _, check := range []struct {
			expert                          string
			attachment, canWrite, canSelect bool
		}{
			{designExpertID, false, true, true}, {plainID, false, false, false},
			{plainID, true, false, true}, {configuredID, false, true, false}, {plainID, false, false, false},
		} {
			session.State = "idle"
			if err := live.save(session); err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(map[string]any{"sessionId": session.ID, "expertId": check.expert})
			if _, err := host.setDesktopSessionExpert(ctx, raw); err != nil {
				t.Fatal(err)
			}
			session, err = live.load(id)
			if err != nil {
				t.Fatal(err)
			}
			turn.CanvasNodes = nil
			if check.attachment {
				turn.CanvasNodes = []acpCanvasNode{{ID: "selected-id"}}
			}
			bridges, instructions, err := host.dshTurnHostTools(ctx, nil, request, session, &turn)
			if err != nil {
				t.Fatal(err)
			}
			names := map[string]bool{}
			for _, bridge := range bridges {
				for _, item := range bridge.Tools {
					names[item.Name] = true
				}
			}
			if names[designTextTool] != check.canWrite || names[designSelectionTool] != check.canSelect ||
				check.expert != designExpertID && (instructions != "" || names[designCommitHTMLTool] || names[designReadTool]) {
				t.Fatalf("expert %s selection=%v: exposed tools %v instructions=%q", check.expert, check.attachment, names, instructions)
			}
		}
		return
	}
	if localGoal {
		live.dsh = cfg
		turn.GoalCheckpoint = &goalCheckpoint{}
		session.Mode = "bypassPermissions"
		session.GoalRun, session.Turns = op, []acpTurn{turn}
		session.Goal = &desktopGoal{ID: op, Generation: 1, Objective: turn.Prompt, Status: "active", Current: true, Limits: desktopGoalLimit{Tokens: 1000}}
		if err := live.save(session); err != nil {
			t.Fatal(err)
		}
		local.address = "127.0.0.1:5173"
		request := &http.Request{Host: local.address, Header: http.Header{}}
		request.Header.Set("Sec-WebSocket-Protocol", "monkeyai.v1, "+local.protocol)
		runCtx := context.WithValue(ctx, goalRuntimeContext{}, newGoalRuntime())
		if customGoal {
			live.client, live.auth = client, authorization{Access: "test"}
			var cleanup func()
			runCtx, cleanup, err = host.bindDSHExpert(runCtx, session)
			if err != nil {
				t.Fatal(err)
			}
			defer cleanup()
		}
		runCtx = context.WithValue(runCtx, dshSideEffectRootKey{}, root)
		if err := host.runDSHGoalTurn(runCtx, nil, request, session, &turn, false, true, func(map[string]any) error {
			live.mu.Lock()
			defer live.mu.Unlock()
			current := live.sessions[id]
			current.Turns = []acpTurn{turn}
			return live.save(current)
		}); err != nil {
			t.Fatal(err, turn.ToolRuns)
		}
		row, _ := live.load(id)
		if row.Goal == nil || row.Goal.Status != "completed" || len(turn.ToolRuns) == 0 || turn.ToolRuns[0].Report.Failed != 0 {
			t.Fatalf("background canvas goal failed: %+v %+v", row.Goal, turn.ToolRuns)
		}
		var marker struct{ LocalDesignPath string }
		if err := json.Unmarshal(workspaceDesign(rootSession, id), &marker); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(session.CWD, marker.LocalDesignPath, "canvas.json"))
		expectedText := "SDK canvas acceptance 1"
		if customGoal {
			expectedText = skillText
		}
		if err != nil || !strings.Contains(string(data), expectedText) {
			t.Fatalf("canvas not written: %s %v", data, err)
		}
		return
	}
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
		runCtx = context.WithValue(runCtx, dshWorkspaceAuthorizeKey{}, func(context.Context) error { return nil })
		if scenario != "recover" && scenario != "recover_pending" {
			runCtx = context.WithValue(runCtx, dshSideEffectRootKey{}, root)
		}
		bridge := host.dshCanvasBridge(runCtx, ws, r, session, &turn, design, catalog)
		runCtx = context.WithValue(runCtx, dshWorkspaceBridgeKey{}, host.dshWorkspaceBridge(runCtx, ws, session, &turn))
		bridges := []dshHostToolBridge{{Tools: tools, Handle: bridge}}
		if discovery {
			var discoverErr error
			var instructions string
			bridges, instructions, discoverErr = host.dshTurnHostTools(runCtx, ws, r, session, &turn)
			found := map[string]bool{}
			for _, item := range bridges {
				for _, tool := range item.Tools {
					found[tool.Name] = true
				}
			}
			if discoverErr != nil || !found[designCreateHTMLTool] || !found[designCommitHTMLTool] || !found[designPreparePPTXTool] {
				done <- fmt.Errorf("canvas tools unavailable through discovery: %v %v", found, discoverErr)
				return
			}
			runCtx = context.WithValue(runCtx, dshExpertInstructionsKey{}, instructions)
		}
		runCtx = context.WithValue(runCtx, dshHostToolBridgesKey{}, bridges)
		result := make(chan error, 1)
		go func() {
			err := runDSHTurn(runCtx, cfg, session, &turn, false, func(map[string]any) error {
				live.mu.Lock()
				defer live.mu.Unlock()
				current := live.sessions[id]
				current.Turns = []acpTurn{turn}
				return live.save(current)
			})
			if err != nil {
				t.Logf("DSH execution stopped: %v; tool runs: %+v", err, turn.ToolRuns)
				ws.CloseNow()
			}
			result <- err
		}()
		permissionsExpected := 1
		if scenario == "" {
			permissionsExpected = 2
		}
		if strings.HasPrefix(scenario, "pptx") {
			permissionsExpected = 4
		}
		if scenario == "pptx_workspace_recover" {
			permissionsExpected = 2
		}
		if scenario == "pptx_two" {
			permissionsExpected = 7
		}
		for i := 0; i < permissionsExpected; i++ {
			var reply acpPermissionReply
			if e = wsjson.Read(runCtx, ws, &reply); e != nil {
				done <- e
				return
			}
			var key string
			_ = json.Unmarshal(reply.ID, &key)
			if waiter, ok := permissions.Load(key); ok {
				waiter.(*acpPermissionWaiter).resolve(reply)
			}
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
	permissionsExpected := 1
	if scenario == "" {
		permissionsExpected = 2
	}
	if strings.HasPrefix(scenario, "pptx") {
		permissionsExpected = 4
	}
	if scenario == "pptx_workspace_recover" {
		permissionsExpected = 2
	}
	if scenario == "pptx_two" {
		permissionsExpected = 7
	}
	for i := 0; i < permissionsExpected; i++ {
		var frame struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		for frame.Method != "session/request_permission" {
			if err = wsjson.Read(ctx, ws, &frame); err != nil {
				t.Fatalf("waiting for approval %d after %d model requests: %v", i, calls.Load(), err)
			}
		}
		if !strings.HasPrefix(scenario, "pptx") && !strings.Contains(string(frame.Params), "SDK canvas acceptance") {
			t.Fatalf("approval lacks content: %s", frame.Params)
		}
		var rpcID any
		_ = json.Unmarshal(frame.ID, &rpcID)
		choice := "allow"
		if scenario == "deny" {
			choice = "deny"
		}
		if err = wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": rpcID, "result": map[string]any{"outcome": map[string]string{"outcome": "selected", "optionId": choice}}}); err != nil {
			t.Fatal(err)
		}
	}

	err = <-done
	claims, claimErr := dshConnectorClaims(root, id)
	if claimErr != nil {
		t.Fatal(claimErr)
	}
	if scenario == "recover" || scenario == "recover_pending" {
		if err != nil || len(claims) != 1 || claims[0].State != "result_ready" {
			t.Fatalf("missing pre-barrier claim: %v %+v", err, claims)
		}
		turn.Failed, turn.Acked, turn.FailureCode, turn.Ended = true, true, "dsh_canvas_outcome_unknown", time.Now().UTC()
		session.Turns, session.State = []acpTurn{turn}, "error"
		live.mu.Lock()
		live.dsh = cfg
		e := live.save(session)
		live.mu.Unlock()
		if e != nil {
			t.Fatal(e)
		}
		// Reconstruct the platform host from the persisted session, as after a
		// process restart; no in-memory turn state is used for reconciliation.
		reopened, e := os.OpenRoot(dir)
		if e != nil {
			t.Fatal(e)
		}
		defer reopened.Close()
		restored, e := readDesktopSession(reopened, id)
		if e != nil {
			t.Fatal(e)
		}
		live = &acpLive{store: &desktopSessionStore{Root: reopened}, dsh: cfg, sessions: map[string]acpSession{id: restored}}
		host.live = live
		if scenario == "recover_pending" {
			pendingPath := filepath.Join(dir, "journal", id, "pending.json")
			pending, _ := json.Marshal(dshLocalRecord{SessionID: id, CallID: "99999999-9999-4999-8999-999999999999", ModelID: modelID, State: "dispatched"})
			if e := os.WriteFile(pendingPath, pending, 0600); e != nil {
				t.Fatal(e)
			}
			before := calls.Load()
			if e := host.recoverDSHCanvasSession(ctx, id); e == nil {
				t.Fatal("unknown model charge allowed recovery")
			}
			if calls.Load() != before || dshConnectorAssertSettled(root, id) == nil {
				t.Fatal("unknown model charge changed state or replayed")
			}
			if e := os.Remove(pendingPath); e != nil {
				t.Fatal(e)
			}
		}
		if e := host.recoverDSHCanvasSession(ctx, id); e != nil {
			t.Fatalf("gateway settlement and history recovery failed: %v", e)
		}

		claims, e = dshConnectorClaims(root, id)
		if e != nil || len(claims) != 1 || claims[0].State != "reconciled" || dshConnectorAssertSettled(root, id) != nil {
			t.Fatalf("recovery did not release: %+v %v", claims, e)
		}
		if e = dshConnectorDispatch(root, id, op, "canvas-call-1"); e == nil {
			t.Fatal("original write can replay")
		}
		// Simulate a second crash after durable claim release but before the
		// platform session update. Recovery must be repeatable without dispatch.
		before := calls.Load()
		live.mu.Lock()
		e = live.save(session)
		live.mu.Unlock()
		if e != nil {
			t.Fatal(e)
		}
		if e = host.recoverDSHCanvasSession(ctx, id); e != nil || calls.Load() != before {
			t.Fatalf("repeat recovery failed or replayed: %v, context=%v, model calls before=%d after=%d", e, ctx.Err(), before, calls.Load())
		}
		return
	}
	if scenario == "deny" {
		if err != nil || len(claims) != 0 {
			t.Fatalf("denial created claim: %v %+v", err, claims)
		}
		return
	}
	if scenario == "pending" {
		if err == nil || len(claims) != 1 || claims[0].CallID != "prior-connector" {
			t.Fatalf("pending connector allowed canvas write: %v %+v", err, claims)
		}
		var marker struct{ LocalDesignPath string }
		_ = json.Unmarshal(design, &marker)
		data, e := os.ReadFile(filepath.Join(session.CWD, filepath.FromSlash(marker.LocalDesignPath), "canvas.json"))
		if e != nil || strings.Contains(string(data), "SDK canvas acceptance") {
			t.Fatalf("canvas changed under pending connector: %s %v", data, e)
		}
		return
	}
	if scenario == "drop" || scenario == "conflict" {
		if err == nil || len(claims) != 1 || claims[0].State != "dispatch_claimed" {
			t.Fatalf("ambiguous write not retained: %v %+v", err, claims)
		}
		if e := dshConnectorDispatch(root, id, op, "later"); e == nil {
			t.Fatal("unknown canvas write allowed later side effect")
		}
		reopened, e := os.OpenRoot(dir)
		if e != nil {
			t.Fatal(e)
		}
		defer reopened.Close()
		if e = dshConnectorAssertSettled(reopened, id); e == nil {
			t.Fatal("restart forgot unknown canvas write")
		}
		if scenario == "drop" {
			var marker struct{ LocalDesignPath string }
			_ = json.Unmarshal(design, &marker)
			data, e := os.ReadFile(filepath.Join(session.CWD, filepath.FromSlash(marker.LocalDesignPath), "canvas.json"))
			if e != nil || !strings.Contains(string(data), "SDK canvas acceptance") {
				t.Fatalf("write did not precede lost reply: %s %v", data, e)
			}
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if scenario == "pptx_recover" || scenario == "pptx_workspace_recover" {
		turn.Failed, turn.Acked, turn.FailureCode, turn.Ended = true, true, "dsh_canvas_outcome_unknown", time.Now().UTC()
		if scenario == "pptx_workspace_recover" {
			turn.FailureCode = "dsh_write_outcome_unknown"
		}
		session.Turns, session.State = []acpTurn{turn}, "error"
		live.mu.Lock()
		live.dsh = cfg
		e := live.save(session)
		live.mu.Unlock()
		if e != nil {
			t.Fatal(e)
		}
		reopened, e := os.OpenRoot(dir)
		if e != nil {
			t.Fatal(e)
		}
		defer reopened.Close()
		restored, e := readDesktopSession(reopened, id)
		if e != nil {
			t.Fatal(e)
		}
		live = &acpLive{store: &desktopSessionStore{Root: reopened}, dsh: cfg, sessions: map[string]acpSession{id: restored}}
		host.live = live
		before := calls.Load()
		recover := host.recoverDSHCanvasSession
		expected := 4
		if scenario == "pptx_workspace_recover" {
			recover = host.recoverDSHWorkspaceSession
			expected = 2
		}
		if e := recover(ctx, id); e != nil {
			t.Fatalf("mixed canvas and workspace recovery: %v", e)
		}
		recovered, e := live.load(id)
		if e != nil || calls.Load() != before || !recovered.Turns[0].DSHReconciled || recovered.Turns[0].DSHProjectionSeq == nil || len(recovered.Turns[0].ToolRuns) != expected {
			t.Fatalf("recovery replayed or lost tools: %+v %v", recovered, e)
		}
		return
	}
	if strings.HasPrefix(scenario, "pptx") {
		expected := 4
		if scenario == "pptx_two" {
			expected = 7
		}
		if err != nil || turn.Answer != "finished" || len(claims) != expected {
			t.Fatalf("PPTX flow failed: %v %+v", err, claims)
		}
		for _, claim := range claims {
			if (claim.Kind != "canvas" && claim.Kind != "workspace") || claim.State != "reconciled" {
				t.Fatalf("PPTX receipt not reconciled: %+v", claim)
			}
		}
		count := 1
		if scenario == "pptx_two" {
			count = 2
		}
		if len(turn.ToolRuns) != expected || !strings.Contains(turn.ToolRuns[expected-1].Output, fmt.Sprintf(`"slide_count":%d`, count)) {
			t.Fatalf("PPTX preparation result: %+v", turn.ToolRuns)
		}
		if scenario == "pptx_two" {
			var result struct {
				ManifestPath string `json:"manifest_path"`
			}
			if json.Unmarshal([]byte(turn.ToolRuns[expected-1].Output), &result) != nil || !filepath.IsLocal(result.ManifestPath) {
				t.Fatal("invalid PPTX manifest")
			}
			exporter, bundle := os.Getenv("DSH_TEST_PPTX_EXPORTER"), os.Getenv("DSH_TEST_PPTX_BUNDLE")
			if exporter == "" || bundle == "" {
				t.Fatal("PPTX exporter fixture is not configured")
			}
			output := filepath.Join(dir, "presentation.pptx")
			export := exec.CommandContext(ctx, electron, exporter)
			export.Env = append(os.Environ(), "DSH_PPTX_WORKSPACE="+session.CWD, "DSH_PPTX_MANIFEST="+filepath.Join(session.CWD, result.ManifestPath), "DSH_PPTX_BUNDLE="+bundle, "DSH_PPTX_OUTPUT="+output)
			if data, e := export.CombinedOutput(); e != nil {
				t.Fatalf("PPTX export: %v %s", e, data)
			}
			file, e := os.Open(output)
			if e != nil {
				t.Fatal(e)
			}
			defer file.Close()
			if e = verifyEditableDeck(file, count); e != nil {
				t.Fatalf("exported PPTX: %v", e)
			}
		}
		return
	}
	if turn.Answer != "finished" || calls.Load() < 3 {
		t.Fatalf("unfinished SDK turn: %+v", turn)
	}
	if len(claims) != 2 {
		t.Fatalf("claims: %+v", claims)
	}
	for _, claim := range claims {
		if claim.State != "reconciled" || claim.SDKSeq == nil || claim.Kind != "canvas" || claim.Output == nil || !strings.HasPrefix(*claim.Output, "v1:") || strings.Contains(*claim.Output, "canvas") {
			t.Fatalf("unverified or plaintext claim: %+v", claim)
		}
	}
	var marker struct {
		LocalDesignPath string `json:"localDesignPath"`
	}
	_ = json.Unmarshal(design, &marker)
	data, err := os.ReadFile(filepath.Join(session.CWD, filepath.FromSlash(marker.LocalDesignPath), "canvas.json"))
	if err != nil || !strings.Contains(string(data), "SDK canvas acceptance 1") || !strings.Contains(string(data), "SDK canvas acceptance 2") {
		t.Fatalf("canvas result: %s %v", data, err)
	}
}
