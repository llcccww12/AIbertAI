package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The test binary acts as the fixed line-oriented SDK process.
func TestDSHMockProcess(t *testing.T) {
	if os.Getenv("DSH_MOCK") != "1" {
		return
	}
	scan := bufio.NewScanner(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	for scan.Scan() {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
			Params struct {
				SessionID string `json:"sessionId"`
			} `json:"params"`
		}
		if json.Unmarshal(scan.Bytes(), &req) != nil {
			os.Exit(2)
		}
		if req.Method == "initialize" && os.Getenv("DSH_MOCK_BAD_FRAME") != "" {
			if os.Getenv("DSH_MOCK_BAD_FRAME") == "large" {
				fmt.Fprintln(os.Stdout, strings.Repeat("x", (8<<20)+1))
			} else {
				fmt.Fprintln(os.Stdout, "{invalid")
			}
			continue
		}
		if req.Method == "session/prompt" && os.Getenv("DSH_MOCK_FOREIGN_RECEIPT") == "1" {
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "aibertai/model-settled", "params": map[string]any{"sessionId": "foreign", "invocationId": "22222222-2222-4222-8222-222222222222", "state": "committed", "seq": 1}})
		}
		if req.Method == "session/prompt" && req.Params.SessionID == "crash" {
			os.Exit(3)
		}
		if req.Method == "session/prompt" && req.Params.SessionID == "approval" {
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": "approval-1", "method": "approval/request", "params": map[string]string{"sessionId": "approval", "toolName": "write"}})
			if !scan.Scan() {
				os.Exit(4)
			}
			var reply struct {
				Result struct {
					Outcome string `json:"outcome"`
				} `json:"result"`
			}
			if json.Unmarshal(scan.Bytes(), &reply) != nil || reply.Result.Outcome != "rejected" {
				os.Exit(5)
			}
		}
		if req.Method == "session/prompt" && os.Getenv("DSH_MOCK_DISCONNECT") != "" {
			f, err := os.OpenFile(os.Getenv("DSH_MOCK_DISCONNECT"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				os.Exit(6)
			}
			_, _ = fmt.Fprintln(f, "prompt")
			_ = f.Close()
		}
		if req.Method == "session/query" && os.Getenv("DSH_MOCK_QUERY") == "1" {
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "session.event", "params": map[string]any{"sessionId": req.Params.SessionID, "event": map[string]any{"type": "goal/change", "seq": 7, "data": map[string]any{"operation": "create"}}}})
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "session.event", "params": map[string]any{"sessionId": req.Params.SessionID, "event": map[string]any{"type": "future/unknown", "seq": 8, "data": map[string]any{"value": "preserved"}}}})
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "session.query.completed", "params": map[string]any{"sessionId": req.Params.SessionID, "commandId": "command-1", "finalResponse": "goal done", "finishReason": "completed"}})
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{"kind": "command", "execution": map[string]any{"commandId": "command-1"}}})
			continue
		}
		if req.Method == "session/wait" {
			if os.Getenv("DSH_MOCK_REASONING") == "1" {
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "aibertai/assistant-stream", "params": map[string]any{"sessionId": req.Params.SessionID, "frame": map[string]any{"type": "start", "attemptId": "a", "revision": 1}}})
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "aibertai/assistant-stream", "params": map[string]any{"sessionId": req.Params.SessionID, "frame": map[string]any{"type": "chunk", "attemptId": "a", "revision": 2, "index": 0, "chunk": map[string]any{"type": "reasoning-delta", "index": 0, "text": "thinking"}}}})
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "aibertai/assistant-stream", "params": map[string]any{"sessionId": req.Params.SessionID, "frame": map[string]any{"type": "chunk", "attemptId": "a", "revision": 3, "index": 1, "chunk": map[string]any{"type": "text-delta", "index": 1, "text": "answer"}}}})
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "session.event", "params": map[string]any{"sessionId": req.Params.SessionID, "event": map[string]any{"type": "assistant/message", "data": map[string]any{"message": map[string]any{"content": []map[string]string{{"type": "reasoning", "text": "final thought"}, {"type": "text", "text": "answer"}}}}}}})
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "session.event", "params": map[string]any{"sessionId": req.Params.SessionID, "event": map[string]any{"type": "turn/end", "data": map[string]any{"reason": map[string]string{"kind": "completed"}}}}})
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{}})
				continue
			}
			if os.Getenv("DSH_MOCK_MISMATCH") == "1" {
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "aibertai/assistant-stream", "params": map[string]any{"sessionId": req.Params.SessionID, "frame": map[string]any{"type": "start", "attemptId": "a", "revision": 1}}})
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "aibertai/assistant-stream", "params": map[string]any{"sessionId": req.Params.SessionID, "frame": map[string]any{"type": "chunk", "attemptId": "a", "revision": 2, "index": 1, "chunk": map[string]any{"type": "text-delta", "index": 0, "text": "partial"}}}})
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "session.event", "params": map[string]any{"sessionId": req.Params.SessionID, "event": map[string]any{"type": "assistant/message", "data": map[string]any{"message": map[string]any{"content": []map[string]string{{"type": "text", "text": "final"}}}}}}})
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "session.event", "params": map[string]any{"sessionId": req.Params.SessionID, "event": map[string]any{"type": "turn/end", "data": map[string]any{"reason": map[string]string{"kind": "completed"}}}}})
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": map[string]any{}})
				continue
			}
			if os.Getenv("DSH_MOCK_DISCONNECT") != "" {
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "session.event", "params": map[string]any{"sessionId": req.Params.SessionID, "event": map[string]any{"type": "tool/call", "data": map[string]any{"callId": "call-1", "name": "read", "arguments": "{}"}}}})
				time.Sleep(700 * time.Millisecond)
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "session.event", "params": map[string]any{"sessionId": req.Params.SessionID, "event": map[string]any{"type": "tool/result", "data": map[string]any{"message": map[string]any{"source": map[string]string{"callId": "call-1"}, "content": []any{map[string]string{"type": "text", "text": "read output"}}}}}}})
			}
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "session.event", "params": map[string]any{"sessionId": "other", "event": map[string]any{"type": "assistant/message", "data": map[string]any{"message": map[string]any{"content": []map[string]string{{"type": "text", "text": "wrong"}}}}}}})
			for i, txt := range []string{"first", "final"} {
				_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "session.event", "params": map[string]any{"sessionId": req.Params.SessionID, "event": map[string]any{"type": "assistant/message", "seq": i + 10, "data": map[string]any{"message": map[string]any{"content": []map[string]string{{"type": "text", "text": txt}}}, "usage": map[string]int{"inputTokens": 1, "outputTokens": 2}}}}})
			}
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "session.event", "params": map[string]any{"sessionId": req.Params.SessionID, "event": map[string]any{"type": "turn/end", "data": map[string]any{"reason": map[string]string{"kind": "completed"}}}}})
		}
		result := map[string]any{}
		if req.Method == "session/prompt" {
			result["messageId"] = "queued"
		}
		if req.Method == "session/resume" && req.Params.SessionID == "missing" {
			_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -1, "message": "missing"}})
			continue
		}
		_ = enc.Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}
	os.Exit(0)
}

func TestDSHReasoningStreamAndHistory(t *testing.T) {
	cfg := mockDSH()
	base := cfg.Command
	cfg.Command = func(ctx context.Context, s acpSession) (*exec.Cmd, error) {
		cmd, err := base(ctx, s)
		cmd.Env = append(cmd.Env, "DSH_MOCK_REASONING=1")
		return cmd, err
	}
	turn := acpTurn{Prompt: "hello"}
	var updates []map[string]any
	err := runDSHTurn(t.Context(), cfg, acpSession{DSHSessionID: "session"}, &turn, false, func(update map[string]any) error {
		if update != nil {
			updates = append(updates, update)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 4 || updates[1]["sessionUpdate"] != "agent_thought_chunk" || updates[2]["sessionUpdate"] != "agent_message_chunk" {
		t.Fatalf("unexpected stream: %#v", updates)
	}
	if len(turn.DSHParts) != 2 || turn.DSHParts[0] != (dshMessagePart{Type: "thought", Text: "final thought"}) || turn.Answer != "answer" {
		t.Fatalf("settled parts lost: %#v", turn.DSHParts)
	}
	row := acpSession{ID: "session", Turns: []acpTurn{turn}}
	message := row.responseMessage(0, map[string]any{})
	parts := message["parts"].([]map[string]string)
	if len(parts) != 2 || parts[0]["type"] != "thought" || parts[0]["text"] != "final thought" || parts[1]["text"] != "answer" {
		t.Fatalf("history parts lost: %#v", parts)
	}
}
func TestDSHDurableMessageWithoutDeltas(t *testing.T) {
	turn := acpTurn{Prompt: "hello"}
	var updates []map[string]any
	err := runDSHTurn(t.Context(), mockDSH(), acpSession{DSHSessionID: "session"}, &turn, false, func(update map[string]any) error {
		updates = append(updates, update)
		return nil
	})
	if err != nil || len(updates) != 2 {
		t.Fatalf("durable messages: %v %v", updates, err)
	}
	for i, update := range updates {
		meta := update["_meta"].(map[string]any)
		if update["sessionUpdate"] != "agent_message_settled" || update["streamAttemptId"] != "" || meta["dshMessageSeq"] != int64(i+10) {
			t.Fatalf("missing SDK message identity: %v", update)
		}
	}
}

func mockDSH() *dshRuntimeConfig {
	return &dshRuntimeConfig{Command: func(ctx context.Context, _ acpSession) (*exec.Cmd, error) {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDSHMockProcess$")
		cmd.Env = append(os.Environ(), "DSH_MOCK=1")
		return cmd, nil
	}, Provider: func(context.Context, acpSession) (string, string, error) { return "mock", "model", nil }}
}
func TestDSHRuntimeLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, id := range []string{"one", "two", "approval"} {
		turn := acpTurn{Prompt: "hello"}
		if err := runDSHTurn(ctx, mockDSH(), acpSession{DSHSessionID: id}, &turn, id == "two", nil); err != nil {
			t.Fatal(err)
		}
		if id == "approval" && (turn.Permissions.Asked != 1 || turn.Permissions.Denied != 1) {
			t.Fatalf("approval counts: %+v", turn.Permissions)
		}
		if turn.Answer != "final" || *turn.Usage.Input != 2 || *turn.Usage.Output != 4 {
			t.Fatalf("unexpected turn: %+v", turn)
		}
	}
	for _, id := range []string{"crash", "missing"} {
		turn := acpTurn{Prompt: "hello"}
		err := runDSHTurn(ctx, mockDSH(), acpSession{DSHSessionID: id}, &turn, id == "missing", nil)
		if err == nil {
			t.Fatal(fmt.Sprintf("%s unexpectedly succeeded", id))
		}
		if id == "missing" && !strings.Contains(err.Error(), "missing") {
			t.Fatal(err)
		}
	}
	turn := acpTurn{Prompt: "again"}
	if err := runDSHTurn(ctx, mockDSH(), acpSession{DSHSessionID: "one"}, &turn, true, nil); err != nil {
		t.Fatal(err)
	}
}

func TestDSHRejectsForeignSettlement(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	local, err := newDSHLocalGateway(nil, id, selectedModelConfig{ID: "22222222-2222-4222-8222-222222222222", Protocol: "openai_chat_completions"}, filepath.Join(t.TempDir(), "journal"))
	if err != nil {
		t.Fatal(err)
	}
	local.requireInvocationID = true
	cfg := &dshRuntimeConfig{Prepare: func(ctx context.Context, s acpSession) (*exec.Cmd, string, string, *dshLocalGateway, func(), error) {
		cmd, err := mockDSH().Command(ctx, s)
		cmd.Env = append(cmd.Env, "DSH_MOCK_FOREIGN_RECEIPT=1")
		return cmd, "mock", "model", local, nil, err
	}}
	turn := acpTurn{Prompt: "hello"}
	err = runDSHTurn(t.Context(), cfg, acpSession{ID: id, DSHSessionID: id}, &turn, false, nil)
	if err == nil || !strings.Contains(err.Error(), "dsh_invalid_settlement") {
		t.Fatalf("foreign settlement accepted: %v", err)
	}
}

func TestDSHInvalidProtocolFramesFailClosed(t *testing.T) {
	for _, mode := range []string{"invalid", "large"} {
		t.Run(mode, func(t *testing.T) {
			cfg := mockDSH()
			base := cfg.Command
			cfg.Command = func(ctx context.Context, s acpSession) (*exec.Cmd, error) {
				cmd, err := base(ctx, s)
				cmd.Env = append(cmd.Env, "DSH_MOCK_BAD_FRAME="+mode)
				return cmd, err
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			turn := acpTurn{Prompt: "hello"}
			err := runDSHTurn(ctx, cfg, acpSession{DSHSessionID: "session"}, &turn, false, nil)
			if err == nil || mode == "invalid" && !strings.Contains(err.Error(), "dsh_invalid_protocol_frame") || mode == "large" && !strings.Contains(err.Error(), "dsh_protocol_read_failed") {
				t.Fatalf("unexpected protocol outcome: %v", err)
			}
		})
	}
}

func TestDSHInvalidFrameKeepsGatewayPending(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	const call = "22222222-2222-4222-8222-222222222222"
	local, err := newDSHLocalGateway(nil, id, selectedModelConfig{ID: call, Protocol: "openai_chat_completions"}, filepath.Join(t.TempDir(), "journal"))
	if err != nil {
		t.Fatal(err)
	}
	local.requireInvocationID = true
	local.pending = &dshLocalRecord{SessionID: id, CallID: call, ModelID: call, State: "delivered"}
	if err = local.writePending(local.pending); err != nil {
		t.Fatal(err)
	}
	cfg := &dshRuntimeConfig{Prepare: func(ctx context.Context, s acpSession) (*exec.Cmd, string, string, *dshLocalGateway, func(), error) {
		cmd, err := mockDSH().Command(ctx, s)
		cmd.Env = append(cmd.Env, "DSH_MOCK_BAD_FRAME=invalid")
		return cmd, "mock", "model", local, nil, err
	}}
	turn := acpTurn{Prompt: "hello"}
	if err = runDSHTurn(t.Context(), cfg, acpSession{ID: id, DSHSessionID: id}, &turn, true, nil); err == nil || !strings.Contains(err.Error(), "dsh_invalid_protocol_frame") {
		t.Fatalf("invalid frame: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(local.journal, "pending.json"))
	if err != nil {
		t.Fatal(err)
	}
	var record dshLocalRecord
	if json.Unmarshal(data, &record) != nil || record.CallID != call || record.State != "delivered" {
		t.Fatalf("pending changed: %s", data)
	}
}

func TestDSHQueryObservesOrderedRawEvents(t *testing.T) {
	cfg := mockDSH()
	base := cfg.Command
	cfg.Command = func(ctx context.Context, s acpSession) (*exec.Cmd, error) {
		cmd, err := base(ctx, s)
		cmd.Env = append(cmd.Env, "DSH_MOCK_QUERY=1")
		return cmd, err
	}
	var notices []string
	opts := dshSDKOptions{Query: true, Observe: func(method string, raw json.RawMessage) error {
		notices = append(notices, method+":"+string(raw))
		return nil
	}}
	ctx, cancel := context.WithTimeout(context.WithValue(t.Context(), dshSDKOptionsKey{}, opts), 5*time.Second)
	defer cancel()
	turn := acpTurn{Prompt: "/goal status"}
	if err := runDSHTurn(ctx, cfg, acpSession{DSHSessionID: "session"}, &turn, false, nil); err != nil {
		t.Fatal(err)
	}
	if turn.Answer != "goal done" || len(notices) != 3 || !strings.Contains(notices[0], "goal/change") || !strings.Contains(notices[1], "future/unknown") || !strings.HasPrefix(notices[2], "session.query.completed:") {
		t.Fatalf("events=%v turn=%+v", notices, turn)
	}
}

type dshTestWriteCloser struct{ bytes.Buffer }

func (w *dshTestWriteCloser) Close() error { return nil }

func TestDSHHistoryBarrierPreservesFrames(t *testing.T) {
	frames := make(chan dshFrame, 8)
	frames <- dshFrame{JSONRPC: "2.0", Method: "session.event", Params: json.RawMessage(`{"event":"first"}`)}
	frames <- dshFrame{JSONRPC: "2.0", Method: "session.event", Params: json.RawMessage(`{"event":"second"}`)}
	frames <- dshFrame{JSONRPC: "2.0", ID: json.RawMessage(`"tool-1"`), Method: "aibertai/host-tool", Params: json.RawMessage(`{}`)}
	frames <- dshFrame{JSONRPC: "2.0", ID: json.RawMessage(`2`), Result: json.RawMessage(`{"history":true}`)}
	frames <- dshFrame{JSONRPC: "2.0", ID: json.RawMessage(`1`), Result: json.RawMessage(`{"done":true}`)}
	var order []string
	p := &dshProcess{in: &dshTestWriteCloser{}, frames: frames}
	p.hostTool = func(json.RawMessage) (any, error) {
		order = append(order, "tool")
		return map[string]string{"output": "ok"}, nil
	}
	p.afterEvent = func(raw json.RawMessage) error {
		if strings.Contains(string(raw), "first") {
			result, err := p.historyCall(t.Context(), map[string]any{"sessionId": "session", "afterSeq": -1})
			if err != nil || string(result) != `{"history":true}` {
				return errors.New("history failed")
			}
			order = append(order, "barrier")
		}
		return nil
	}
	result, err := p.call(t.Context(), "session/wait", nil, func(raw json.RawMessage) error {
		if strings.Contains(string(raw), "first") {
			order = append(order, "first")
		} else {
			order = append(order, "second")
		}
		return nil
	})
	if err != nil || string(result) != `{"done":true}` || strings.Join(order, ",") != "first,barrier,second,tool" || len(p.deferred) != 0 {
		t.Fatalf("result=%s order=%v buffered=%d err=%v", result, order, len(p.deferred), err)
	}
}

func TestDSHHistoryBarrierBufferLimit(t *testing.T) {
	frames := make(chan dshFrame, 4097)
	for i := 0; i < 4097; i++ {
		frames <- dshFrame{JSONRPC: "2.0", Method: "session.event", Params: json.RawMessage(`{}`)}
	}
	p := &dshProcess{in: &dshTestWriteCloser{}, frames: frames}
	_, err := p.historyCall(t.Context(), map[string]any{})
	if err == nil || err.Error() != "dsh_history_buffer_exceeded" || len(p.deferred) != 4096 {
		t.Fatalf("buffer overflow did not fail closed: %v, buffered=%d", err, len(p.deferred))
	}
}

func TestDSHHistoryBarrierByteLimit(t *testing.T) {
	frames := make(chan dshFrame, 2)
	for i := 0; i < 2; i++ {
		frames <- dshFrame{JSONRPC: "2.0", Method: "session.event", Params: make(json.RawMessage, 5<<20)}
	}
	p := &dshProcess{in: &dshTestWriteCloser{}, frames: frames}
	_, err := p.historyCall(t.Context(), map[string]any{})
	if err == nil || err.Error() != "dsh_history_buffer_exceeded" || len(p.deferred) != 1 {
		t.Fatalf("byte overflow did not fail closed: %v, buffered=%d", err, len(p.deferred))
	}
}

func TestDSHSettlementObserverFailureDoesNotAck(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	const call = "22222222-2222-4222-8222-222222222222"
	local, err := newDSHLocalGateway(nil, id, selectedModelConfig{ID: call, Protocol: "openai_chat_completions"}, filepath.Join(t.TempDir(), "journal"))
	if err != nil {
		t.Fatal(err)
	}
	local.requireInvocationID = true
	local.pending = &dshLocalRecord{SessionID: id, CallID: call, ModelID: call, State: "delivered"}
	if err = local.writePending(local.pending); err != nil {
		t.Fatal(err)
	}
	incoming := make(chan dshFrame, 1)
	raw, _ := json.Marshal(map[string]any{"sessionId": id, "invocationId": call, "state": "committed", "seq": 3})
	incoming <- dshFrame{JSONRPC: "2.0", Method: "aibertai/model-settled", Params: raw}
	p := &dshProcess{in: &dshTestWriteCloser{}, frames: incoming, activity: func(_ string, _ json.RawMessage) error { return errors.New("usage_persist_failed") }, notification: func(_ string, _ json.RawMessage) error { return local.acceptSDKSettlement(id, call, "committed", 3) }}
	if _, err = p.call(t.Context(), "session/wait", nil, nil); err == nil || !strings.Contains(err.Error(), "usage_persist_failed") {
		t.Fatalf("observer error: %v", err)
	}
	if _, err = os.Stat(filepath.Join(local.journal, "pending.json")); err != nil || local.pending == nil {
		t.Fatal("settlement acknowledged before usage persisted")
	}
}

func TestDSHStreamMismatchSettlesByReplacement(t *testing.T) {
	cfg := mockDSH()
	base := cfg.Command
	cfg.Command = func(ctx context.Context, s acpSession) (*exec.Cmd, error) {
		cmd, err := base(ctx, s)
		cmd.Env = append(cmd.Env, "DSH_MOCK_MISMATCH=1")
		return cmd, err
	}
	var updates []map[string]any
	turn := acpTurn{Prompt: "hello"}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := runDSHTurn(ctx, cfg, acpSession{DSHSessionID: "session"}, &turn, false, func(update map[string]any) error {
		if update != nil {
			updates = append(updates, update)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if turn.Answer != "final" || !turn.DSHStreamed || len(updates) != 3 || updates[0]["sessionUpdate"] != "agent_message_start" || updates[1]["sessionUpdate"] != "agent_message_chunk" || updates[2]["sessionUpdate"] != "agent_message_settled" {
		t.Fatalf("turn=%+v updates=%v", turn, updates)
	}
	if updates[2]["content"].(map[string]string)["text"] != "final" {
		t.Fatal("settlement did not replace partial")
	}
}

func TestDSHRenderedPartsPreserveTextOrderWithoutDuplication(t *testing.T) {
	text := "前面后面"
	parts := []dshMessagePart{{Type: "text", End: len("前面")}, {Type: "thought", Text: "考虑"}, {Type: "text", End: len(text)}}
	rendered := dshRenderedParts(parts, text)
	if len(rendered) != 3 || rendered[0]["text"] != "前面" || rendered[1]["text"] != "考虑" || rendered[2]["text"] != "后面" {
		t.Fatalf("invalid parts: %#v", rendered)
	}
	encoded, _ := json.Marshal(parts)
	if strings.Contains(string(encoded), "前面") || strings.Contains(string(encoded), "后面") {
		t.Fatal("snapshot duplicated answer text")
	}
	invalid := dshRenderedParts([]dshMessagePart{{Type: "text", End: len(text) + 1}}, text)
	if len(invalid) != 1 || invalid[0]["text"] != text {
		t.Fatalf("invalid offsets lost answer: %#v", invalid)
	}
}

func TestDSHReasoningHistorySurroundsTool(t *testing.T) {
	turn := acpTurn{
		Operation: "op", Answer: "完成", DSHParts: []dshMessagePart{{Type: "thought", Text: "最后核对"}, {Type: "text", End: len("完成")}},
		ToolRuns: []workspaceToolRun{{DSHCallID: "call", Preamble: "先读取", DSHParts: []dshMessagePart{{Type: "thought", Text: "检查文件"}, {Type: "text", End: len("先读取")}}, Report: reportedTool{Name: "read"}}},
	}
	row := acpSession{ID: "session", Turns: []acpTurn{turn}}
	parts := row.responseMessage(0, map[string]any{})["parts"].([]map[string]string)
	if len(parts) != 5 || parts[0]["text"] != "检查文件" || parts[1]["text"] != "先读取" || parts[2]["type"] != "tool_call" || parts[3]["text"] != "最后核对" || parts[4]["text"] != "完成" {
		t.Fatalf("wrong history order: %#v", parts)
	}
}
