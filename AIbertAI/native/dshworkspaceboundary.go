package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
)

const maxDSHWorkspaceText = 4 << 20

type dshBoundaryStateKey struct{}
type dshBoundaryState struct {
	capable bool
	history dshConnectorHistoryFetch
	proof   dshConnectorHistoryFetch
}

func dshVerifyWorkspaceBoundary(w dshWorkspaceWitness, arguments json.RawMessage, digest string, response json.RawMessage) (int64, error) {
	fail := errors.New("unverified")
	canonical, err := strictDSHJSON(arguments)
	if err != nil {
		return 0, fail
	}
	sum := sha256.Sum256(canonical)
	if !validUUID(w.Session) || !validUUID(w.Operation) || w.Call == "" || (w.Name != "write" && w.Name != "edit") || w.CallSeq < 0 || w.Turn < 1 || w.Step < 1 || fmt.Sprintf("%x", sum) != w.ArgumentsSHA {
		return 0, fail
	}
	if _, err = strictDSHJSON(response); err != nil {
		return 0, fail
	}
	var envelope struct {
		SessionID, OperationID, CallID string
		CallSeq, ProofSeq              *int64
		Events                         []dshConnectorHistoryEvent
	}
	if json.Unmarshal(response, &envelope) != nil || envelope.SessionID != w.Session || envelope.OperationID != w.Operation || envelope.CallID != w.Call || envelope.CallSeq == nil || *envelope.CallSeq != w.CallSeq || envelope.ProofSeq == nil || len(envelope.Events) != 3 {
		return 0, fail
	}
	call, result, proof := envelope.Events[0], envelope.Events[1], envelope.Events[2]
	if call.Seq == nil || *call.Seq != w.CallSeq || call.Type != "tool/call" || result.Seq == nil || *result.Seq <= w.CallSeq || result.Type != "tool/result" || proof.Seq == nil || *proof.Seq != *envelope.ProofSeq || *proof.Seq <= *result.Seq || proof.Type != "aibertai/workspace-result" || len(result.SourceEventSeqs) != 1 || result.SourceEventSeqs[0] != w.CallSeq {
		return 0, fail
	}
	var c struct {
		Turn, Step              *int64
		CallID, Name, Arguments string
	}
	if json.Unmarshal(call.Data, &c) != nil || c.Turn == nil || *c.Turn != w.Turn || c.Step == nil || *c.Step != w.Step || c.CallID != w.Call || c.Name != w.Name {
		return 0, fail
	}
	actual, err := strictDSHJSON([]byte(c.Arguments))
	if err != nil || !bytes.Equal(actual, canonical) {
		return 0, fail
	}
	var r struct {
		Turn, Step *int64
		Message    struct {
			Role, ToolCallID string
			IsError          bool
			Source           struct{ Kind, CallID string }
		}
	}
	if json.Unmarshal(result.Data, &r) != nil || r.Turn == nil || *r.Turn != w.Turn || r.Step == nil || *r.Step != w.Step || r.Message.Role != "tool" || r.Message.ToolCallID != w.Call || r.Message.IsError || r.Message.Source.Kind != "tool" || r.Message.Source.CallID != w.Call {
		return 0, fail
	}
	var p struct {
		OperationID, CallID, ToolName, Digest string
		ToolCallSeq, ToolResultSeq            *int64
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(proof.Data, &fields) != nil || len(fields) != 6 || json.Unmarshal(proof.Data, &p) != nil || p.OperationID != w.Operation || p.CallID != w.Call || p.ToolName != w.Name || p.Digest != digest || p.ToolCallSeq == nil || *p.ToolCallSeq != w.CallSeq || p.ToolResultSeq == nil || *p.ToolResultSeq != *result.Seq {
		return 0, fail
	}
	return *result.Seq, nil
}

func dshAuditWorkspacePrefix(ctx context.Context, session, operation, call, name string, args json.RawMessage, fetch dshConnectorHistoryFetch) (dshWorkspaceWitness, error) {
	return dshAuditWorkspacePrefixForGoal(ctx, session, operation, call, name, args, fetch, "")
}

func dshAuditWorkspacePrefixForGoal(ctx context.Context, session, operation, call, name string, args json.RawMessage, fetch dshConnectorHistoryFetch, goalID string) (dshWorkspaceWitness, error) {
	var out dshWorkspaceWitness
	fail := errors.New("unverified prefix")
	canonical, err := strictDSHJSON(args)
	if err != nil {
		return out, fail
	}
	var callSeq, turn, step int64 = -1, 0, 0
	var calls, declared, goalBindings int
	var assistantSeq int64 = -1
	var assistantTurn, assistantStep int64
	ceiling, err := dshRepairHistory(ctx, session, -1, fetch, func(seq int64, raw json.RawMessage) error {
		var e struct {
			Type string
			Data struct {
				Turn, Step              int64
				CallID, Name, Arguments string
				Message                 struct {
					Role    string
					Source  struct{ Kind string }
					Content []struct{ Type, ID, Name, Arguments string }
				}
			}
		}
		if json.Unmarshal(raw, &e) != nil {
			return fail
		}
		if e.Type == "aibertai/query-binding" && goalID != "" {
			var b struct{ OperationID, GoalID string }
			if json.Unmarshal(raw, &struct {
				Data *struct{ OperationID, GoalID string } `json:"data"`
			}{Data: &b}) != nil {
				return fail
			}
			if b.OperationID == operation {
				if b.GoalID != goalID {
					return fail
				}
				goalBindings++
			}
		}
		if e.Type == "tool/call" && e.Data.CallID == call {
			calls++
			callSeq, turn, step = seq, e.Data.Turn, e.Data.Step
			got, err := strictDSHJSON([]byte(e.Data.Arguments))
			if err != nil || e.Data.Name != name || !bytes.Equal(got, canonical) {
				return fail
			}
		}
		if e.Type == "assistant/message" {
			for _, block := range e.Data.Message.Content {
				if block.Type == "tool-call" && block.ID == call {
					declared++
					assistantSeq = seq
					assistantTurn, assistantStep = e.Data.Turn, e.Data.Step
					got, err := strictDSHJSON([]byte(block.Arguments))
					if err != nil || block.Name != name || e.Data.Message.Role != "assistant" || e.Data.Message.Source.Kind != "model" || !bytes.Equal(got, canonical) {
						return fail
					}
				}
			}
		}
		return nil
	})
	if err != nil || (goalID != "" && goalBindings != 1) || calls != 1 || declared != 1 || callSeq <= assistantSeq || assistantSeq < 0 || turn < 1 || step < 1 || assistantTurn != turn || assistantStep != step {
		return out, fail
	}
	// The second pass fixes its SDK ceiling at the first pass's durable prefix.
	virtual := func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
		after, ok := params["afterSeq"].(int64)
		if !ok || after > ceiling {
			return nil, fail
		}
		forward := map[string]any{"sessionId": session, "afterSeq": after, "throughSeq": ceiling}
		// Oversized events (>512 KiB) arrive as fragments. Pass them through
		// against the real ceiling and report the virtual one the caller checks.
		if seq, fragment := params["fragmentSeq"]; fragment {
			if after >= ceiling {
				return nil, fail
			}
			forward["fragmentSeq"], forward["offset"] = seq, params["offset"]
			raw, err := fetch(ctx, forward)
			if err != nil {
				return nil, err
			}
			var part map[string]json.RawMessage
			var through int64
			if json.Unmarshal(raw, &part) != nil || json.Unmarshal(part["throughSeq"], &through) != nil || through != ceiling {
				return nil, fail
			}
			part["throughSeq"], _ = json.Marshal(ceiling + 1)
			return json.Marshal(part)
		}
		raw, err := fetch(ctx, forward)
		if err != nil {
			return nil, err
		}
		var page struct {
			SessionID string            `json:"sessionId"`
			Through   int64             `json:"throughSeq"`
			Events    []json.RawMessage `json:"events"`
			Large     json.RawMessage   `json:"largeEvent,omitempty"`
		}
		if json.Unmarshal(raw, &page) != nil || page.SessionID != session || page.Through != ceiling {
			return nil, fail
		}
		if after == ceiling {
			fake := map[string]any{"seq": ceiling + 1, "type": "tool/result", "sourceEventSeqs": []int64{callSeq}, "data": map[string]any{"turn": turn, "step": step, "message": map[string]any{"role": "tool", "toolCallId": call, "source": map[string]any{"kind": "tool", "callId": call}, "isError": false}}}
			b, _ := json.Marshal(fake)
			page.Events = []json.RawMessage{b}
		}
		page.Through = ceiling + 1
		return json.Marshal(page)
	}
	if _, err = dshMatchConnectorHistory(ctx, session, operation, call, name, args, virtual); err != nil {
		return out, err
	}
	sum := sha256.Sum256(canonical)
	return dshWorkspaceWitness{Session: session, Operation: operation, Call: call, Name: name, ArgumentsSHA: fmt.Sprintf("%x", sum), CallSeq: callSeq, Turn: turn, Step: step}, nil
}

func dshVerifyWorkspaceBoundaryPages(w dshWorkspaceWitness, args json.RawMessage, digest string, pages []json.RawMessage) (int64, error) {
	fail := errors.New("unverified pages")
	if len(pages) != 3 {
		return 0, fail
	}
	var events []json.RawMessage
	var proofSeq int64 = -1
	for _, raw := range pages {
		if len(raw) > 9<<20 {
			return 0, fail
		}
		if _, err := strictDSHJSON(raw); err != nil {
			return 0, fail
		}
		var page struct {
			SessionID, OperationID, CallID string
			CallSeq, ProofSeq              *int64
			Events                         []json.RawMessage
		}
		if json.Unmarshal(raw, &page) != nil || page.SessionID != w.Session || page.OperationID != w.Operation || page.CallID != w.Call || page.CallSeq == nil || *page.CallSeq != w.CallSeq || page.ProofSeq == nil || len(page.Events) != 1 {
			return 0, fail
		}
		if proofSeq < 0 {
			proofSeq = *page.ProofSeq
		} else if proofSeq != *page.ProofSeq {
			return 0, fail
		}
		events = append(events, page.Events[0])
	}
	envelope, err := json.Marshal(map[string]any{"sessionId": w.Session, "operationId": w.Operation, "callId": w.Call, "callSeq": w.CallSeq, "proofSeq": proofSeq, "events": events})
	if err != nil {
		return 0, err
	}
	return dshVerifyWorkspaceBoundary(w, args, digest, envelope)
}

func workspaceBoundaryCapable(raw json.RawMessage) bool {
	var init struct {
		Capability *struct {
			Version, MaxEventBytes, MaxTextBytes, MaxHistoryBytes int
			Paged                                                 bool
		} `json:"aibertaiWorkspaceBoundary"`
	}
	return json.Unmarshal(raw, &init) == nil && init.Capability != nil && init.Capability.Version == 1 && init.Capability.MaxEventBytes >= 8<<20 && init.Capability.MaxTextBytes >= maxDSHWorkspaceText && init.Capability.MaxHistoryBytes == 64<<20 && init.Capability.Paged
}

func dshVerifyBoundaryClaim(ctx context.Context, claim dshConnectorClaim, name string, args json.RawMessage) (int64, error) {
	fail := errors.New("dsh_workspace_reconciliation_required")
	state, _ := ctx.Value(dshBoundaryStateKey{}).(*dshBoundaryState)
	w := claim.WorkspaceBoundary
	if state == nil || !state.capable || state.proof == nil || w == nil || claim.Output == nil || name != w.Name || w.Session != claim.SessionID || w.Operation != claim.OperationID || w.Call != claim.CallID {
		return 0, fail
	}
	pages := make([]json.RawMessage, 0, 3)
	for _, part := range []string{"call", "result", "proof"} {
		page, err := state.proof(ctx, map[string]any{"sessionId": w.Session, "operationId": w.Operation, "callId": w.Call, "callSeq": w.CallSeq, "part": part})
		if err != nil {
			return 0, errors.Join(fail, err)
		}
		pages = append(pages, page)
	}
	seq, err := dshVerifyWorkspaceBoundaryPages(*w, args, *claim.Output, pages)
	if err != nil {
		return 0, fail
	}
	return seq, nil
}
