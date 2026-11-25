package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"unicode/utf8"
)

const dshConnectorMaxResult = 256 << 10

type dshWorkspaceWitness struct {
	Session, Operation, Call, Name      string
	ArgumentsSHA                        string
	HistoryOperation, GoalID, SDKGoalID string
	CallSeq, Turn, Step                 int64
}

type dshConnectorClaim struct {
	WorkspaceBoundary *dshWorkspaceWitness     `json:"workspaceBoundary,omitempty"`
	SessionID         string                   `json:"sessionId"`
	OperationID       string                   `json:"operationId"`
	CallID            string                   `json:"callId"`
	State             string                   `json:"state"`
	Kind              string                   `json:"kind,omitempty"`
	Output            *string                  `json:"output,omitempty"`
	SDKSeq            *int64                   `json:"sdkSeq,omitempty"`
	Target            *dshConnectorTarget      `json:"target,omitempty"`
	Review            *dshUserSideEffectReview `json:"review,omitempty"`
	CanvasBefore      *string                  `json:"canvasBefore,omitempty"`
}

func connectorClaimPath(session, operation, call string) string {
	sum := sha256.Sum256([]byte(session + "\x00" + operation + "\x00" + call))
	return fmt.Sprintf("dsh-connector-dispatch/%s/%x", session, sum)
}
func syncConnectorDir(root *os.Root, dir string) error {
	f, err := root.Open(dir)
	if err != nil {
		return err
	}
	return errors.Join(syncDirFile(f), f.Close())
}
func readConnectorClaim(root *os.Root, name, session string) (dshConnectorClaim, error) {
	var claim dshConnectorClaim
	info, err := root.Lstat(name)
	if err != nil {
		return claim, err
	}
	if !info.Mode().IsRegular() || info.Size() > dshConnectorMaxResult*6+4096 {
		return claim, errors.New("dsh_connector_claim_invalid")
	}
	f, err := root.Open(name)
	if err != nil {
		return claim, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return claim, errors.New("dsh_connector_claim_invalid")
	}
	dec := json.NewDecoder(io.LimitReader(f, dshConnectorMaxResult*6+4097))
	dec.DisallowUnknownFields()
	if dec.Decode(&claim) != nil || dec.Decode(new(any)) != io.EOF || claim.SessionID != session || !validUUID(claim.OperationID) || claim.CallID == "" || len(claim.CallID) > 1024 {
		return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
	}
	if w := claim.WorkspaceBoundary; w != nil {
		hash, hashErr := hex.DecodeString(w.ArgumentsSHA)
		if (w.HistoryOperation != "" && (!validUUID(w.HistoryOperation) || !validUUID(w.GoalID) || (!strings.HasPrefix(w.SDKGoalID, "goal-") || !validUUID(strings.TrimPrefix(w.SDKGoalID, "goal-"))))) || (w.HistoryOperation == "" && (w.GoalID != "" || w.SDKGoalID != "")) || w.Session != claim.SessionID || w.Operation != claim.OperationID || w.Call != claim.CallID || (w.Name != "write" && w.Name != "edit") || w.CallSeq < 0 || w.Turn < 1 || w.Step < 1 || hashErr != nil || len(hash) != sha256.Size || hex.EncodeToString(hash) != w.ArgumentsSHA || claim.Target != nil || claim.Kind != "" && claim.Kind != "workspace" {
			return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
		}
	}
	if claim.Target != nil && (claim.Kind == "workspace" || claim.Kind == "canvas" || claim.Kind == "native" || !validDSHConnectorTarget(*claim.Target)) {
		return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
	}
	if claim.State != "user_reviewed" && claim.Review != nil {
		return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
	}
	if claim.Kind != "" && claim.Kind != "workspace" && claim.Kind != "canvas" && claim.Kind != "native" {
		return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
	}
	// The pre-dispatch canvas digest is the only evidence left when a guarded
	// write never produced a result, so it is bound to canvas claims alone.
	if claim.CanvasBefore != nil {
		before, beforeErr := hex.DecodeString(*claim.CanvasBefore)
		if claim.Kind != "canvas" || beforeErr != nil || len(before) != sha256.Size || hex.EncodeToString(before) != *claim.CanvasBefore {
			return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
		}
	}
	switch claim.State {
	case "dispatch_claimed":
		if claim.Output != nil || claim.SDKSeq != nil || (claim.Kind != "" && claim.Kind != "canvas" && claim.Kind != "native") {
			return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
		}
	case "result_ready":
		if claim.Output == nil || len(*claim.Output) > dshConnectorMaxResult || claim.SDKSeq != nil {
			return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
		}
	case "user_reviewed":
		if (claim.Kind != "workspace" && claim.Kind != "canvas") || claim.Target != nil || claim.SDKSeq != nil || claim.Review == nil || !validDSHUserSideEffectReview(*claim.Review) || claim.Output != nil && len(*claim.Output) > dshConnectorMaxResult {
			return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
		}
	case "reconciled":
		if claim.Output == nil || len(*claim.Output) > dshConnectorMaxResult || claim.SDKSeq == nil || *claim.SDKSeq < 0 {
			return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
		}
	default:
		return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
	}
	return claim, nil
}

// A sentinel without a corresponding record remains uncertain after a crash.
func dshConnectorClaims(root *os.Root, session string) ([]dshConnectorClaim, error) {
	if root == nil || !validUUID(session) {
		return nil, errors.New("dsh_connector_claim_invalid")
	}
	dir := "dsh-connector-dispatch/" + session
	info, err := root.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil || !info.IsDir() {
		return nil, errors.New("dsh_connector_claim_invalid")
	}
	d, err := root.Open(dir)
	if err != nil {
		return nil, errors.New("dsh_connector_claim_invalid")
	}
	defer d.Close()
	opened, err := d.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("dsh_connector_claim_invalid")
	}
	entries, err := d.ReadDir(4098)
	if err != nil && !errors.Is(err, io.EOF) || len(entries) > 4097 {
		return nil, errors.New("dsh_connector_claim_invalid")
	}
	out := make([]dshConnectorClaim, 0, len(entries))
	var pending *dshConnectorClaim
	var matched bool
	for _, entry := range entries {
		name := dir + "/" + entry.Name()
		claim, e := readConnectorClaim(root, name, session)
		if e != nil {
			return nil, errors.New("dsh_connector_claim_invalid")
		}
		if entry.Name() == ".pending" {
			if claim.State != "dispatch_claimed" {
				return nil, errors.New("dsh_connector_claim_invalid")
			}
			p := claim
			pending = &p
			continue
		}
		if connectorClaimPath(session, claim.OperationID, claim.CallID) != name {
			return nil, errors.New("dsh_connector_claim_invalid")
		}
		out = append(out, claim)
	}
	if pending != nil {
		for _, claim := range out {
			if claim.OperationID == pending.OperationID && claim.CallID == pending.CallID {
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, *pending)
		}
	}
	return out, nil
}
func connectorPending(root *os.Root, session, operation, call string) (dshConnectorClaim, error) {
	if root == nil || !validUUID(session) || !validUUID(operation) || call == "" || len(call) > 1024 {
		return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
	}
	pending, err := readConnectorClaim(root, "dsh-connector-dispatch/"+session+"/.pending", session)
	if err != nil || pending.State != "dispatch_claimed" || pending.OperationID != operation || pending.CallID != call {
		return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
	}
	claim, err := readConnectorClaim(root, connectorClaimPath(session, operation, call), session)
	if err != nil || claim.OperationID != operation || claim.CallID != call {
		return dshConnectorClaim{}, errors.New("dsh_connector_claim_invalid")
	}
	return claim, nil
}
func writeConnectorClaim(root *os.Root, claim dshConnectorClaim) error {
	data, err := json.Marshal(claim)
	if err != nil {
		return err
	}
	name := connectorClaimPath(claim.SessionID, claim.OperationID, claim.CallID)
	if err = writeSourceState(root, name, data); err != nil {
		return err
	}
	return syncConnectorDir(root, "dsh-connector-dispatch/"+claim.SessionID)
}

// A complete bounded tool response is stored before returning to the SDK.
func dshConnectorRecordResult(root *os.Root, session, operation, call, output string) error {
	if len(output) > dshConnectorMaxResult || !utf8.ValidString(output) {
		return errors.New("dsh_connector_result_unavailable")
	}
	claim, err := connectorPending(root, session, operation, call)
	if err != nil || claim.State != "dispatch_claimed" {
		return errors.New("dsh_connector_result_unavailable")
	}
	claim.State = "result_ready"
	claim.Output = &output
	if err = writeConnectorClaim(root, claim); err != nil {
		return errors.New("dsh_connector_result_unavailable")
	}
	return nil
}

// readDurable must read an SDK persistence snapshot, not a live notification.
// It must verify that the result follows the unique matching tool/call bound to
// this platform operation; call IDs alone may be reused across turns. The host
// also validates session, call, success and exact text. This callback is a
// trusted host capability and must never come from an untrusted SDK request.
func dshConnectorReconcile(root *os.Root, session, operation, call string, readDurable func(string, string, string) (json.RawMessage, error)) error {
	claim, err := connectorPending(root, session, operation, call)
	if err != nil || (claim.Kind != "" && claim.Kind != "canvas" && claim.Kind != "native") || (claim.State != "result_ready" && claim.State != "reconciled") || readDurable == nil {
		return errors.New("dsh_connector_reconciliation_required")
	}
	raw, err := readDurable(session, operation, call)
	if err != nil {
		return errors.New("dsh_connector_reconciliation_required")
	}
	seq, err := dshVerifiedConnectorResult(claim, raw)
	if err != nil {
		return err
	}
	return dshReleaseSideEffect(root, claim, seq)
}

func dshVerifiedConnectorResult(claim dshConnectorClaim, raw json.RawMessage) (int64, error) {
	var receipt struct {
		SessionID string `json:"sessionId"`
		Event     struct {
			Type string `json:"type"`
			Seq  *int64 `json:"seq"`
			Data struct {
				Message struct {
					Role       string `json:"role"`
					ToolCallID string `json:"toolCallId"`
					IsError    bool   `json:"isError"`
					Source     struct {
						Kind   string `json:"kind"`
						CallID string `json:"callId"`
					} `json:"source"`
					Content []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					} `json:"content"`
				} `json:"message"`
			} `json:"data"`
		} `json:"event"`
	}
	if claim.Output == nil || len(raw) > dshConnectorMaxResult*6+4096 || json.Unmarshal(raw, &receipt) != nil || receipt.SessionID != claim.SessionID || receipt.Event.Type != "tool/result" || receipt.Event.Seq == nil || *receipt.Event.Seq < 0 || receipt.Event.Data.Message.Role != "tool" || receipt.Event.Data.Message.Source.Kind != "tool" || receipt.Event.Data.Message.Source.CallID != claim.CallID || receipt.Event.Data.Message.ToolCallID != claim.CallID || receipt.Event.Data.Message.IsError || len(receipt.Event.Data.Message.Content) != 1 || receipt.Event.Data.Message.Content[0].Type != "text" {
		return 0, errors.New("dsh_connector_reconciliation_required")
	}
	actual := receipt.Event.Data.Message.Content[0].Text
	if claim.Kind == "canvas" {
		actual = dshCanvasResultDigest(actual)
	}
	if actual != *claim.Output {
		return 0, errors.New("dsh_connector_reconciliation_required")
	}
	return *receipt.Event.Seq, nil
}

func dshReleaseSideEffect(root *os.Root, claim dshConnectorClaim, seq int64) error {
	session, operation, call := claim.SessionID, claim.OperationID, claim.CallID
	current, err := connectorPending(root, session, operation, call)
	if err != nil || !reflect.DeepEqual(current, claim) {
		return errors.New("dsh_connector_reconciliation_required")
	}
	if claim.State == "reconciled" {
		if claim.SDKSeq == nil || *claim.SDKSeq != seq {
			return errors.New("dsh_connector_reconciliation_required")
		}
	} else {
		claim.State = "reconciled"
		claim.SDKSeq = &seq
		if err = writeConnectorClaim(root, claim); err != nil {
			return errors.New("dsh_connector_reconciliation_required")
		}
	}
	if err = root.Remove("dsh-connector-dispatch/" + session + "/.pending"); err != nil {
		return errors.New("dsh_connector_reconciliation_required")
	}
	if err = syncConnectorDir(root, "dsh-connector-dispatch/"+session); err != nil {
		return errors.New("dsh_connector_reconciliation_required")
	}
	return nil
}

// dshConnectorSavedResult can resend a persisted result, never the side effect.
// The caller must bind it to the exact durable SDK call before presenting it.
func dshConnectorSavedResult(root *os.Root, session, operation, call string) (string, error) {
	claim, err := connectorPending(root, session, operation, call)
	if err != nil || claim.Kind != "" || claim.Output == nil || (claim.State != "result_ready" && claim.State != "reconciled") {
		return "", errors.New("dsh_connector_result_unavailable")
	}
	return *claim.Output, nil
}

// dshConnectorAssertSettled is an admission check, not a cross-operation lock.
// Callers must hold the same session execution lock until their side effect is
// durably claimed; checking and then dispatching independently is racy.
func dshConnectorAssertSettled(root *os.Root, session string) error {
	claims, err := dshConnectorClaims(root, session)
	if err != nil {
		return errors.New("dsh_connector_reconciliation_required")
	}
	for _, claim := range claims {
		if claim.State != "reconciled" && claim.State != "user_reviewed" {
			return errors.New("dsh_connector_reconciliation_required")
		}
	}
	_, err = root.Lstat("dsh-connector-dispatch/" + session + "/.pending")
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return errors.New("dsh_connector_reconciliation_required")
	}
	return nil
}
