package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopDesignWorkspaceSurvivesRestart(t *testing.T) {
	previousModel := modelID
	modelID = "91c29565-0bb8-4e70-9278-032c847bddcf"
	t.Cleanup(func() { modelID = previousModel })
	const server = "729437d1-1ee3-4181-b0a6-3d4399016be7"
	const machine = "3825bddd-81da-493a-a040-7544a606a3f4"
	const operation = "1ad55c3b-8b36-44ed-8526-35b22356fe9a"
	const canvas = "e04d45f0-0496-46f1-b3f1-681c8bedf4e6"
	profile := t.TempDir()
	live, err := openEmployeeSessions(profile, server, machine, "")
	if err != nil {
		t.Fatal(err)
	}
	id, err := live.create(operation, "")
	if err != nil {
		t.Fatal(err)
	}
	design := filepath.Join(live.workspaces.defaultPath(id), ".monkeyai", "design")
	for _, name := range []string{"sessions", canvas} {
		if err := os.MkdirAll(filepath.Join(design, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	contextPath := filepath.Join(design, canvas, "design-context.json")
	marker := fmt.Sprintf(`{"canvasId":%q,"localDesignPath":%q}`, canvas, ".monkeyai/design/"+canvas)
	for name, data := range map[string]string{
		filepath.Join(design, "sessions", id+".json"): marker,
		filepath.Join(design, canvas, "canvas.json"):  `{}`,
		contextPath: fmt.Sprintf(`{"designSessionId":%q,"canvasId":%q}`, id, canvas),
	} {
		if err := os.WriteFile(name, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := live.workspaces.root.Close(); err != nil {
		t.Fatal(err)
	}
	if err := live.store.Close(); err != nil {
		t.Fatal(err)
	}
	live, err = openEmployeeSessions(profile, server, machine, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		live.workspaces.root.Close()
		live.store.Close()
	})
	host := relaySessionHost{profile: profile, live: live}
	params, _ := json.Marshal(historyQuery{SessionID: id, Limit: 20})
	workspace, err := host.workspaceQuery(t.Context(), "_monkeyai/session/workspace", params)
	if err != nil {
		t.Fatal(err)
	}
	capabilities, _ := workspace.(map[string]any)["capabilities"].(map[string]json.RawMessage)
	if string(capabilities["design"]) != marker {
		t.Fatalf("workspace design did not survive restart: %s", capabilities["design"])
	}
	page, err := host.localQuery(t.Context(), "_monkeyai/session/messages", params)
	if err != nil {
		t.Fatal(err)
	}
	meta := page.(map[string]any)["session"].(map[string]any)
	if string(meta["capabilities"].(map[string]json.RawMessage)["design"]) != marker {
		t.Fatal("source history lost its canvas metadata")
	}
	if err := os.WriteFile(contextPath, []byte(fmt.Sprintf(`{"designSessionId":%q,"canvasId":%q}`, operation, canvas)), 0600); err != nil {
		t.Fatal(err)
	}
	workspace, err = host.workspaceQuery(t.Context(), "_monkeyai/session/workspace", params)
	if err != nil || workspace.(map[string]any)["capabilities"] != nil {
		t.Fatalf("a canvas from another session was accepted: %v", err)
	}
}
