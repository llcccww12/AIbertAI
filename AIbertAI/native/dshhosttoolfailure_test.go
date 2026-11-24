package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// DSH reports a failed tool as an isError result and the loop continues. A tool
// outcome must never end the run: the model gets a reason and picks the next step.
func TestDSHHostToolFailureDoesNotEndTheRun(t *testing.T) {
	frames := make(chan dshFrame, 4)
	frames <- dshFrame{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "aibertai/host-tool", Params: json.RawMessage(`{"name":"first"}`)}
	frames <- dshFrame{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "aibertai/host-tool", Params: json.RawMessage(`{"name":"second"}`)}
	frames <- dshFrame{JSONRPC: "2.0", ID: json.RawMessage(`1`), Result: json.RawMessage(`{"done":true}`)}
	in := &dshTestWriteCloser{}
	p := &dshProcess{in: in, frames: frames}
	calls := 0
	p.hostTool = func(json.RawMessage) (any, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("dsh_connector_outcome_unknown")
		}
		return map[string]string{"output": "switched to another tool"}, nil
	}
	result, err := p.call(t.Context(), "session/wait", nil, nil)
	if err != nil || string(result) != `{"done":true}` || calls != 2 {
		t.Fatalf("a tool outcome ended the run: result=%s calls=%d err=%v", result, calls, err)
	}
	out := in.String()
	if !strings.Contains(out, "connector_outcome_unknown") || !strings.Contains(out, "switched to another tool") {
		t.Fatalf("model-facing responses: %s", out)
	}
}

func TestDSHWorkspaceMutationFailureDoesNotEndTheRun(t *testing.T) {
	frames := make(chan dshFrame, 4)
	frames <- dshFrame{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "aibertai/workspace-mutate", Params: json.RawMessage(`{"name":"write"}`)}
	frames <- dshFrame{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: "aibertai/workspace-mutate", Params: json.RawMessage(`{"name":"edit"}`)}
	frames <- dshFrame{JSONRPC: "2.0", ID: json.RawMessage(`1`), Result: json.RawMessage(`{"done":true}`)}
	in := &dshTestWriteCloser{}
	p := &dshProcess{in: in, frames: frames}
	calls := 0
	p.hostRequest = func(json.RawMessage) (any, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("dsh_connector_outcome_unknown")
		}
		return map[string]any{"output": "edited"}, nil
	}
	result, err := p.call(t.Context(), "session/wait", nil, nil)
	if err != nil || string(result) != `{"done":true}` || calls != 2 {
		t.Fatalf("a failed file change ended the run: result=%s calls=%d err=%v", result, calls, err)
	}
	out := in.String()
	if !strings.Contains(out, "write_outcome_unknown") || !strings.Contains(out, "edited") {
		t.Fatalf("model-facing responses: %s", out)
	}
}
