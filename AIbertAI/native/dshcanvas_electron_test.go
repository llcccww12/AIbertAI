package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Exercises the Go bridge against the actual Electron canvas worker through a
// stdio adapter. It does not launch the SDK or employee workbench.
func TestDSHCanvasRealElectronBridge(t *testing.T) {
	electron := os.Getenv("DSH_TEST_ELECTRON")
	worker := os.Getenv("DSH_TEST_CANVAS_WORKER")
	adapter := os.Getenv("DSH_TEST_CANVAS_ADAPTER")
	if electron == "" || worker == "" || adapter == "" {
		t.Skip("isolated Electron canvas worker not configured")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
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
	bridge := &desktopDesignBridge{pending: map[string]chan designToolResult{}, send: func(v any) error { return json.NewEncoder(stdin).Encode(v) }}
	go bridge.read(stdout)
	session := acpSession{ID: "22222222-2222-4222-8222-222222222222", CWD: t.TempDir()}
	opened, err := bridge.call(ctx, session, nil, "design_session_open", json.RawMessage(`{"designType":"slide_deck","designTypeSource":"user","templateRef":null,"designSystemRef":null,"mediaModelId":null}`))
	if err != nil {
		t.Fatal(err)
	}
	design := json.RawMessage(opened)
	var marker struct {
		LocalDesignPath string `json:"localDesignPath"`
	}
	if err := json.Unmarshal(design, &marker); err != nil {
		t.Fatal(err)
	}
	capability, err := bridge.call(ctx, session, design, "design_guard_capabilities", json.RawMessage(`{}`))
	if err != nil || !strings.Contains(capability, `"version":4`) {
		t.Fatalf("guard handshake: %s %v", capability, err)
	}
	canvas := filepath.Join(session.CWD, filepath.FromSlash(marker.LocalDesignPath), "canvas.json")
	contents, err := os.ReadFile(canvas)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(contents)
	revision := int64(0)
	first, err := bridge.callGuarded(ctx, session, design, designTextTool, json.RawMessage(`{"content":"approved"}`), &revision, fmt.Sprintf("%x", sum), "", "", "")
	if err != nil || !strings.Contains(first, `"revision":1`) {
		t.Fatalf("first guarded call: %s %v", first, err)
	}
	_, err = bridge.callGuarded(ctx, session, design, designTextTool, json.RawMessage(`{"content":"stale"}`), &revision, fmt.Sprintf("%x", sum), "", "", "")
	if err == nil || !strings.Contains(err.Error(), "canvas_revision_conflict") {
		t.Fatalf("stale call: %v", err)
	}
	contents, err = os.ReadFile(canvas)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct{ Nodes []struct{ Content string } }
	if json.Unmarshal(contents, &doc) != nil || len(doc.Nodes) != 1 || doc.Nodes[0].Content != "approved" {
		t.Fatalf("unexpected canvas: %s", contents)
	}
	// Simulate the user's style decision through the worker's UI operation.
	if _, err := bridge.call(ctx, session, design, "design_test_style", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(session.CWD)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	var id struct{ CanvasID string }
	if err := json.Unmarshal(design, &id); err != nil {
		t.Fatal(err)
	}
	artifactDigest, err := dshCanvasArtifacts(root, id.CanvasID, contents)
	if err != nil {
		t.Fatal(err)
	}
	canvasDigest := sha256.Sum256(contents)
	contextData, err := os.ReadFile(filepath.Join(session.CWD, filepath.FromSlash(marker.LocalDesignPath), "design-context.json"))
	if err != nil {
		t.Fatal(err)
	}
	contextSum := sha256.Sum256(contextData)
	contextDigest := fmt.Sprintf("%x", contextSum)
	revision = 1
	created, err := bridge.callGuarded(ctx, session, design, designCreateHTMLTool, json.RawMessage(`{"title":"Slide 1","slide_index":1}`), &revision, fmt.Sprintf("%x", canvasDigest), artifactDigest, contextDigest, "")
	if err != nil {
		t.Fatalf("guarded slide: %s %v", created, err)
	}
	var slide struct {
		Node         struct{ ID string }
		ArtifactPath string `json:"artifact_path"`
	}
	if json.Unmarshal([]byte(created), &slide) != nil || slide.Node.ID == "" {
		t.Fatalf("invalid slide: %s", created)
	}
	contents, err = os.ReadFile(canvas)
	if err != nil {
		t.Fatal(err)
	}
	artifactDigest, err = dshCanvasArtifacts(root, id.CanvasID, contents)
	if err != nil {
		t.Fatal(err)
	}
	canvasDigest = sha256.Sum256(contents)
	revision = 2
	_, err = bridge.callGuarded(ctx, session, design, designPreparePPTXTool, json.RawMessage(`{}`), &revision, fmt.Sprintf("%x", canvasDigest), artifactDigest, contextDigest, "")
	if err == nil || !strings.Contains(err.Error(), "pptx_slide_not_committed") {
		t.Fatalf("draft export: %v", err)
	}
	if err := os.WriteFile(filepath.Join(session.CWD, filepath.FromSlash(slide.ArtifactPath)), []byte("<html><body>Slide one</body></html>"), 0600); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]string{"node_id": slide.Node.ID})
	_, err = bridge.callGuarded(ctx, session, design, designCommitHTMLTool, args, &revision, fmt.Sprintf("%x", canvasDigest), artifactDigest, contextDigest, "")
	if err == nil || !strings.Contains(err.Error(), "canvas_artifact_changed") {
		t.Fatalf("changed HTML: %v", err)
	}
	artifactDigest, err = dshCanvasArtifacts(root, id.CanvasID, contents)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bridge.callGuarded(ctx, session, design, designCommitHTMLTool, args, &revision, fmt.Sprintf("%x", canvasDigest), artifactDigest, contextDigest, ""); err != nil {
		t.Fatal(err)
	}
	contents, err = os.ReadFile(canvas)
	if err != nil {
		t.Fatal(err)
	}
	artifactDigest, err = dshCanvasArtifacts(root, id.CanvasID, contents)
	if err != nil {
		t.Fatal(err)
	}
	canvasDigest = sha256.Sum256(contents)
	revision = 3
	prepared, err := bridge.callGuarded(ctx, session, design, designPreparePPTXTool, json.RawMessage(`{}`), &revision, fmt.Sprintf("%x", canvasDigest), artifactDigest, contextDigest, "")
	if err != nil || !strings.Contains(prepared, `"slide_count":1`) {
		t.Fatalf("prepared: %s %v", prepared, err)
	}
}
