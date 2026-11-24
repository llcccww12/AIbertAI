package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The recovery allowlist must be exactly the guarded canvas writes the bridge
// can dispatch. A second hand-maintained list drifts as soon as a tool is added
// to only one of them, which silently makes every failure of that tool
// unrecoverable.
func TestDSHRecoverableCanvasToolsMatchGuardedWrites(t *testing.T) {
	universe := []string{
		designReadTool, designSelectionTool, designTextTool, designCreateHTMLTool, designCommitHTMLTool,
		designProductionTool, designPreparePPTXTool, designStyleChoiceTool,
		"media_get_image_model", "media_generate_image", "media_edit_image",
		"canvas_add_artifact_asset", "canvas_capture_preview", "design_jury_submit",
		"canvas_create_image", "canvas_create_video", "canvas_compose_media_project",
		"canvas_create_layer_group", "canvas_group_story_assets", "canvas_update_node",
		"canvas_delete_node", "design_catalog_search", "design_catalog_detail", "design_catalog_logo_batch",
	}
	for _, name := range universe {
		want := dshCanvasGuarded(name) && designWriteTool(name)
		if got := dshRecoverableCanvasTool(name); got != want {
			t.Errorf("dshRecoverableCanvasTool(%q) = %v, want %v", name, got, want)
		}
	}
}

// Image generation mutates the canvas through the same guarded claim as the
// canvas_* writes, so an unknown outcome must be reconcilable.
func TestDSHRecoverableCanvasToolCoversImageGeneration(t *testing.T) {
	for _, name := range []string{"media_generate_image", "canvas_create_image", designTextTool, designCreateHTMLTool, designCommitHTMLTool, designPreparePPTXTool} {
		if !dshRecoverableCanvasTool(name) {
			t.Errorf("%s is not recoverable", name)
		}
	}
	for _, name := range []string{designReadTool, designSelectionTool, "media_get_image_model", designStyleChoiceTool, "canvas_update_node", "canvas_delete_node"} {
		if dshRecoverableCanvasTool(name) {
			t.Errorf("%s must not be treated as a recoverable canvas write", name)
		}
	}
}

func writeRecoveryTestFile(t *testing.T, root *os.Root, name, body string) {
	t.Helper()
	if err := root.MkdirAll(filepath.Dir(name), 0700); err != nil && filepath.Dir(name) != "." {
		t.Fatal(err)
	}
	if err := root.WriteFile(name, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

// A canvas write that timed out never recorded a result. The canvas file is
// then the only evidence, and it must decide whether the write landed instead
// of leaving the session locked forever.
func TestDSHSettleStaleCanvasWriteUsesCanvasEvidence(t *testing.T) {
	const (
		session   = "11111111-1111-4111-8111-111111111111"
		operation = "22222222-2222-4222-8222-222222222222"
		call      = "33333333-3333-4333-8333-333333333333"
		canvasID  = "44444444-4444-4444-8444-444444444444"
	)
	for _, tc := range []struct{ name, canvas, decision string }{
		{"unchanged", `{"revision":7,"nodes":[]}`, "observed_not_applied"},
		{"changed", `{"revision":8,"nodes":[{"id":"a"}]}`, "observed_applied"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			before := fmt.Sprintf("%x", sha256.Sum256([]byte(`{"revision":7,"nodes":[]}`)))
			writeRecoveryTestFile(t, root, ".aibertai/design/sessions/"+session+".json",
				`{"canvasId":"`+canvasID+`","localDesignPath":".aibertai/design/`+canvasID+`"}`)
			writeRecoveryTestFile(t, root, ".aibertai/design/"+canvasID+"/design-context.json",
				`{"designSessionId":"`+session+`","canvasId":"`+canvasID+`"}`)
			writeRecoveryTestFile(t, root, ".aibertai/design/"+canvasID+"/canvas.json", tc.canvas)
			if err = dshConnectorDispatch(root, session, operation, call); err != nil {
				t.Fatal(err)
			}
			claim, err := connectorPending(root, session, operation, call)
			if err != nil {
				t.Fatal(err)
			}
			claim.Kind, claim.CanvasBefore = "canvas", &before
			if err = writeConnectorClaim(root, claim); err != nil {
				t.Fatal(err)
			}
			callUse := &modelToolUse{ID: call}
			callUse.Function.Name = "media_generate_image"
			row := acpSession{ID: session, Turns: []acpTurn{{
				Operation: operation, Failed: true, Acked: true, Ended: time.Now().UTC(),
				FailureCode: "dsh_canvas_outcome_unknown",
				ToolRuns:    []workspaceToolRun{{DSHCallID: call, Call: callUse}},
			}}}
			settled, err := dshSettleStaleCanvasWrite(root, row)
			if err != nil || !settled {
				t.Fatalf("settle: %v %v", settled, err)
			}
			settledClaim, err := readConnectorClaim(root, connectorClaimPath(session, operation, call), session)
			if err != nil || settledClaim.State != "user_reviewed" || settledClaim.Review == nil || settledClaim.Review.Decision != tc.decision {
				t.Fatalf("claim not settled from evidence: %+v %v", settledClaim, err)
			}
			if dshConnectorAssertSettled(root, session) != nil {
				t.Fatal("settled canvas write still blocks admission")
			}
		})
	}
}

// One failed tool must not wedge the session: a claim left by an ended
// operation only refuses that same call, and its sentinel is retired so the
// next call can dispatch.
func TestDSHConnectorDispatchOnlyRefusesTheSameCall(t *testing.T) {
	const (
		session   = "55555555-5555-4555-8555-555555555555"
		firstOp   = "66666666-6666-4666-8666-666666666666"
		secondOp  = "77777777-7777-4777-8777-777777777777"
		firstCall = "88888888-8888-4888-8888-888888888888"
		nextCall  = "99999999-9999-4999-8999-999999999999"
	)
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err = dshConnectorDispatch(root, session, firstOp, firstCall); err != nil {
		t.Fatal(err)
	}
	// The first call never recorded a result, as after an unknown outcome.
	if err = dshConnectorDispatch(root, session, firstOp, firstCall); err == nil {
		t.Fatal("the unresolved call was allowed to replay")
	}
	if err = dshConnectorDispatch(root, session, secondOp, nextCall); err != nil {
		t.Fatalf("a different call was blocked by an ended operation: %v", err)
	}
	// The ended operation is still on record even though its sentinel moved on.
	claims, err := dshConnectorClaims(root, session)
	if err != nil {
		t.Fatal(err)
	}
	var kept bool
	for _, claim := range claims {
		if claim.CallID == firstCall && claim.State == "dispatch_claimed" {
			kept = true
		}
	}
	if !kept {
		t.Fatal("the unresolved claim was dropped instead of kept as evidence")
	}
}

func TestDSHHostToolReasonIsModelFacing(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want string
	}{
		{errors.New("dsh_canvas_outcome_unknown: x"), "canvas_outcome_unknown"},
		{errors.New("dsh_connector_outcome_unknown"), "connector_outcome_unknown"},
		{errors.New("tool_permission_cancelled"), "tool_denied"},
		{errors.New("dsh_canvas_changed"), "canvas_conflict"},
		{errors.New("something_else"), "host_tool_denied"},
	} {
		if got := dshHostToolReason(tc.err); !strings.HasPrefix(got, tc.want+":") {
			t.Errorf("reason for %q = %q, want prefix %q", tc.err, got, tc.want)
		}
	}
}

// A design-service refusal before generation (no image model) is a normal tool
// failure: the claim settles as not applied and the session stays usable, but
// only while the canvas still matches its pre-dispatch digest.
func TestDSHCanvasReleaseRejectedSettlesNotApplied(t *testing.T) {
	const (
		session   = "aaaaaaaa-1111-4111-8111-111111111111"
		operation = "bbbbbbbb-2222-4222-8222-222222222222"
		call      = "cccccccc-3333-4333-8333-333333333333"
		canvasID  = "dddddddd-4444-4444-8444-444444444444"
	)
	if !dshCanvasDefiniteRejection(errors.New("design_media_model_unavailable")) || dshCanvasDefiniteRejection(errors.New("design_tool_unavailable")) {
		t.Fatal("only a known pre-execution refusal is definite")
	}
	for _, tc := range []struct {
		name, canvas string
		released     bool
	}{
		{"unchanged", `{"revision":7,"nodes":[]}`, true},
		{"changed", `{"revision":8,"nodes":[{"id":"a"}]}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			before := fmt.Sprintf("%x", sha256.Sum256([]byte(`{"revision":7,"nodes":[]}`)))
			writeRecoveryTestFile(t, root, ".aibertai/design/sessions/"+session+".json",
				`{"canvasId":"`+canvasID+`","localDesignPath":".aibertai/design/`+canvasID+`"}`)
			writeRecoveryTestFile(t, root, ".aibertai/design/"+canvasID+"/design-context.json",
				`{"designSessionId":"`+session+`","canvasId":"`+canvasID+`"}`)
			writeRecoveryTestFile(t, root, ".aibertai/design/"+canvasID+"/canvas.json", tc.canvas)
			if err = dshConnectorDispatch(root, session, operation, call); err != nil {
				t.Fatal(err)
			}
			effect := &dshSideEffect{root: root, session: session, operation: operation, call: call}
			if err = dshCanvasMarkClaim(effect, before); err != nil {
				t.Fatal(err)
			}
			err = dshCanvasReleaseRejected(effect, acpSession{ID: session})
			if (err == nil) != tc.released {
				t.Fatalf("release = %v, want released %v", err, tc.released)
			}
			settled := dshConnectorAssertSettled(root, session) == nil
			if settled != tc.released {
				t.Fatalf("settled = %v, want %v", settled, tc.released)
			}
		})
	}
}
