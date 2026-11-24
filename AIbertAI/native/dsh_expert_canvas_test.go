package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"
)

func TestDSHCanvasExpertManifestDiscoveryAndDispatch(t *testing.T) {
	const cloud = "11111111-1111-4111-8111-111111111111"
	const other = "22222222-2222-4222-8222-222222222222"
	const canvas = "33333333-3333-4333-8333-333333333333"
	live, err := openEmployeeSessions(t.TempDir(), cloud, other, "")
	if err != nil {
		t.Fatal(err)
	}
	defer live.workspaces.root.Close()
	defer live.store.Close()
	id, err := live.createSession(canvas, "", "", "", expertSelection{}, nil, "", desktopMediaModels{}, true)
	if err != nil {
		t.Fatal(err)
	}
	row := live.sessions[id]
	root, err := live.workspaces.sessionRoot(row)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	base := filepath.Join(workspaceDataDir, "design")
	for _, dir := range []string{"sessions", canvas} {
		if err = root.MkdirAll(filepath.Join(base, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	patchFixtureFiles(t, root, map[string]string{
		filepath.Join(base, "sessions", id+".json"):        fmt.Sprintf(`{"canvasId":%q,"localDesignPath":%q}`, canvas, filepath.ToSlash(filepath.Join(base, canvas))),
		filepath.Join(base, canvas, "design-context.json"): fmt.Sprintf(`{"designSessionId":%q,"canvasId":%q}`, id, canvas),
		filepath.Join(base, canvas, "canvas.json"):         `{"revision":1,"nodes":[]}`,
	})
	worker := &desktopDesignBridge{pending: map[string]chan designToolResult{}}
	schema := json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)
	catalog := []desktopDesignTool{{Name: designReadTool, Schema: schema}, {Name: designSelectionTool, Schema: schema}, {Name: designTextTool, Schema: schema}}
	workerReads := 0
	worker.send = func(v any) error {
		event := v.(map[string]any)
		var result any
		switch event["name"] {
		case "design_tool_catalog":
			result = catalog
		case "design_guard_capabilities":
			result = map[string]any{"version": 4, "tools": []string{designReadTool, designSelectionTool, designTextTool}}
		case designReadTool:
			workerReads++
			result = map[string]string{"title": "fixture canvas"}
		default:
			return fmt.Errorf("unexpected worker call %v", event["name"])
		}
		b, err := json.Marshal(result)
		if err != nil {
			return err
		}
		worker.pending[event["id"].(string)] <- designToolResult{Result: b}
		return nil
	}
	live.client, err = newBackendClient("https://fixture.invalid", false)
	if err != nil {
		t.Fatal(err)
	}
	live.auth = authorization{Access: "fixture"}
	h := &relaySessionHost{live: live, local: &desktopRuntimeAccess{address: "localhost", protocol: "monkeyai.token.fixture"}, designBridge: worker}
	req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	req.Header.Set("Sec-WebSocket-Protocol", "monkeyai.v1, monkeyai.token.fixture")
	for _, tc := range []struct {
		name, expert, field string
		imported, want      []string
		attached            bool
	}{
		{"new cloud expert", cloud, `,"canvas_tools":["canvas_get_document"]`, nil, []string{designReadTool}, false},
		{"other expert", other, "", nil, nil, false},
		{"explicitly empty overrides local", cloud, `,"canvas_tools":[]`, []string{designReadTool}, nil, false},
		{"missing inherits local", cloud, "", []string{designReadTool}, []string{designReadTool}, false},
		{"no wildcard", cloud, `,"canvas_tools":["canvas_*"]`, nil, nil, false},
		{"selected nodes only", other, `,"canvas_tools":[]`, nil, []string{designSelectionTool}, true},
		{"built-in design", designExpertID, "", nil, []string{designReadTool, designSelectionTool, designTextTool}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manifests := 0
			live.client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/api/v1/experts/"+tc.expert+"/manifest" || r.Header.Get("Authorization") != "Bearer fixture" {
					return nil, fmt.Errorf("unexpected manifest request")
				}
				manifests++
				w := httptest.NewRecorder()
				fmt.Fprintf(w, `{"expert_id":%q,"available":true%s}`, tc.expert, tc.field)
				return w.Result(), nil
			})
			h.importedResources = &importedDesktopResources{Experts: map[string]importedExpertRuntime{tc.expert: {DesignTools: tc.imported}}}
			call := modelToolUse{ID: "projected"}
			call.Function.Name, call.Function.Arguments = designReadTool, "{}"
			turn := acpTurn{Operation: canvas, ToolRuns: []workspaceToolRun{{DSHCallID: "call", Pending: true, Call: &call}}}
			if tc.attached {
				turn.CanvasNodes = []acpCanvasNode{{ID: "selected"}}
			}
			row.ExpertID, row.Turns = tc.expert, []acpTurn{turn}
			live.sessions[id] = row
			ctx := t.Context()
			wantManifests := 0
			if tc.expert != designExpertID {
				bound, release, err := h.bindDSHExpert(ctx, row)
				if err != nil {
					t.Fatal(err)
				}
				defer release()
				ctx, wantManifests = bound, 1
			}
			bridges, _, err := h.dshTurnHostTools(ctx, nil, req, row, &turn)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, bridge := range bridges {
				for _, tool := range bridge.Tools {
					names = append(names, tool.Name)
				}
			}
			if !slices.Equal(names, tc.want) {
				t.Fatalf("discovery: got %v, want %v", names, tc.want)
			}
			// Pass the unfiltered worker catalog for denied cases to exercise the
			// execution check independently of the SDK's discovery filter.
			handle := h.dshCanvasBridge(ctx, nil, req, row, &turn, workspaceDesign(root, id), catalog)
			allowed := slices.Contains(tc.want, designReadTool)
			if allowed {
				handle = bridges[0].Handle
			}
			raw, err := json.Marshal(dshMutation{SessionID: id, CallID: "call", Name: designReadTool, Arguments: json.RawMessage(`{}`)})
			if err != nil {
				t.Fatal(err)
			}
			before := workerReads
			_, err = handle(raw)
			if (err == nil) != allowed || (workerReads == before+1) != allowed || manifests != wantManifests {
				t.Fatalf("dispatch allowed=%v error=%v worker reads=%d manifest requests=%d", allowed, err, workerReads-before, manifests)
			}
		})
	}
}
