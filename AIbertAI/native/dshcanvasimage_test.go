package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestDSHCanvasImageSourceAndReceipt(t *testing.T) {
	const source = ".aibertai/media/generated/11111111-1111-4111-8111-111111111111.png"
	for _, raw := range []string{`{"source_path":"../file.png"}`, `{"source_path":"` + source + `","x":1}`, `{"source_path":"` + source + `","anchor_node_id":null}`, `{"source_path":"` + source + `","title":null}`, `{"source_path":"` + source + `","title":"` + strings.Repeat("字", 161) + `"}`} {
		if _, err := dshCanvasImageSource(json.RawMessage(raw)); err == nil {
			t.Fatalf("accepted invalid arguments: %s", raw)
		}
	}
	input := json.RawMessage(`{"source_path":"` + source + `","title":"验证图"}`)
	if got, err := dshCanvasImageSource(input); err != nil || got != source {
		t.Fatalf("valid image input: %s %v", got, err)
	}
	workspace, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer workspace.Close()
	journal, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	const session = "22222222-2222-4222-8222-222222222222"
	const other = "33333333-3333-4333-8333-333333333333"
	const operation = "44444444-4444-4444-8444-444444444444"
	const call = "generate-01"
	const original = "original contents"
	if err := workspace.MkdirAll(".aibertai/media/generated", 0700); err != nil {
		t.Fatal(err)
	}
	if err := workspace.WriteFile(source, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(original)))
	data := `{"source_path":"` + source + `","source_sha256":"` + digest + `"}`
	model := modelToolUse{ID: "image-call"}
	model.Function.Name = "media_generate_image"
	row := acpSession{ID: session, DSHSessionID: session, Turns: []acpTurn{{Operation: operation, ToolRuns: []workspaceToolRun{{Call: &model, DSHCallID: call, Output: data}}}}}
	output := dshCanvasResultDigest(data)
	seq := int64(3)
	if err := journal.MkdirAll("dsh-connector-dispatch/"+session, 0700); err != nil {
		t.Fatal(err)
	}
	claim := dshConnectorClaim{SessionID: session, OperationID: operation, CallID: call, State: "reconciled", Kind: "canvas", Output: &output, SDKSeq: &seq}
	if err := writeConnectorClaim(journal, claim); err != nil {
		t.Fatal(err)
	}
	verify := func(want bool) {
		t.Helper()
		got, err := dshGeneratedImageDigest(journal, workspace, row, source)
		if (err == nil) != want || want && got != digest {
			t.Fatalf("expected source authorized=%t; got %q %v", want, got, err)
		}
	}
	verify(true)
	model.Function.Name = "media_edit_image"
	verify(true)
	model.Function.Name = "media_remove_background"
	verify(false)
	model.Function.Name = "media_generate_image"
	row.ID, row.DSHSessionID = other, other
	verify(false)
	row.ID, row.DSHSessionID = session, session
	run := &row.Turns[0].ToolRuns[0]
	run.Pending = true
	verify(false)
	run.Pending = false
	run.Report.Failed = 1
	verify(false)
	run.Report.Failed = 0
	run.Output = " " + data
	verify(false)
	run.Output = data
	if err := workspace.WriteFile(source, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	verify(false)
	if err := workspace.Rename(source, source+".saved"); err != nil {
		t.Fatal(err)
	}
	if err := workspace.Symlink(source+".saved", source); err != nil {
		t.Fatal(err)
	}
	verify(false)
}

func TestGuardedCanvasImageAdvertisesOnlyNewNode(t *testing.T) {
	const name = "canvas_create_image"
	item := desktopDesignTool{Name: name, Schema: json.RawMessage(`{"type":"object","properties":{"source_path":{"type":"string"},"title":{"type":"string"},"anchor_node_id":{"type":"string"},"x":{"type":"number"}},"required":["source_path"],"additionalProperties":false}`)}
	const caps = `{"version":4,"tools":["canvas_create_image"]}`
	got := dshCanvasRestrictedCatalog([]desktopDesignTool{item}, true, caps)
	if len(got) != 1 {
		t.Fatal("missing guarded image placement")
	}
	var fields struct {
		Properties map[string]struct {
			Pattern   string
			MaxLength int
		}
		Required []string
	}
	if json.Unmarshal(got[0].Schema, &fields) != nil || len(fields.Properties) != 2 || fields.Properties["source_path"].Pattern != dshGeneratedImagePattern || fields.Properties["title"].MaxLength != 160 || len(fields.Required) != 1 || fields.Required[0] != "source_path" {
		t.Fatal(string(got[0].Schema))
	}
	if len(dshCanvasRestrictedCatalog([]desktopDesignTool{item}, false, caps)) != 0 || len(dshCanvasRestrictedCatalog([]desktopDesignTool{item}, true, "{}")) != 0 || dshExpertAllowsCanvas("custom", []string{"media_generate_image"}, false, name) {
		t.Fatal("image placement bypassed availability or expert selection")
	}
}
