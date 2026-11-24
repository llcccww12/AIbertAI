package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed dsh_host_disabled_plugins.txt
var dshHostDisabledPlugins string

// One local gateway belongs to exactly one platform session and one SDK child.
// The caller owns listener lifetime and must persist SDK settlement before Ack.
type dshLocalGateway struct {
	standardMode                             bool
	agentTeamEnabled                         *bool // New SDK sessions only. Existing sessions restore their journaled team/mode.
	gateway                                  *dshGateway
	selected                                 selectedModelConfig
	session, model, protocol, token, journal string
	mu                                       sync.Mutex
	pending                                  *dshLocalRecord
	operationID                              string
	// Set only for a version-pinned SDK that supplies persisted invocation IDs.
	requireInvocationID bool
	ackSignal           chan struct{}
	serving             chan struct{} // closed when the in-flight model handler returns
	dispatchGate        chan struct{}
	admitGoalInvocation func(string) error
	requestTooLarge     bool
	goalDenied          map[string]struct{}
	goalDeniedTerminal  bool
}

type dshLocalRecord struct {
	Finish        string `json:"finish,omitempty"`
	SessionID     string `json:"session_id"`
	OperationID   string `json:"operation_id,omitempty"`
	CallID        string `json:"call_id"`
	ModelID       string `json:"model_id"`
	Protocol      string `json:"protocol,omitempty"`
	UpstreamModel string `json:"upstream_model,omitempty"`
	Body          []byte `json:"body"`
	State         string `json:"state"` // dispatched, delivered; dispatched also covers unknown
	Status        int    `json:"status,omitempty"`
	Transaction   string `json:"transaction,omitempty"`
	GatewayCode   string `json:"gateway_code,omitempty"`
	FailureCode   string `json:"failure_code,omitempty"`
	// Unverified marks a failure settled from the SDK record alone, without a
	// final platform ledger receipt.
	Unverified bool `json:"unverified,omitempty"`
}

// newDSHLocalGateway requires a private, per-session journal directory. An
// existing pending record blocks all requests until independently reconciled.
func newDSHLocalGateway(g *dshGateway, session string, model selectedModelConfig, journal string) (*dshLocalGateway, error) {
	if !validUUID(session) || !validUUID(model.ID) || (model.Protocol != "openai_chat_completions" && model.Protocol != "openai_responses") {
		return nil, errors.New("invalid_dsh_gateway_config")
	}
	if err := os.MkdirAll(journal, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(journal)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("invalid_dsh_gateway_journal_directory")
	}
	token, err := randomState()
	if err != nil {
		return nil, err
	}
	local := &dshLocalGateway{gateway: g, selected: model, session: session, model: model.ID, protocol: model.Protocol, token: token, journal: journal, dispatchGate: make(chan struct{}, 1)}
	data, err := os.ReadFile(filepath.Join(journal, "pending.json"))
	if err == nil {
		var record dshLocalRecord
		if json.Unmarshal(data, &record) != nil || record.SessionID != session || record.ModelID != model.ID || (record.OperationID != "" && !validUUID(record.OperationID)) || !validUUID(record.CallID) || (record.State != "dispatched" && record.State != "delivered" && (record.State != "rejected" || !dshTerminalModelFailure(record.FailureCode))) {
			return nil, errors.New("invalid_dsh_gateway_journal")
		}
		local.pending = &record
		local.ackSignal = make(chan struct{})
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return local, nil
}

// resetTurnState is called only while the SDK session lease is exclusively
// held and settled. A pending invocation must still be reconciled first.
func (g *dshLocalGateway) resetTurnState() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.pending != nil {
		return errors.New("dsh_reconciliation_required")
	}
	g.requestTooLarge = false
	g.goalDenied = nil
	g.goalDeniedTerminal = false
	return nil
}

// Reconciliation can follow a durable journal repair while the SDK remains
// alive. Reload under the gateway lock; never trust stale in-memory absence.
func (g *dshLocalGateway) reloadPending() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	data, err := os.ReadFile(filepath.Join(g.journal, "pending.json"))
	if errors.Is(err, os.ErrNotExist) && g.pending == nil {
		return nil
	}
	if err != nil {
		return errors.New("dsh_reconciliation_required")
	}
	var record dshLocalRecord
	if json.Unmarshal(data, &record) != nil || record.SessionID != g.session || record.ModelID != g.model || (record.OperationID != "" && !validUUID(record.OperationID)) || !validUUID(record.CallID) || (record.State != "dispatched" && record.State != "delivered" && (record.State != "rejected" || !dshTerminalModelFailure(record.FailureCode))) {
		return errors.New("dsh_reconciliation_required")
	}
	if g.pending != nil && !reflect.DeepEqual(*g.pending, record) {
		return errors.New("dsh_reconciliation_required")
	}
	if g.pending == nil {
		g.pending = &record
		g.ackSignal = make(chan struct{})
	}
	return nil
}

// writePending is durable before any model request is sent.
func (g *dshLocalGateway) writePending(record *dshLocalRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(g.journal, ".pending-")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if err = file.Chmod(0600); err == nil {
		_, err = file.Write(data)
	}
	err = errors.Join(err, file.Sync(), file.Close())
	if err != nil {
		return err
	}
	if err = os.Rename(name, filepath.Join(g.journal, "pending.json")); err != nil {
		return err
	}
	return syncDSHJournal(g.journal)
}
func syncDSHJournal(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(d.Sync(), d.Close())
}

// Ack must only follow a durable, successful assistant settlement for this
// exact call. Unknown or interrupted calls require external reconciliation.
func (g *dshLocalGateway) Ack(callID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	// The SDK can settle on [DONE] before the upstream connection closes and
	// the handler records delivery. Wait for that handler to finish.
	if g.pending != nil && g.pending.State != "delivered" {
		g.waitServedLocked(callID)
	}
	if g.pending == nil || g.pending.CallID != callID || g.pending.State != "delivered" {
		return errors.New("dsh_gateway_not_delivered")
	}
	return g.archivePendingLocked(callID)
}

// waitServedLocked releases g.mu while the handler serving callID finishes.
// It reports false if the handler is still running after the wait.
func (g *dshLocalGateway) waitServedLocked(callID string) bool {
	served := g.serving
	if served == nil || g.pending == nil || g.pending.CallID != callID {
		return true
	}
	g.mu.Unlock()
	defer g.mu.Lock()
	select {
	case <-served:
		return true
	case <-time.After(10 * time.Second):
		return false
	}
}

func (g *dshLocalGateway) archivePendingLocked(callID string) error {
	if err := os.Rename(filepath.Join(g.journal, "pending.json"), filepath.Join(g.journal, callID+".json")); err != nil {
		return err
	}
	if err := syncDSHJournal(g.journal); err != nil {
		return err
	}
	g.pending = nil
	if g.ackSignal != nil {
		close(g.ackSignal)
		g.ackSignal = nil
	}
	return nil
}

// acceptSDKSettlement is called only on a notification or reconciliation
// response read from this session's authenticated SDK stdio connection.
func (g *dshLocalGateway) acceptSDKSettlement(session, invocationID, state string, seq int64) error {
	if g.requireInvocationID && session == g.session && validUUID(invocationID) && state == "failed" && seq > 0 {
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.pending != nil && g.pending.CallID == invocationID {
			if g.pending.State == "rejected" && dshTerminalModelFailure(g.pending.FailureCode) {
				return g.archivePendingLocked(invocationID)
			}
			if g.pending.Status >= 400 || g.pending.Status == 0 {
				return errors.New("dsh_model_outcome_unknown")
			}
			if g.pending.Status >= 200 && g.pending.Status < 300 {
				return errors.New("dsh_model_response_interrupted")
			}
		}
	}
	if !g.requireInvocationID || session != g.session || !validUUID(invocationID) || state != "committed" || seq <= 0 {
		return errors.New("invalid_dsh_gateway_settlement")
	}
	return g.Ack(invocationID)
}

type dshFailedCompactionReceipt struct {
	MemberInterrupt *dshMemberInterruptReceipt `json:"memberInterrupt,omitempty"`
	SessionID       string                     `json:"sessionId"`
	InvocationID    string                     `json:"invocationId"`
	OperationID     string                     `json:"operationId"`
	State           string                     `json:"state"`
	Seq             int64                      `json:"seq"`
	UsageSeq        *int64                     `json:"usageSeq"`
	Usage           *struct {
		Input  *int64 `json:"inputTokens"`
		Output *int64 `json:"outputTokens"`
	} `json:"usage"`
}

func (g *dshLocalGateway) validateFailedCompaction(receipt dshFailedCompactionReceipt, capable bool) error {
	if !capable || !g.requireInvocationID || receipt.SessionID != g.session || !validUUID(receipt.InvocationID) || !validUUID(receipt.OperationID) || receipt.State != "failed_with_usage" || receipt.UsageSeq == nil || *receipt.UsageSeq < 0 || receipt.Seq <= *receipt.UsageSeq || receipt.Usage == nil || receipt.Usage.Input == nil || receipt.Usage.Output == nil || *receipt.Usage.Input < 0 || *receipt.Usage.Output < 0 {
		return errors.New("invalid_dsh_gateway_settlement")
	}
	g.mu.Lock()
	match := g.pending != nil && g.pending.CallID == receipt.InvocationID && g.pending.OperationID == receipt.OperationID && g.pending.State == "delivered"
	g.mu.Unlock()
	if !match {
		return errors.New("invalid_dsh_gateway_settlement")
	}
	return nil
}

func (g *dshLocalGateway) acceptFailedCompaction(receipt dshFailedCompactionReceipt, capable bool) error {
	if err := g.validateFailedCompaction(receipt, capable); err != nil {
		return err
	}
	return g.Ack(receipt.InvocationID)
}

// A denial recorded under the gateway lock happened before dispatch. This
// process-local evidence is valid only for the matching failed SDK receipt.
func (g *dshLocalGateway) deniedGoalSettlement(session, id, state string, seq int64, consume bool) bool {
	if session != g.session || !validUUID(id) || state != "failed" || seq <= 0 {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	_, ok := g.goalDenied[id]
	if ok && consume {
		delete(g.goalDenied, id)
	}
	return ok
}

func (g *dshLocalGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+g.token)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	expected := "/v1/chat/completions"
	if g.protocol == "openai_responses" {
		expected = "/v1/responses"
	}
	if r.URL.Path != expected || r.URL.RawQuery != "" {
		http.Error(w, "invalid route", http.StatusNotFound)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, dshGatewayMaxRequestBytes+1))
	if len(body) > dshGatewayMaxRequestBytes {
		g.mu.Lock()
		g.requestTooLarge = true
		g.mu.Unlock()
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	var request struct {
		Model string `json:"model"`
	}
	if json.Unmarshal(body, &request) != nil || request.Model != g.model {
		http.Error(w, "invalid model", http.StatusBadRequest)
		return
	}
	id := r.Header.Get("X-AIBERTAI-Invocation-ID")
	// In-process teammates share their root's single durable billing journal.
	// Queue live requests until the previous stream has finished delivering.
	if g.requireInvocationID {
		select {
		case g.dispatchGate <- struct{}{}:
			defer func() { <-g.dispatchGate }()
		case <-r.Context().Done():
			return
		}
	}
	g.mu.Lock()
	if g.pending != nil && g.requireInvocationID && validUUID(id) && id != g.pending.CallID && g.ackSignal != nil {
		wait := g.ackSignal
		g.mu.Unlock()
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-wait:
		case <-r.Context().Done():
			return
		case <-timer.C:
			http.Error(w, "model call requires reconciliation", http.StatusConflict)
			return
		}
		g.mu.Lock()
	}
	if g.pending != nil {
		g.mu.Unlock()
		http.Error(w, "model call requires reconciliation", http.StatusConflict)
		return
	}
	if g.goalDeniedTerminal {
		g.mu.Unlock()
		http.Error(w, "goal invocation not admitted", http.StatusConflict)
		return
	}
	if g.requireInvocationID {
		if len(r.Header.Values("X-AIBERTAI-Invocation-ID")) != 1 || !validUUID(id) {
			g.mu.Unlock()
			http.Error(w, "invalid invocation", http.StatusBadRequest)
			return
		}
		_, statErr := os.Lstat(filepath.Join(g.journal, id+".json"))
		if statErr == nil || !errors.Is(statErr, os.ErrNotExist) {
			g.mu.Unlock()
			http.Error(w, "invocation requires reconciliation", http.StatusConflict)
			return
		}
	} else {
		id, err = newUUID()
		if err != nil {
			g.mu.Unlock()
			http.Error(w, "unavailable", http.StatusInternalServerError)
			return
		}
	}
	if g.admitGoalInvocation != nil {
		if err := g.admitGoalInvocation(id); err != nil {
			if err.Error() == "goal_not_active" || err.Error() == "goal_token_budget" {
				if g.goalDenied == nil {
					g.goalDenied = make(map[string]struct{})
				}
				g.goalDenied[id] = struct{}{}
				g.goalDeniedTerminal = true
			}
			g.mu.Unlock()
			http.Error(w, "goal invocation not admitted", http.StatusConflict)
			return
		}
	}
	// Catalog identifiers append the platform UUID; billing hashes the request
	// after the proxy substitutes the upstream model. A changed model still
	// fails the server's exact hash check during recovery.
	upstreamModel := strings.TrimSuffix(g.selected.Model, "@"+g.model)
	record := &dshLocalRecord{SessionID: g.session, OperationID: g.operationID, CallID: id, ModelID: g.model, Protocol: g.protocol, UpstreamModel: upstreamModel, Body: body, State: "dispatched"}
	g.pending = record
	g.ackSignal = make(chan struct{})
	served := make(chan struct{})
	g.serving = served
	defer func() {
		g.mu.Lock()
		if g.serving == served {
			g.serving = nil
		}
		g.mu.Unlock()
		close(served)
	}()
	if err = g.writePending(record); err != nil {
		g.mu.Unlock()
		http.Error(w, "journal unavailable", http.StatusInternalServerError)
		return
	}
	g.mu.Unlock()
	resp, err := g.invokeWithPreflightRetry(r.Context(), dshGatewayCall{SessionID: g.session, CallID: id, ModelID: g.model, Body: body})
	if err != nil {
		var preflight *dshPreflightError
		if errors.As(err, &preflight) && r.Context().Err() == nil {
			status := dshPreflightStatus(preflight)
			code := dshFailureCode(status)
			if g.noteModelRejection(record, status, code) == nil {
				writeDSHModelError(w, status, code)
				return
			}
		}
		writeDSHModelError(w, http.StatusConflict, "dsh_model_outcome_unknown")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 && resp.Header.Get(dshNotDispatchedHeader) == id && resp.Header.Get("X-Billing-Transaction-ID") == "" {
		code := dshFailureCode(resp.StatusCode)
		if g.noteModelRejection(record, resp.StatusCode, code) == nil {
			writeDSHModelError(w, resp.StatusCode, code)
			return
		}
		writeDSHModelError(w, http.StatusConflict, "dsh_model_outcome_unknown")
		return
	}
	transaction := resp.Header.Get("X-Billing-Transaction-ID")
	code := ""
	if resp.StatusCode == http.StatusConflict {
		// The billing API returns the original transaction in its JSON error,
		// not necessarily in a header. Preserve the body for the SDK while
		// retaining only the safe identifiers needed for reconciliation.
		prefix, readErr := io.ReadAll(io.LimitReader(resp.Body, (64<<10)+1))
		if readErr != nil {
			http.Error(w, "model call outcome unknown", http.StatusBadGateway)
			return
		}
		resp.Body = &dshPrefixedBody{Reader: io.MultiReader(bytes.NewReader(prefix), resp.Body), Closer: resp.Body}
		if len(prefix) <= 64<<10 {
			var result struct {
				Error struct {
					Code       string `json:"code"`
					References struct {
						Transaction string `json:"transaction_id"`
					} `json:"references"`
				} `json:"error"`
			}
			if json.Unmarshal(prefix, &result) == nil && (result.Error.Code == "request_already_accepted" || result.Error.Code == "idempotency_conflict") && validUUID(result.Error.References.Transaction) {
				code, transaction = result.Error.Code, result.Error.References.Transaction
			}
		}
	}
	if transaction != "" && !validUUID(transaction) {
		transaction = ""
	}
	g.mu.Lock()
	record.Status = resp.StatusCode
	record.Transaction = transaction
	record.GatewayCode = code
	err = g.writePending(record)
	g.mu.Unlock()
	if err != nil {
		http.Error(w, "journal unavailable", http.StatusInternalServerError)
		return
	}
	// Once accepted upstream, DSH must not automatically resubmit an HTTP
	// failure. Only the platform ledger can authorize a subsequent fresh step.
	if resp.StatusCode >= 400 {
		writeDSHModelError(w, http.StatusConflict, "dsh_model_outcome_unknown")
		return
	}
	for _, name := range []string{"Content-Type", "X-Billing-Transaction-ID"} {
		if value := resp.Header.Get(name); value != "" {
			w.Header().Set(name, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	// Flush each chunk: DSH receives progress without waiting for stream completion.
	buf := make([]byte, 32<<10)
	complete := true
	stream := dshStreamCompletion{protocol: g.protocol, valid: strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream")}
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			stream.add(buf[:n])
			if _, err = w.Write(buf[:n]); err != nil {
				complete = false
				break
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				complete = false
			}
			break
		}
	}
	if !complete || !stream.complete() || resp.StatusCode < 200 || resp.StatusCode >= 300 || r.Context().Err() != nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	record.State = "delivered"
	record.Finish = stream.terminal
	record.Status = resp.StatusCode
	record.Transaction = transaction
	if err := g.writePending(record); err != nil {
		record.State = "dispatched"
	}
}

type dshPrefixedBody struct {
	io.Reader
	io.Closer
}

// A clean EOF is not evidence of a complete model stream. Keep ambiguous,
// truncated, malformed and non-SSE responses pending for reconciliation.
type dshStreamCompletion struct {
	protocol            string
	terminal            string
	pending             []byte
	valid, done, finish bool
}

func (s *dshStreamCompletion) add(chunk []byte) {
	if !s.valid {
		return
	}
	for len(chunk) > 0 {
		end := bytes.IndexByte(chunk, '\n')
		if end < 0 {
			if len(s.pending)+len(chunk) > 1<<20 {
				s.valid = false
				return
			}
			s.pending = append(s.pending, chunk...)
			return
		}
		if len(s.pending)+end > 1<<20 {
			s.valid = false
			return
		}
		s.pending = append(s.pending, chunk[:end]...)
		s.line(bytes.TrimSuffix(s.pending, []byte{'\r'}))
		s.pending = s.pending[:0]
		chunk = chunk[end+1:]
	}
}

func (s *dshStreamCompletion) line(line []byte) {
	if !bytes.HasPrefix(line, []byte("data:")) {
		return
	}
	data := bytes.TrimSpace(line[len("data:"):])
	if s.done {
		// The SDK may stop reading at [DONE]. Data after it cannot be used
		// to prove the same usage or result was delivered to the SDK.
		s.valid = false
		return
	}
	if s.protocol == "openai_responses" && s.finish && !bytes.Equal(data, []byte("[DONE]")) {
		s.valid = false
		return
	}
	if bytes.Equal(data, []byte("[DONE]")) {
		s.done = true
		return
	}
	var event struct {
		Type    string          `json:"type"`
		Error   json.RawMessage `json:"error"`
		Choices []struct {
			Finish *string `json:"finish_reason"`
		} `json:"choices"`
		Response struct {
			Status            string `json:"status"`
			IncompleteDetails struct {
				Reason string `json:"reason"`
			} `json:"incomplete_details"`
		} `json:"response"`
	}
	if json.Unmarshal(data, &event) != nil {
		s.valid = false
		return
	}
	if len(event.Error) > 0 && !bytes.Equal(event.Error, []byte("null")) {
		s.valid = false
		return
	}
	if s.protocol == "openai_responses" {
		switch event.Type {
		case "response.completed":
			s.finish = event.Response.Status == "completed"
		case "response.incomplete":
			if event.Response.Status == "incomplete" && event.Response.IncompleteDetails.Reason == "max_output_tokens" {
				s.finish = true
				s.terminal = "length"
			} else if event.Response.Status == "incomplete" && event.Response.IncompleteDetails.Reason == "content_filter" {
				s.finish = true
				s.terminal = "content_filter"
			} else {
				s.valid = false
			}
		case "response.failed", "error":
			s.valid = false
		}
		return
	}
	for _, choice := range event.Choices {
		if choice.Finish == nil {
			continue
		}
		switch *choice.Finish {
		case "stop", "tool_calls", "function_call", "length", "content_filter":
			s.finish = true
			s.terminal = *choice.Finish
		default:
			s.valid = false
		}
	}
}
func (s *dshStreamCompletion) complete() bool {
	return s.valid && len(s.pending) == 0 && s.finish && (s.protocol == "openai_responses" || s.done)
}

// Start binds only loopback. The returned token is a child-only capability and
// must never be written into the config file, logs, or model request body.
func (g *dshLocalGateway) Start() (string, func(context.Context) error, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	server := &http.Server{Handler: g, ReadHeaderTimeout: 5 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() { _ = server.Serve(listener) }()
	return "http://" + listener.Addr().String() + "/v1", server.Shutdown, nil
}

// SDK runtime writes this JSON document as a YAML patch for the sdk profile and provides the token
// in DSH_AIBERTAI_LOCAL_TOKEN. It passes provider aibertai and model.ID to initialize.
func (g *dshLocalGateway) Patch(baseURL string, model selectedModelConfig, disabledPlugins ...string) ([]byte, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Hostname() != "127.0.0.1" || u.Path != "/v1" || u.RawQuery != "" || u.Fragment != "" || model.ID != g.model || model.Protocol != g.protocol {
		return nil, errors.New("invalid_dsh_gateway_config")
	}
	port, err := strconv.ParseUint(u.Port(), 10, 16)
	if err != nil || port == 0 {
		return nil, errors.New("invalid_dsh_gateway_config")
	}
	api := "openai-completions"
	if g.protocol == "openai_responses" {
		api = "openai-responses"
	}
	name := model.DisplayName
	if name == "" {
		name = model.ID
	}
	entry := map[string]any{"id": model.ID, "name": name}
	if model.SupportsVision {
		entry["input"] = []string{"text", "image"}
	}
	if model.ContextWindowTokens > 0 {
		entry["contextWindow"] = model.ContextWindowTokens
	}
	if model.MaxOutputTokens > 0 {
		entry["maxTokens"] = model.MaxOutputTokens
	}
	patch := []any{map[string]any{"id": "llm-pi-ai", "config": map[string]any{"providers": map[string]any{"aibertai": map[string]any{"api": api, "baseURL": baseURL, "apiKeyEnv": "DSH_AIBERTAI_LOCAL_TOKEN", "transport": "sse", "models": []any{entry}, "retryPolicy": map[string]any{"mode": "normal", "maxRetries": 0}}}}}}
	// The SDK's 64K headroom can leave smaller routes with no valid pressure
	// threshold. Keep its defaults wherever its 16% verbatim tail still fits.
	const defaultHeadroom = 65536
	headroom := defaultHeadroom
	inputBudget := model.ContextWindowTokens - max(0, model.MaxOutputTokens)
	if inputBudget > 0 && inputBudget-defaultHeadroom <= inputBudget*16/100 {
		headroom = inputBudget / 5
	}
	summaryLimit := max(1, headroom)
	if model.MaxOutputTokens > 0 {
		summaryLimit = min(summaryLimit, model.MaxOutputTokens)
	}
	if headroom != defaultHeadroom || summaryLimit != defaultHeadroom {
		policy := map[string]any{"provider": "aibertai", "model": model.ID, "maxTokens": summaryLimit}
		if headroom != defaultHeadroom {
			policy["headroomTokens"] = headroom
		}
		patch = append(patch, map[string]any{"id": "compaction-basic", "config": map[string]any{"modelPolicies": []any{policy}}})
	}
	patch = append(patch, map[string]any{"id": "agent-team", "config": map[string]bool{"defaultEnabled": (g.agentTeamEnabled == nil || *g.agentTeamEnabled) && !slices.Contains(disabledPlugins, "tool-agent-team")}})
	// Web Standard enables discovery and selection on spawn. DSH limits both to
	// the parent's authorized route when no other session routes were granted.
	if g.standardMode {
		patch = append(patch, map[string]any{"id": "tool-subagent", "config": map[string]any{
			"provider": "spawn", "toolName": "subagent", "modelDiscovery": true, "modelSelection": true,
		}})
	}
	// Plugin management always requires a fresh user grant, including in a full-access SDK session.
	patch = append(patch, map[string]any{"id": "tool-plugin-manager", "config": map[string]any{"requireApproval": true, "manageableEntries": dshManagedEntries}})
	// The SDK scheduler owns one host-wide task file and delivers directly to SDK agents.
	// A child leased to one platform session cannot own that file or authorize timed turns.
	// Pin the SDK HMR policy above editable profile settings for the lifetime of the lease.
	for _, id := range append(strings.Fields(dshHostDisabledPlugins), disabledPlugins...) {
		patch = append(patch, map[string]any{"id": id, "disabled": true})
	}
	data, err := json.Marshal(patch)
	if err != nil {
		return nil, fmt.Errorf("dsh gateway patch: %w", err)
	}
	return data, nil
}
