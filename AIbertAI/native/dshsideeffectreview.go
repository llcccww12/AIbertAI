package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
)

// A review is sensitive: only an authenticated user with access to the
// session may receive its arguments. It is evidence for a human decision,
// not evidence that the side effect did or did not execute.
type dshSideEffectReview struct {
	SessionID         string              `json:"sessionId"`
	OperationID       string              `json:"operationId"`
	CallID            string              `json:"callId"`
	ToolName          string              `json:"toolName"`
	ObservedFileState string              `json:"observedFileState,omitempty"`
	ObservedContent   *string             `json:"observedContent,omitempty"`
	WorkspaceRoot     string              `json:"workspaceRoot,omitempty"`
	DisplayName       string              `json:"displayName,omitempty"`
	Arguments         json.RawMessage     `json:"arguments"`
	ClaimState        string              `json:"claimState"`
	ClaimKind         string              `json:"claimKind,omitempty"`
	Target            *dshConnectorTarget `json:"target,omitempty"`
	Fingerprint       string              `json:"fingerprint"`
}

// Inspect the committed platform session and pending claim. An unbound,
// missing, ambiguous, changed or malformed call is never offered for review.
func dshInspectSideEffect(root *os.Root, session string) (dshSideEffectReview, error) {
	fail := func() (dshSideEffectReview, error) {
		return dshSideEffectReview{}, errors.New("dsh_side_effect_review_unavailable")
	}
	if root == nil || !validUUID(session) {
		return fail()
	}
	pending, err := readConnectorClaim(root, "dsh-connector-dispatch/"+session+"/.pending", session)
	if err != nil {
		return fail()
	}
	claim, err := connectorPending(root, session, pending.OperationID, pending.CallID)
	if err != nil || claim.State != "dispatch_claimed" && claim.State != "result_ready" {
		return fail()
	}
	row, err := readDesktopSession(root, session)
	if err != nil || row.Engine != "dsh" || row.ID != session {
		return fail()
	}
	var selected *workspaceToolRun
	matches := 0
	for _, turn := range row.Turns {
		if turn.Operation != claim.OperationID {
			continue
		}
		for i := range turn.ToolRuns {
			run := &turn.ToolRuns[i]
			if run.DSHCallID == claim.CallID {
				selected = run
				matches++
			}
		}
	}
	if matches != 1 || selected.Call == nil || selected.Call.Function.Name == "" || len(selected.Call.Function.Name) > 1024 || len(selected.Call.Function.Arguments) > 8*maxToolText {
		return fail()
	}
	if claim.Target != nil {
		if claim.Target.ResourceRevision != row.ResourceRevision || (connectorTool{ConnectorID: claim.Target.ConnectorID, Name: claim.Target.Tool}).alias() != selected.Call.Function.Name {
			return fail()
		}
	} else if claim.Kind != "workspace" && selected.Call.Function.Name != "write" && selected.Call.Function.Name != "edit" {
		return fail()
	}
	args, err := strictDSHJSON([]byte(selected.Call.Function.Arguments))
	if err != nil || len(args) == 0 || args[0] != '{' {
		return fail()
	}
	review := dshSideEffectReview{SessionID: session, OperationID: claim.OperationID, CallID: claim.CallID, ToolName: selected.Call.Function.Name, DisplayName: selected.DisplayName, Arguments: args, ClaimState: claim.State, ClaimKind: claim.Kind, Target: claim.Target}
	if claim.Target == nil && (review.ToolName == "write" || review.ToolName == "edit") {
		review.WorkspaceRoot = row.CWD
	}
	// Include the entire claim in the fingerprint without displaying its output.
	// This makes a delayed result invalidate any earlier review confirmation.
	binding, err := json.Marshal(struct {
		Review dshSideEffectReview `json:"review"`
		Claim  dshConnectorClaim   `json:"claim"`
	}{review, claim})
	if err != nil {
		return fail()
	}
	digest := sha256.Sum256(binding)
	review.Fingerprint = hex.EncodeToString(digest[:])
	return review, nil
}

// Check a human response against a fresh review. This function deliberately
// does not clear a claim or synthesize an SDK result. A future caller must also
// prove the executor has stopped before persisting a distinct reviewed outcome.
func dshValidateSideEffectReview(root *os.Root, session, fingerprint, decision string, reauthorize func(dshConnectorTarget) error) (dshSideEffectReview, error) {
	if decision != "observed_applied" && decision != "observed_not_applied" {
		return dshSideEffectReview{}, errors.New("dsh_side_effect_review_invalid")
	}
	review, err := dshInspectSideEffect(root, session)
	if err != nil || fingerprint == "" || fingerprint != review.Fingerprint {
		return dshSideEffectReview{}, errors.New("dsh_side_effect_review_changed")
	}
	if review.Target != nil && (reauthorize == nil || reauthorize(*review.Target) != nil) {
		return dshSideEffectReview{}, errors.New("dsh_side_effect_review_changed")
	}
	return review, nil
}
