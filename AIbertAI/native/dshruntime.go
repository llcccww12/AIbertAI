package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// The installer supplies the fixed SDK profile and persistence location; the
// router supplies the authorized model route. Neither is enabled by default.
type dshRuntimeConfig struct {
	VerifiedResourceRoot string
	// JournalRoot holds each session's model-call journal (<root>/<session>/pending.json).
	JournalRoot            string
	SharedPluginHome       string
	ResolveDisabledPlugins func(context.Context, string) ([]string, error)
	LeasePool              *dshLeasePool
	ValidateLease          func(context.Context, acpSession) (selectedModelConfig, error)
	BindLease              func(context.Context, acpSession, string, *dshLocalGateway) error
	Command                func(context.Context, acpSession) (*exec.Cmd, error)
	Provider               func(context.Context, acpSession) (string, string, error)
	Prepare                func(context.Context, acpSession) (*exec.Cmd, string, string, *dshLocalGateway, func(), error)
}

// dshInstalledRuntime describes a platform-provisioned SDK bundle. Env is an
// explicit allowlist; ambient permission-mode variables are not inherited.
type dshInstalledRuntime struct {
	StandardSkillsDir string // Bundled with the verified Standard preset.
	Executable        string
	Entry             string
	Home              string
	Patch             string
	Env               []string
	Provider          func(context.Context, acpSession) (string, string, error)
}

// dshTrustedLaunchKey marks a turn whose SDK child may start with a writable
// DSH preset: a local, interactive turn whose caller holds files:write.
// promptACP sets it; nothing else does.
type dshTrustedLaunchKey struct{}

// dshSessionPermissionMode is the DSH permission preset a fresh SDK session is
// pinned to (DSH records it in the session and keeps it on resume), following
// the conversation's level: 仅可查看 is read-only, 工作区内修改 is DSH's
// default workspace-write, and 完全权限 is DSH's danger-full-access, whose
// approval policy "never" runs tools without asking the host. Launches that
// are not trusted (remote relay, background goals, callers without
// files:write) stay read-only and escalate through the host.
func dshSessionPermissionMode(ctx context.Context, session acpSession) string {
	if trusted, _ := ctx.Value(dshTrustedLaunchKey{}).(bool); trusted {
		switch session.permissionMode() {
		case modeWorkspaceWrite:
			return "workspace-write"
		case modeFullAccess:
			return "danger-full-access"
		}
	}
	return "read-only"
}

func (r dshInstalledRuntime) config() (*dshRuntimeConfig, error) {
	if !filepath.IsAbs(r.Executable) || !filepath.IsAbs(r.Home) || r.Entry != "" && !filepath.IsAbs(r.Entry) || r.Patch != "" && !filepath.IsAbs(r.Patch) || r.Provider == nil {
		return nil, errors.New("invalid_dsh_runtime_config")
	}
	for _, e := range r.Env {
		key, _, ok := strings.Cut(e, "=")
		if !ok || strings.EqualFold(key, "DSH_PERMISSION_MODE") || strings.EqualFold(key, "DSH_HOME") {
			return nil, errors.New("invalid_dsh_runtime_environment")
		}
	}
	return &dshRuntimeConfig{Provider: r.Provider, Command: func(ctx context.Context, session acpSession) (*exec.Cmd, error) {
		args := []string{}
		if r.Entry != "" {
			args = append(args, r.Entry)
		}
		args = append(args, "--profile", "sdk")
		if r.Patch != "" {
			args = append(args, "--patch", r.Patch)
		}
		cmd := exec.CommandContext(ctx, r.Executable, args...)
		cmd.Dir = session.CWD
		cmd.Env = append(append([]string{}, r.Env...), "DSH_HOME="+r.Home, "DSH_TELEMETRY_DISABLED=1", "DSH_PERMISSION_MODE="+dshSessionPermissionMode(ctx, session))
		if session.ExpertID == "" && r.StandardSkillsDir != "" {
			cmd.Env = append(cmd.Env, "DSH_BUNDLED_SKILL_DIR="+r.StandardSkillsDir)
		}
		cmd.Stderr = os.Stderr
		return cmd, nil
	}}, nil
}

type dshFrame struct {
	JSONRPC string `json:"jsonrpc"`
	readErr error
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}
type dshProcess struct {
	writeMu                      sync.Mutex
	observerReplies              sync.Map // human control requests are independent of the model turn
	team                         dshTeamSessions
	teamSettled                  func(dshFailedCompactionReceipt) error
	sharedProfile                dshSharedProfile
	cmd                          *exec.Cmd
	in                           io.WriteCloser
	frames                       chan dshFrame
	done                         chan error
	stop                         chan struct{}
	next                         int
	approval                     func(json.RawMessage) (string, error)
	question                     dshQuestionHandler
	questions                    map[string]*pendingQuestion
	questionReplies              chan dshQuestionReply
	notification                 func(string, json.RawMessage) error
	hostRequest                  func(json.RawMessage) (any, error)
	hostTool                     func(json.RawMessage) (any, error)
	afterEvent                   func(json.RawMessage) error
	deferred                     []dshFrame
	deferredBytes                int
	activity                     func(string, json.RawMessage) error
	cancelSession                string
	cancelAccepted               bool
	leaseSession, leaseOperation string
	workspaceBoundary            bool
	failedCompactionUsage        bool
}

func startDSH(ctx context.Context, cmd *exec.Cmd) (*dshProcess, error) {
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		in.Close()
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		in.Close()
		return nil, err
	}
	p := &dshProcess{cmd: cmd, in: in, frames: make(chan dshFrame), done: make(chan error, 1), stop: make(chan struct{}), questions: make(map[string]*pendingQuestion), questionReplies: make(chan dshQuestionReply, 1)}
	go func() {
		defer close(p.frames)
		defer func() { p.done <- cmd.Wait() }()
		s := bufio.NewScanner(out)
		var assembler dshEventAssembler
		s.Buffer(make([]byte, 4096), 8<<20)
		for s.Scan() {
			var f dshFrame
			if json.Unmarshal(s.Bytes(), &f) != nil || f.JSONRPC != "2.0" || (f.Method == "" && (len(f.ID) == 0 || string(f.ID) == "null")) {
				f = dshFrame{readErr: errors.New("dsh_invalid_protocol_frame")}
			}
			if p.deliverObserverReply(f) {
				continue
			}
			if f.readErr == nil {
				var emit bool
				var err error
				f, emit, err = assembler.accept(f)
				if err != nil {
					f = dshFrame{readErr: err}
				} else if !emit {
					continue
				}
			}
			select {
			case p.frames <- f:
			case <-ctx.Done():
				return
			case <-p.stop:
				return
			}
			if f.readErr != nil {
				return
			}
		}
		if err := s.Err(); err != nil {
			select {
			case p.frames <- dshFrame{readErr: fmt.Errorf("dsh_protocol_read_failed: %w", err)}:
			case <-ctx.Done():
			case <-p.stop:
			}
		}
	}()
	return p, nil
}
func (p *dshProcess) write(v any) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	return json.NewEncoder(p.in).Encode(v)
}

// historyCall owns the existing reader briefly; other frames are replayed by
// the outer call only after the durable history check has completed.
func (p *dshProcess) historyCall(ctx context.Context, params map[string]any) (json.RawMessage, error) {
	return p.boundaryCall(ctx, "aibertai/history", params)
}
func (p *dshProcess) boundaryCall(ctx context.Context, method string, params map[string]any) (json.RawMessage, error) {
	p.next++
	id := p.next
	if err := p.write(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case f, ok := <-p.frames:
			if !ok {
				return nil, errors.New("dsh_runtime_exited")
			}
			if f.readErr != nil {
				return nil, f.readErr
			}
			var received int
			if f.Method == "" && json.Unmarshal(f.ID, &received) == nil && received == id {
				if f.Error != nil {
					return nil, fmt.Errorf("dsh history rpc: %d: %s", f.Error.Code, f.Error.Message)
				}
				return f.Result, nil
			}
			if err := p.deferFrame(f); err != nil {
				return nil, err
			}
		}
	}
}

// A file change that may have applied is reported, not replayed. The model must
// inspect the file before deciding; the harness never repeats the mutation.
func dshWorkspaceToolReason(err error) string {
	switch {
	case strings.Contains(err.Error(), "dsh_connector_outcome_unknown"):
		return "write_outcome_unknown: this file change was interrupted and its outcome is unknown; it may or may not have been applied. Read the file first, then decide whether to change it again."
	case strings.Contains(err.Error(), "dsh_connector_dispatch_uncertain"):
		return "write_not_dispatched: this change was not applied because another write in this turn is still unverified. Read the file to confirm its state, then continue."
	case strings.Contains(err.Error(), "dsh_workspace_changed"), strings.Contains(err.Error(), "dsh_workspace_conflict"):
		return "write_conflict: the file changed since it was read, so this change was not applied. Re-read and retry."
	case errors.Is(err, errWorkspaceReadOnly):
		return "read_only: this conversation is read-only, so the change was not applied. If the change is needed, call the tool again with sandbox_permissions \"workspace-write\" and a justification so the user can approve it, or explain the limitation to the user."
	case errors.Is(err, errWorkspacePath):
		return "outside_workspace: write and edit only change files inside this conversation's workspace, so this change was not applied. Use a path inside the workspace, or change the file with bash; outside the workspace bash needs sandbox_permissions with a justification unless the session already has full access."
	}
	return "workspace_mutation_denied: the change was not applied. Choose another approach or explain the limitation to the user."
}

// The model, not the session, decides what a failed tool means. The reason is
// model-facing, so it states what happened and what may not be repeated.
func dshHostToolReason(err error) string {
	switch {
	case strings.Contains(err.Error(), "dsh_canvas_outcome_unknown"):
		return "canvas_outcome_unknown: this canvas write was interrupted and its outcome is unknown; it may or may not have been applied. Read the canvas first. Repeat the write only if it is missing, otherwise continue from what is there or explain the limitation to the user."
	case strings.Contains(err.Error(), "dsh_connector_outcome_unknown"), strings.Contains(err.Error(), "dsh_connector_dispatch_uncertain"):
		return "connector_outcome_unknown: this connector call was interrupted and its outcome is unknown; the remote side may have completed it. Verify the remote state or ask the user before repeating it."
	case strings.Contains(err.Error(), "tool_approval_unavailable"), strings.Contains(err.Error(), "tool_permission_cancelled"):
		return "tool_denied: the approval channel was unavailable or the call was cancelled. The tool did not run; choose another approach or ask the user."
	case strings.Contains(err.Error(), "design_media_model_unavailable"):
		return "image_model_unavailable: no image generation model is configured or available for this conversation, so nothing was generated. Do not retry image tools in this turn. Tell the user an administrator needs to enable an image model (or that they can pick one in the canvas), and offer an alternative such as an SVG/CSS illustration or a clearly labeled placeholder."
	case strings.Contains(err.Error(), "design_media_cancelled"):
		return "image_generation_cancelled: the image generation was cancelled and produced no image. Do not retry unless the user asks."
	case strings.Contains(err.Error(), "design_media_generation_unknown"), strings.Contains(err.Error(), "design_media_generation_already_submitted"):
		return "media_generation_unconfirmed: the media service accepted the task but could not confirm its result, and nothing was saved to the workspace. It may still be billed, so do not submit it again in this turn. Tell the user the generation could not be confirmed and that they can check their usage records or try again later."
	case strings.HasPrefix(err.Error(), "design_media_"):
		return "media_generation_failed (" + strings.TrimPrefix(err.Error(), "design_") + "): the media service did not produce a result and nothing was saved. Tell the user it failed; do not repeat the same call unchanged in this turn."
	case strings.Contains(err.Error(), "dsh_canvas_atomic_guard_unavailable"), strings.Contains(err.Error(), "dsh_canvas_changed"), strings.Contains(err.Error(), "dsh_canvas_session_changed"):
		return "canvas_conflict: the canvas changed underneath this call, so it was not applied. Re-read the canvas and retry."
	}
	if code := err.Error(); strings.HasPrefix(code, "design_") && !strings.ContainsAny(code, " \n") && code != "design_tool_unavailable" && code != "design_tool_failed" {
		return "design_tool_error (" + strings.TrimPrefix(code, "design_") + "): the design service did not complete this call and nothing was applied. Adjust the request or tell the user; do not repeat the same call unchanged."
	}
	return "host_tool_denied: the tool failed before it could apply. It is safe to use a different tool or approach."
}

// Reads notifications in wire order before the response. Native approvals are
// rejected unless the host binds them to a durable, pending tool call.
func (p *dshProcess) stopQuestionState() {
	for id, pending := range p.questions {
		pending.cancel()
		delete(p.questions, id)
	}
}

func (p *dshProcess) call(ctx context.Context, method string, params any, event func(json.RawMessage) error) (result json.RawMessage, callErr error) {
	defer func() {
		if callErr != nil || method == "session/wait" || method == "aibertai/query-wait" {
			p.stopQuestionState()
		}
	}()
	p.next++
	id := p.next
	if err := p.write(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	done := ctx.Done()
	var deadline <-chan time.Time
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	var cancelling, cancelAck bool
	var stopErr error
	var saved *dshFrame
	beginCancel := func() error {
		if cancelling {
			return nil
		}
		if p.cancelSession == "" || (method != "session/wait" && method != "aibertai/query-wait" && method != "session/query" && method != "session/prompt" && method != "aibertai/member-prompt") {
			return ctx.Err()
		}
		if err := p.write(map[string]any{"jsonrpc": "2.0", "id": -1, "method": "aibertai/session-cancel", "params": map[string]string{"sessionId": p.cancelSession}}); err != nil {
			return errors.Join(ctx.Err(), err)
		}
		cancelling = true
		done = nil
		// A root interrupt also drains delegated agents before acknowledging.
		timer = time.NewTimer(25 * time.Second)
		deadline = timer.C
		return nil
	}
	finish := func(f dshFrame) (json.RawMessage, error) {
		if stopErr != nil {
			return nil, stopErr
		}
		if f.Error != nil {
			return nil, fmt.Errorf("dsh rpc %s: %d: %s", method, f.Error.Code, f.Error.Message)
		}
		return f.Result, nil
	}
	for {
		frames := (<-chan dshFrame)(p.frames)
		var staged chan dshFrame
		if len(p.deferred) != 0 {
			staged = make(chan dshFrame, 1)
			staged <- p.deferred[0]
			frames = staged
		}
		select {
		case reply := <-p.questionReplies:
			pending := p.questions[reply.requestID]
			if pending == nil || pending != reply.pending {
				continue
			}
			delete(p.questions, reply.requestID)
			pending.cancel()
			if ctx.Err() != nil && !cancelling {
				if err := beginCancel(); err != nil {
					return nil, err
				}
			}
			response := map[string]any{"jsonrpc": "2.0", "id": reply.rpcID}
			if reply.err != nil {
				response["error"] = map[string]any{"code": -32603, "message": "user_question_unavailable"}
			} else {
				response["result"] = reply.result
			}
			if err := p.write(response); err != nil {
				return nil, err
			}
		case <-done:
			if err := beginCancel(); err != nil {
				return nil, err
			}
		case <-deadline:
			if stopErr != nil {
				return nil, stopErr
			}
			return nil, ctx.Err()
		case f, ok := <-frames:
			if staged != nil {
				p.deferredBytes -= len(f.ID) + len(f.Params) + len(f.Result) + len(f.Method)
				p.deferred = p.deferred[1:]
			}
			if !ok {
				return nil, errors.New("dsh_runtime_exited")
			}
			if f.readErr != nil {
				return nil, f.readErr
			}
			if f.Method != "" {
				if len(f.ID) == 0 && p.teamSettled != nil {
					p.team.track(f.Method, f.Params)
					if f.Method == "aibertai/model-settled" {
						var receipt dshFailedCompactionReceipt
						if json.Unmarshal(f.Params, &receipt) == nil && receipt.SessionID != p.team.root {
							if p.team.calls[receipt.InvocationID] != receipt.SessionID || receipt.SessionID == "" {
								return nil, errors.New("dsh_invalid_settlement")
							}
							if err := p.teamSettled(receipt); err != nil {
								return nil, err
							}
							delete(p.team.calls, receipt.InvocationID)
							continue
						}
					}
				}
				if len(f.ID) == 0 && p.leaseOperation != "" && (f.Method == "session.event" || f.Method == "aibertai/assistant-stream") && !dshLeaseEventMatches(f.Params, p.leaseSession, p.leaseOperation) {
					continue
				}
				if f.Method == "user-question/ask" && len(f.ID) != 0 && p.question != nil {
					var request struct {
						RequestID string `json:"requestId"`
						SessionID string `json:"sessionId"`
					}
					if json.Unmarshal(f.Params, &request) != nil || request.RequestID == "" || request.SessionID == "" || p.questions[request.RequestID] != nil || p.leaseOperation != "" && !dshLeaseRequestMatches(f.Params, p.leaseSession, p.leaseOperation) {
						if err := p.write(map[string]any{"jsonrpc": "2.0", "id": f.ID, "error": map[string]any{"code": -32602, "message": "invalid_user_question"}}); err != nil {
							return nil, err
						}
						continue
					}
					questionCtx, cancel := context.WithCancel(ctx)
					pending := &pendingQuestion{sessionID: request.SessionID, cancel: cancel}
					p.questions[request.RequestID] = pending
					handler := p.question
					go func(requestID string, id, raw json.RawMessage, pending *pendingQuestion) {
						result, err := handler(questionCtx, raw)
						select {
						case p.questionReplies <- dshQuestionReply{requestID, id, result, err, pending}:
						case <-questionCtx.Done():
						case <-p.stop:
						}
					}(request.RequestID, f.ID, f.Params, pending)
					continue
				}
				if f.Method == "user-question.cancelled" && len(f.ID) == 0 {
					var request struct {
						RequestID string `json:"requestId"`
						SessionID string `json:"sessionId"`
					}
					if json.Unmarshal(f.Params, &request) == nil && (p.leaseOperation == "" || dshLeaseRequestMatches(f.Params, p.leaseSession, p.leaseOperation)) {
						if pending := p.questions[request.RequestID]; pending != nil && pending.sessionID == request.SessionID {
							delete(p.questions, request.RequestID)
							pending.cancel()
						}
					}
					continue
				}
				if len(f.ID) != 0 {
					response := map[string]any{"jsonrpc": "2.0", "id": f.ID}
					if f.Method == "approval/request" {
						outcome := "rejected"
						if p.approval != nil {
							var err error
							outcome, err = p.approval(f.Params)
							if err != nil {
								return nil, err
							}
						}
						response["result"] = map[string]string{"outcome": outcome}
					} else if f.Method == "aibertai/workspace-mutate" && p.hostRequest != nil {
						result, err := p.hostRequest(f.Params)
						if err != nil {
							if errors.Is(err, errDSHGoalApprovalUnavailable) {
								stopErr = err
								if cancelErr := beginCancel(); cancelErr != nil {
									return nil, err
								}
							}
							// A failed file change is a tool outcome, not a session
							// failure. The model receives the reason and decides.
							if ctx.Err() != nil {
								return nil, err
							}
							response["error"] = map[string]any{"code": -32603, "message": dshWorkspaceToolReason(err)}
						} else {
							response["result"] = result
						}
					} else if f.Method == "aibertai/host-tool" && p.hostTool != nil {
						result, err := p.hostTool(f.Params)
						if err != nil {
							// The SDK reports a failed tool as an isError result and the
							// model chooses the next step. Only a dead context may end
							// the run; a tool outcome never may.
							if ctx.Err() != nil {
								return nil, err
							}
							response["error"] = map[string]any{"code": -32603, "message": dshHostToolReason(err)}
						} else {
							response["result"] = result
						}
					} else {
						response["error"] = map[string]any{"code": -32601, "message": "unsupported host request"}
					}
					// Stop the Agent before releasing a failed approval back to the SDK.
					if ctx.Err() != nil && !cancelling && (method == "session/wait" || method == "aibertai/query-wait" || method == "session/query") {
						if err := beginCancel(); err != nil {
							return nil, err
						}
					}
					if f.Method == "approval/request" && ctx.Err() != nil {
						response["result"] = map[string]string{"outcome": "rejected"}
					}
					if err := p.write(response); err != nil {
						return nil, err
					}
				} else if f.Method == "aibertai/model-settled" && p.notification != nil {
					if p.activity != nil {
						if err := p.activity(f.Method, f.Params); err != nil {
							return nil, err
						}
					}
					if err := p.notification(f.Method, f.Params); err != nil {
						return nil, err
					}
				} else if f.Method == "session.event" && event != nil {
					if err := event(f.Params); err != nil {
						return nil, err
					}
					if p.activity != nil {
						if err := p.activity(f.Method, f.Params); err != nil {
							return nil, err
						}
					}
					if p.afterEvent != nil {
						if err := p.afterEvent(f.Params); err != nil {
							return nil, err
						}
					}
				} else if (f.Method == "aibertai/assistant-stream" || f.Method == "session.query.completed") && p.activity != nil {
					if err := p.activity(f.Method, f.Params); err != nil {
						return nil, err
					}
				}
				continue
			}
			var got int
			if json.Unmarshal(f.ID, &got) != nil {
				continue
			}
			if got == -1 && cancelling {
				if f.Error != nil {
					if stopErr != nil {
						return nil, stopErr
					}
					return nil, ctx.Err()
				}
				var ack struct {
					Accepted bool `json:"accepted"`
				}
				if json.Unmarshal(f.Result, &ack) != nil {
					if stopErr != nil {
						return nil, stopErr
					}
					return nil, ctx.Err()
				}
				p.cancelAccepted = p.cancelAccepted || ack.Accepted
				cancelAck = true
				if saved != nil {
					return finish(*saved)
				}
				continue
			}
			if got != id {
				continue
			}
			if cancelling && !cancelAck {
				saved = &f
				continue
			}
			return finish(f)
		}
	}
}
func (p *dshProcess) close() {
	for _, pending := range p.questions {
		pending.cancel()
	}
	close(p.stop)
	_ = p.in.Close()
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	<-p.done
}

type dshContextUsage struct {
	Used      *int64           `json:"used"`
	Breakdown map[string]int64 `json:"breakdown,omitempty"`
	Window    int64            `json:"window"`
}

func (c *dshContextUsage) valid() bool {
	return c != nil && c.Used != nil && *c.Used >= 0 && *c.Used <= 1<<53-1 && c.Window > 0 && c.Window <= 1<<53-1
}

type dshEvent struct {
	TeamBoard *dshTeamBoard    `json:"aibertaiTeamBoard"`
	Context   *dshContextUsage `json:"aibertaiContext"`
	SessionID string           `json:"sessionId"`
	Event     struct {
		Type string `json:"type"`
		Seq  *int64 `json:"seq"`
		Data struct {
			Turn         int64  `json:"turn"`
			Step         int    `json:"step"`
			InvocationID string `json:"invocationId"`
			Message      struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
			Reason struct {
				Kind string `json:"kind"`
			} `json:"reason"`
			Usage *struct {
				Input      int64  `json:"inputTokens"`
				Output     int64  `json:"outputTokens"`
				CacheRead  *int64 `json:"cacheReadTokens"`
				CacheWrite *int64 `json:"cacheWriteTokens"`
			} `json:"usage"`
		} `json:"data"`
	} `json:"event"`
}

// dshSDKOptions lets the goal driver observe ordered SDK facts and use the
// command-aware query path. Budget enforcement remains at the gateway boundary.
type dshSDKOptions struct {
	Query          bool
	OperationID    string
	SkipOperation  bool
	LeaseControl   bool
	Observe        func(string, json.RawMessage) error
	OnCommand      func(string) error
	VerifyCommand  func(context.Context, string, string, dshConnectorHistoryFetch) error
	InspectSession func(context.Context, *dshProcess) error
	AfterReconcile func(context.Context, *dshProcess) error
	CheckStopped   func(context.Context, *dshProcess, *dshLocalGateway) error
	CheckCancelled func(context.Context, *dshProcess, *dshLocalGateway, *dshLocalRecord, int64) error
}
type dshSDKOptionsKey struct{}

type dshHostTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
	Title       string          `json:"-"`
	Kind        string          `json:"-"`
	Category    string          `json:"-"`
	ResourceID  string          `json:"-"`
}

type dshHostToolBridge struct {
	Tools  []dshHostTool
	Handle func(json.RawMessage) (any, error)
}

type dshExpertInstructionsKey struct{}
type dshHostToolBridgesKey struct{}
type dshSideEffectRootKey struct{}

func runDSHTurn(ctx context.Context, cfg *dshRuntimeConfig, session acpSession, turn *acpTurn, resume bool, project func(map[string]any) error) error {
	return runDSHSession(ctx, cfg, session, turn, resume, false, project)
}

// Reconcile does not enqueue input or clear the platform turn's failure state.
func reconcileDSHSession(ctx context.Context, cfg *dshRuntimeConfig, session acpSession) error {
	return runDSHSession(ctx, cfg, session, &acpTurn{}, true, true, nil)
}

func runDSHSession(ctx context.Context, cfg *dshRuntimeConfig, session acpSession, turn *acpTurn, resume, reconcileOnly bool, project func(map[string]any) error) error {
	for {
		err := runDSHSessionWithProfile(ctx, cfg, session, turn, resume, reconcileOnly, project)
		if !errors.Is(err, errDSHSharedProfileChanged) {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func runDSHSessionWithProfile(ctx context.Context, cfg *dshRuntimeConfig, session acpSession, turn *acpTurn, resume, reconcileOnly bool, project func(map[string]any) error) (resultErr error) {
	opts, _ := ctx.Value(dshSDKOptionsKey{}).(dshSDKOptions)
	boundary, _ := ctx.Value(dshBoundaryStateKey{}).(*dshBoundaryState)
	if boundary == nil {
		boundary = &dshBoundaryState{}
		ctx = context.WithValue(ctx, dshBoundaryStateKey{}, boundary)
	}
	if cfg == nil {
		return errors.New("dsh_runtime_unavailable")
	}
	cancelConfirmed := false
	denialConfirmed, terminalConfirmed := false, false
	var pendingModelFailure error
	operationID := turn.Operation
	if opts.OperationID != "" {
		operationID = opts.OperationID
	}
	leaseOperationID := operationID
	if cfg.LeasePool != nil && opts.LeaseControl {
		var err error
		leaseOperationID, err = newUUID()
		if err != nil {
			return err
		}
	}
	leased := cfg.LeasePool != nil && !reconcileOnly && ((!opts.SkipOperation && validUUID(operationID)) || (opts.LeaseControl && validUUID(leaseOperationID)))
	disabled, bound := ctx.Value(dshDisabledPluginsKey{}).([]string)
	if !bound && cfg.ResolveDisabledPlugins != nil && session.ExpertID != "" && session.ExpertID != designExpertID {
		var err error
		disabled, err = cfg.ResolveDisabledPlugins(ctx, session.ExpertID)
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, dshDisabledPluginsKey{}, disabled)
	}
	var slot *dshLeaseSlot
	var sharedProfile dshSharedProfile
	checkProfile := leased && cfg.SharedPluginHome != ""
	if cfg.LeasePool != nil {
		var err error
		slot, err = cfg.LeasePool.acquire(ctx, session.DSHSessionID)
		if err != nil {
			return err
		}
		defer func() { <-slot.gate }()
		matchingProfile := slot.cwd == session.CWD && slot.model == session.Model && slot.standardMode == (session.ExpertID == "") && slices.Equal(slot.disabledPlugins, disabled)
		if checkProfile {
			sharedProfile, err = readDSHSharedProfile(ctx, cfg.SharedPluginHome)
			if err != nil {
				return err
			}
			if slot.process != nil && slot.process.sharedProfile != sharedProfile {
				matchingProfile = false
			}
		}
		reuseReconcile := reconcileOnly && slot.process != nil && matchingProfile
		if slot.process != nil && ((!leased && !reuseReconcile) || !matchingProfile) {
			slot.discard()
		}
		if reuseReconcile {
			defer func() {
				p := slot.process
				if p != nil {
					p.notification = nil
				}
				if resultErr != nil {
					slot.discard()
				}
			}()
		}
	}
	if leased {
		if cfg.ValidateLease == nil {
			return errors.New("dsh_lease_validation_unavailable")
		}
		selected, validateErr := cfg.ValidateLease(ctx, session)
		if validateErr != nil || selected.ID != session.Model {
			if slot.process != nil {
				slot.discard()
			}
			if validateErr != nil {
				return validateErr
			}
			return errors.New("invalid_dsh_gateway_model")
		}
		if slot.process != nil && (slot.local == nil || slot.local.selected != selected) {
			slot.discard()
		}
	}
	var provider, model string
	var local *dshLocalGateway
	var p *dshProcess
	var err error
	fresh := slot == nil || slot.process == nil
	if fresh {
		processCtx, stopProcess := context.WithCancel(context.WithoutCancel(ctx))
		var cmd *exec.Cmd
		var cleanup func()
		if cfg.Prepare != nil {
			cmd, provider, model, local, cleanup, err = cfg.Prepare(processCtx, session)
		} else {
			if cfg.Command == nil || cfg.Provider == nil {
				stopProcess()
				return errors.New("dsh_runtime_unavailable")
			}
			provider, model, err = cfg.Provider(ctx, session)
			if err == nil {
				cmd, err = cfg.Command(processCtx, session)
			}
		}
		if err != nil {
			stopProcess()
			return err
		}
		p, err = startDSH(processCtx, cmd)
		if err != nil {
			if cleanup != nil {
				cleanup()
			}
			stopProcess()
			return err
		}
		if leased {
			slot.process, slot.local, slot.cleanup, slot.stop = p, local, cleanup, stopProcess
			slot.cwd, slot.model = session.CWD, session.Model
			slot.standardMode = session.ExpertID == ""
			slot.disabledPlugins = slices.Clone(disabled)
		} else {
			defer stopProcess()
			if cleanup != nil {
				defer cleanup()
			}
			defer p.close()
		}
	} else {
		p, local = slot.process, slot.local
	}
	if leased {
		defer func() {
			keep := false
			if resultErr == nil || cancelConfirmed || terminalConfirmed || (turn.DSHHasImages && turn.dshImageAdmission == dshImageNotSent && ctx.Err() == nil) {
				releaseCtx := ctx
				if cancelConfirmed || resultErr == nil && ctx.Err() != nil {
					var stop context.CancelFunc
					releaseCtx, stop = context.WithTimeout(context.Background(), 3*time.Second)
					defer stop()
				}
				_, releaseErr := p.call(releaseCtx, "aibertai/turn-release", map[string]string{"sessionId": session.DSHSessionID, "operationId": leaseOperationID}, nil)
				if releaseErr == nil {
					keep = true
				} else {
					resultErr = errors.Join(resultErr, releaseErr)
				}
			}
			if keep && local != nil {
				local.mu.Lock()
				local.admitGoalInvocation = nil
				local.mu.Unlock()
			}
			p.stopQuestionState()
			p.approval, p.notification, p.hostRequest, p.hostTool, p.afterEvent, p.activity, p.question = nil, nil, nil, nil, nil, nil, nil
			p.cancelSession, p.cancelAccepted = "", false
			p.leaseSession, p.leaseOperation = "", ""
			if !keep || cfg.LeasePool.isClosed() {
				slot.discard()
			}
		}()
	}
	if local != nil {
		local.mu.Lock()
		if !reconcileOnly {
			local.operationID = operationID
		}
		local.mu.Unlock()
		p.notification = func(_ string, raw json.RawMessage) error {
			var receipt struct {
				SessionID    string `json:"sessionId"`
				InvocationID string `json:"invocationId"`
				State        string `json:"state"`
				Seq          int64  `json:"seq"`
			}
			if json.Unmarshal(raw, &receipt) != nil || receipt.SessionID != session.DSHSessionID {
				return errors.New("dsh_invalid_settlement")
			}
			if receipt.State == "failed_with_usage" {
				var failure dshFailedCompactionReceipt
				if json.Unmarshal(raw, &failure) != nil {
					return errors.New("dsh_invalid_settlement")
				}
				return local.acceptFailedCompaction(failure, p.failedCompactionUsage)
			}
			if local.deniedGoalSettlement(receipt.SessionID, receipt.InvocationID, receipt.State, receipt.Seq, true) {
				denialConfirmed = true
				return nil
			}
			err := local.acceptModelSettlement(ctx, session.ID, receipt.InvocationID, receipt.State, receipt.Seq)
			if !reconcileOnly && err != nil && (err.Error() == "dsh_model_outcome_unknown" || err.Error() == "dsh_model_response_interrupted") {
				// The SDK still needs to persist the terminal turn. The journal
				// remains closed to further model dispatch until proof is available.
				pendingModelFailure = err
				return nil
			}
			return err
		}
	}

	initialize := map[string]any{"cwd": session.CWD, "provider": provider, "model": model, "approvalRequests": true, "aibertaiEventFragments": true}
	if question, ok := ctx.Value(dshQuestionKey{}).(dshQuestionHandler); ok && question != nil && !reconcileOnly {
		p.question = question
		initialize["capabilities"] = map[string]bool{"userQuestions": true}
	} else {
		p.question = nil
	}
	// An SDK lease outlives the turn which creates it. Advertise the protocol
	// once; an individual turn still needs a current authorized handler.
	if leased {
		initialize["capabilities"] = map[string]bool{"userQuestions": true}
	}
	binding := map[string]any{"sessionId": session.DSHSessionID, "operationId": leaseOperationID, "workspaceBridge": false, "hostTools": []dshHostTool{}, "expertInstructions": ""}
	if leased {
		initialize["aibertaiLeaseSessionId"] = session.DSHSessionID
	} else if opts.InspectSession != nil || opts.CheckStopped != nil {
		initialize["aibertaiHostSessionId"] = session.DSHSessionID
	}
	instructions, _ := ctx.Value(dshExpertInstructionsKey{}).(string)
	const recoveryGuidance = "When a team member times out, check its current status. If it fails or reaches its output limit, check the existing files and shared task results before deciding what remains to validate, complete or delegate. Respect an employee's explicit stop and await their direction before restarting that work."
	if session.AgentTeamEnabled != nil && *session.AgentTeamEnabled && (session.AgentTeamAvailable == nil || *session.AgentTeamAvailable) && len(instructions)+2+len(recoveryGuidance) <= 256<<10 {
		instructions = strings.TrimSpace(instructions + "\n\n" + recoveryGuidance)
	}
	if instructions != "" && !reconcileOnly {
		if leased {
			binding["expertInstructions"] = instructions
		} else {
			initialize["aibertaiHostSessionId"] = session.DSHSessionID
			initialize["aibertaiExpertInstructions"] = instructions
		}
	}
	if skills, ok := ctx.Value(dshExpertSkillsKey{}).([]dshExpertSkill); ok && len(skills) > 0 && !reconcileOnly {
		if leased {
			binding["expertSkills"] = skills
		} else {
			initialize["aibertaiHostSessionId"] = session.DSHSessionID
			initialize["aibertaiExpertSkills"] = skills
		}
	}
	hostToolViews := map[string]dshHostTool{}
	if bridge, ok := ctx.Value(dshWorkspaceBridgeKey{}).(func(json.RawMessage) (any, error)); ok && !reconcileOnly {
		if !validUUID(session.DSHSessionID) || !validUUID(turn.Operation) {
			return errors.New("invalid_dsh_workspace_identity")
		}
		p.hostRequest = func(raw json.RawMessage) (any, error) {
			if leased && !dshLeaseRequestMatches(raw, session.DSHSessionID, leaseOperationID) {
				return nil, errors.New("dsh_lease_identity_mismatch")
			}
			return bridge(raw)
		}
		if leased {
			binding["workspaceBridge"] = true
			binding["workspaceProofOperationId"] = turn.Operation
		} else {
			initialize["aibertaiWorkspaceBridge"] = true
			initialize["aibertaiWorkspaceSessionId"] = session.DSHSessionID
			initialize["aibertaiWorkspaceOperationId"] = turn.Operation
		}
	}
	if bridges, ok := ctx.Value(dshHostToolBridgesKey{}).([]dshHostToolBridge); ok && !reconcileOnly {
		var definitions []dshHostTool
		handlers := make(map[string]func(json.RawMessage) (any, error))
		for _, bridge := range bridges {
			if bridge.Handle == nil && len(bridge.Tools) != 0 {
				return errors.New("invalid_dsh_host_tools")
			}
			for _, tool := range bridge.Tools {
				if tool.Name == "" || handlers[tool.Name] != nil || !json.Valid(tool.InputSchema) {
					return errors.New("invalid_dsh_host_tools")
				}
				handlers[tool.Name] = bridge.Handle
				hostToolViews[tool.Name] = tool
				definitions = append(definitions, tool)
			}
		}
		if len(definitions) != 0 {
			if leased {
				binding["hostTools"] = definitions
			} else {
				initialize["aibertaiHostSessionId"] = session.DSHSessionID
				initialize["aibertaiHostTools"] = definitions
			}
			p.hostTool = func(raw json.RawMessage) (any, error) {
				var request struct {
					SessionID string `json:"sessionId"`
					CallID    string `json:"callId"`
					Name      string `json:"name"`
				}
				if json.Unmarshal(raw, &request) != nil || request.SessionID != session.DSHSessionID || request.CallID == "" || handlers[request.Name] == nil || leased && !dshLeaseRequestMatches(raw, session.DSHSessionID, leaseOperationID) {
					return nil, errors.New("dsh_host_tool_invalid")
				}
				return handlers[request.Name](raw)
			}
		}
	}
	if fresh {
		initResult, initErr := p.call(ctx, "initialize", initialize, nil)
		if initErr != nil {
			return initErr
		}
		if checkProfile {
			// initialize waits for the SDK to compose its actual profile. Its first
			// boot may create defaults, or another session may publish new packages.
			// Retry before resume, binding tools, or dispatching a model request.
			current, err := readDSHSharedProfile(ctx, cfg.SharedPluginHome)
			if err != nil {
				return err
			}
			if current != sharedProfile {
				return errDSHSharedProfileChanged
			}
			p.sharedProfile = current
		}
		p.workspaceBoundary = workspaceBoundaryCapable(initResult)
		var caps struct {
			Failed *struct {
				Version int `json:"version"`
			} `json:"aibertaiFailedCompactionUsage"`
		}
		if json.Unmarshal(initResult, &caps) == nil && caps.Failed != nil && caps.Failed.Version == 1 {
			p.failedCompactionUsage = true
		}
	}
	if leased {
		slot.observerOwner.Store(p)
	}
	boundary.capable = p.workspaceBoundary
	boundary.history = p.historyCall
	boundary.proof = func(ctx context.Context, params map[string]any) (json.RawMessage, error) {
		return p.boundaryCall(ctx, "aibertai/workspace-proof", params)
	}
	if resume && fresh {
		if _, err = p.call(ctx, "session/resume", map[string]string{"sessionId": session.DSHSessionID}, nil); err != nil {
			return err
		}
	}
	if opts.InspectSession != nil {
		// Inspect the restored SDK root under its per-session gate, without
		// binding a turn, reconciling accounting, or sending model input.
		return opts.InspectSession(ctx, p)
	}
	p.approval = func(raw json.RawMessage) (string, error) {
		outcome := "rejected"
		if leased && !dshLeaseRequestMatches(raw, session.DSHSessionID, leaseOperationID) {
			return outcome, nil
		}
		if bridge, ok := ctx.Value(dshNativeApprovalKey{}).(dshNativeApproval); ok {
			if err := dshNativeApprovalHistory(ctx, session.DSHSessionID, raw, turn, p.historyCall); err != nil {
				return outcome, nil
			}
			var err error
			outcome, err = bridge(raw, turn)
			if project != nil {
				if saveErr := project(nil); saveErr != nil {
					return "rejected", saveErr
				}
			}
			return outcome, err
		}
		turn.Permissions.Asked++
		turn.Permissions.Denied++
		if project != nil {
			return outcome, project(nil)
		}
		return outcome, nil
	}

	if local != nil {
		if reconcileOnly {
			if err := local.reloadPending(); err != nil {
				return err
			}
			if opts.CheckStopped != nil {
				return opts.CheckStopped(ctx, p, local)
			}
		}
		local.mu.Lock()
		pending := local.pending
		local.mu.Unlock()
		if pending != nil {
			if !resume {
				return errors.New("dsh_reconciliation_required")
			}
			raw, err := p.call(ctx, "aibertai/reconcile", map[string]string{"sessionId": session.DSHSessionID, "invocationId": pending.CallID}, nil)
			if err != nil {
				return errors.Join(errors.New("dsh_reconciliation_required"), err)
			}
			var receipt struct {
				State string `json:"state"`
				Seq   int64  `json:"seq"`
			}
			if json.Unmarshal(raw, &receipt) != nil {
				return errors.New("dsh_reconciliation_required")
			}
			if receipt.State == "failed_with_usage" {
				var failure dshFailedCompactionReceipt
				if json.Unmarshal(raw, &failure) != nil {
					return errors.New("dsh_reconciliation_required")
				}
				failure.SessionID = session.DSHSessionID
				failure.InvocationID = pending.CallID
				if err := local.validateFailedCompaction(failure, p.failedCompactionUsage); err != nil {
					return errors.Join(errors.New("dsh_reconciliation_required"), err)
				}
			}
			if opts.Observe != nil {
				var observed map[string]json.RawMessage
				if json.Unmarshal(raw, &observed) != nil || observed == nil {
					return errors.New("dsh_reconciliation_required")
				}
				observed["sessionId"], _ = json.Marshal(session.DSHSessionID)
				observed["invocationId"], _ = json.Marshal(pending.CallID)
				payload, err := json.Marshal(observed)
				if err != nil {
					return errors.Join(errors.New("dsh_reconciliation_required"), err)
				}
				if err := opts.Observe("aibertai/model-settled", payload); err != nil {
					return errors.Join(errors.New("dsh_reconciliation_required"), err)
				}
			}
			if receipt.State == "failed_with_usage" {
				var failure dshFailedCompactionReceipt
				if json.Unmarshal(raw, &failure) != nil {
					return errors.New("dsh_reconciliation_required")
				}
				failure.SessionID = session.DSHSessionID
				failure.InvocationID = pending.CallID
				if err := local.acceptFailedCompaction(failure, p.failedCompactionUsage); err != nil {
					return errors.Join(errors.New("dsh_reconciliation_required"), err)
				}
			} else if reconcileOnly && receipt.State == "failed" && opts.CheckCancelled != nil {
				if err := opts.CheckCancelled(ctx, p, local, pending, receipt.Seq); err != nil {
					return errors.Join(errors.New("dsh_reconciliation_required"), err)
				}
			} else if err := local.acceptModelSettlement(ctx, session.ID, pending.CallID, receipt.State, receipt.Seq); err != nil {
				return errors.Join(errors.New("dsh_reconciliation_required"), err)
			}
		} else if reconcileOnly && opts.CheckCancelled != nil {
			if err := opts.CheckCancelled(ctx, p, local, nil, -1); err != nil {
				return errors.Join(errors.New("dsh_reconciliation_required"), err)
			}
		}
	}
	if reconcileOnly {
		if local == nil {
			return errors.New("dsh_gateway_unavailable")
		}
		if opts.AfterReconcile != nil {
			return opts.AfterReconcile(ctx, p)
		}
		return nil
	}
	if leased {
		if local != nil {
			if err = local.resetTurnState(); err != nil {
				return err
			}
			if cfg.BindLease == nil {
				return errors.New("dsh_lease_authority_unavailable")
			}
			if err = cfg.BindLease(ctx, session, leaseOperationID, local); err != nil {
				return err
			}
		}
		if _, err = p.call(ctx, "aibertai/turn-bind", binding, nil); err != nil {
			return err
		}
		p.leaseSession, p.leaseOperation = session.DSHSessionID, leaseOperationID
	}
	var queryCompletions = map[string]struct {
		FinalResponse string `json:"finalResponse"`
		FinishReason  string `json:"finishReason"`
	}{}
	var streamID string
	var streamRevision int64
	var streamText string
	var streamSettled, streamAwaitCommit bool
	var answer, reason, lastAttempt string
	var input, output int64
	var hasUsage bool
	var tokens turnTokenFacts
	if local != nil && !reconcileOnly {
		p.team.begin(session.DSHSessionID, p.leaseOperation)
		defer func() { p.teamSettled = nil }()
		p.teamSettled = func(receipt dshFailedCompactionReceipt) error {
			if receipt.MemberInterrupt != nil {
				return local.acceptMemberInterrupt(ctx, operationID, receipt)
			}
			if local.deniedGoalSettlement(session.ID, receipt.InvocationID, receipt.State, receipt.Seq, true) {
				denialConfirmed = true
				return nil
			}
			if receipt.State == "failed" && receipt.Seq > 0 && validUUID(receipt.InvocationID) {
				local.mu.Lock()
				owned := local.pending != nil && local.pending.OperationID == operationID && local.pending.CallID == receipt.InvocationID && (local.pending.State == "rejected" || local.pending.Status >= 400 || local.pending.Finish == "content_filter")
				local.mu.Unlock()
				if owned {
					return local.acceptModelSettlement(ctx, session.ID, receipt.InvocationID, receipt.State, receipt.Seq)
				}
			}
			if err := local.verifyTeamSettlement(receipt, operationID); err != nil {
				return err
			}
			in, out := *receipt.Usage.Input, *receipt.Usage.Output
			if input > math.MaxInt64-in || output > math.MaxInt64-out {
				return errors.New("dsh_invalid_settlement")
			}
			if opts.Observe != nil {
				// Goal budgets belong to the owning root, including billed child calls.
				receipt.SessionID = session.DSHSessionID
				payload, err := json.Marshal(receipt)
				if err != nil {
					return err
				}
				if err := opts.Observe("aibertai/model-settled", payload); err != nil {
					return err
				}
			}
			input, output, hasUsage = input+in, output+out, true
			tokens.member(in, out, receipt.Usage.CacheRead, receipt.Usage.CacheWrite)
			totalIn, totalOut := input, output
			turn.Usage, turn.Tokens = modelUsage{Input: &totalIn, Output: &totalOut}, tokens
			if project != nil {
				if err := project(nil); err != nil {
					return err
				}
			}
			return local.Ack(receipt.InvocationID)
		}
	}
	type compactUsage struct {
		seq                   int64
		input, output         int64
		cacheRead, cacheWrite *int64
	}
	compactPending := map[string]compactUsage{}
	failedCompact := map[string]dshFailedCompactionReceipt{}
	compactSeen := map[string]bool{}
	pendingGoalThoughts := map[string]string{}
	defer func() {
		for _, id := range pendingGoalThoughts {
			if id != "" && project != nil {
				_ = project(map[string]any{"sessionUpdate": "agent_message_reset", "streamAttemptId": id})
			}
		}
	}()
	p.activity = func(method string, raw json.RawMessage) error {
		if method == "aibertai/model-settled" && local != nil {
			var receipt struct {
				SessionID    string `json:"sessionId"`
				InvocationID string `json:"invocationId"`
				State        string `json:"state"`
				Seq          int64  `json:"seq"`
			}
			if json.Unmarshal(raw, &receipt) == nil && local.deniedGoalSettlement(receipt.SessionID, receipt.InvocationID, receipt.State, receipt.Seq, false) {
				return nil
			}
		}
		var sessionFrame struct {
			SessionID     string `json:"sessionId"`
			CommandID     string `json:"commandId"`
			FinalResponse string `json:"finalResponse"`
			FinishReason  string `json:"finishReason"`
			Frame         struct {
				Type      string `json:"type"`
				AttemptID string `json:"attemptId"`
				Revision  int64  `json:"revision"`
				Index     int    `json:"index"`
				Chunk     struct {
					Type  string `json:"type"`
					Index int    `json:"index"`
					Text  string `json:"text"`
				} `json:"chunk"`
			} `json:"frame"`
		}
		if json.Unmarshal(raw, &sessionFrame) != nil || sessionFrame.SessionID != session.DSHSessionID {
			return nil
		}
		if method == "session.query.completed" {
			queryCompletions[sessionFrame.CommandID] = struct {
				FinalResponse string `json:"finalResponse"`
				FinishReason  string `json:"finishReason"`
			}{sessionFrame.FinalResponse, sessionFrame.FinishReason}
		}
		if method == "aibertai/assistant-stream" {
			// Stream frames are provisional. Once the turn is cancelled, do not
			// persist a frame that was queued before cancellation but read after it.
			// Durable assistant messages and settlement receipts are handled below.
			if ctx.Err() != nil {
				return nil
			}
			f := sessionFrame.Frame
			if f.AttemptID == "" || f.Revision <= 0 {
				return errors.New("dsh_invalid_assistant_stream")
			}
			switch f.Type {
			case "start":
				if f.Revision <= streamRevision || (!opts.Query && reason != "") {
					break
				}
				streamID, streamRevision, streamText, streamSettled, streamAwaitCommit = f.AttemptID, f.Revision, "", false, true
				turn.Answer = ""
				turn.DSHParts = nil
				turn.DSHThoughtProof = nil
				turn.dshCurrentInvocation = ""
				turn.DSHStreamed = false
				if project != nil {
					if err := project(map[string]any{"sessionUpdate": "agent_message_start", "streamAttemptId": streamID, "streamRevision": f.Revision}); err != nil {
						return err
					}
				}
			case "chunk":
				if f.AttemptID != streamID || streamSettled || f.Revision <= streamRevision {
					break
				}
				streamRevision = f.Revision
				if f.Chunk.Type == "reasoning-delta" {
					if project != nil {
						if err := project(map[string]any{"sessionUpdate": "agent_thought_chunk", "content": map[string]string{"type": "text", "text": f.Chunk.Text}, "streamAttemptId": streamID, "streamRevision": f.Revision, "streamIndex": f.Index, "streamBlockIndex": f.Chunk.Index}); err != nil {
							return err
						}
					}
					break
				}
				if f.Chunk.Type != "text-delta" {
					break
				}
				streamText += f.Chunk.Text
				turn.Answer = streamText
				if project != nil {
					if err := project(map[string]any{"sessionUpdate": "agent_message_chunk", "content": map[string]string{"type": "text", "text": f.Chunk.Text}, "streamAttemptId": streamID, "streamRevision": f.Revision, "streamIndex": f.Index, "streamBlockIndex": f.Chunk.Index}); err != nil {
						return err
					}
				}
			case "end":
				if f.AttemptID == streamID && f.Revision > streamRevision {
					streamRevision = f.Revision
					streamSettled = true
				}
			default:
				return errors.New("dsh_invalid_assistant_stream")
			}
		}
		if method == "aibertai/model-settled" {
			var receipt struct {
				SessionID, InvocationID, State string
				Seq                            int64
				UsageSeq                       *int64
				Usage                          *struct {
					Input  *int64 `json:"inputTokens"`
					Output *int64 `json:"outputTokens"`
				}
			}
			if json.Unmarshal(raw, &receipt) != nil {
				return errors.New("dsh_compaction_usage_invalid")
			}
			if receipt.State == "failed_with_usage" {
				var failure dshFailedCompactionReceipt
				if json.Unmarshal(raw, &failure) != nil || !p.failedCompactionUsage {
					return errors.New("dsh_compaction_usage_invalid")
				}
				if local == nil || local.validateFailedCompaction(failure, p.failedCompactionUsage) != nil {
					return errors.New("dsh_compaction_usage_invalid")
				}
				evidence, ok := failedCompact[failure.InvocationID]
				if !ok || failure.SessionID != session.DSHSessionID || failure.OperationID != operationID || failure.UsageSeq == nil || evidence.UsageSeq == nil || *failure.UsageSeq != *evidence.UsageSeq || failure.Seq <= *failure.UsageSeq || failure.Usage == nil || evidence.Usage == nil || failure.Usage.Input == nil || failure.Usage.Output == nil || evidence.Usage.Input == nil || evidence.Usage.Output == nil || *failure.Usage.Input != *evidence.Usage.Input || *failure.Usage.Output != *evidence.Usage.Output || input > math.MaxInt64-*failure.Usage.Input || output > math.MaxInt64-*failure.Usage.Output {
					return errors.New("dsh_compaction_usage_invalid")
				}
				input += *failure.Usage.Input
				output += *failure.Usage.Output
				hasUsage = true
				tokens.cache(failure.Usage.CacheRead, failure.Usage.CacheWrite)
				delete(failedCompact, failure.InvocationID)
				in, out := input, output
				turn.Usage, turn.Tokens = modelUsage{Input: &in, Output: &out}, tokens
				if project != nil {
					if err := project(nil); err != nil {
						return err
					}
				}
			}
			if pending, ok := compactPending[receipt.InvocationID]; ok {
				if receipt.SessionID != session.DSHSessionID || receipt.State != "committed" || receipt.UsageSeq == nil || *receipt.UsageSeq != pending.seq || receipt.Seq != pending.seq+2 || receipt.Usage == nil || receipt.Usage.Input == nil || receipt.Usage.Output == nil || *receipt.Usage.Input != pending.input || *receipt.Usage.Output != pending.output || input > math.MaxInt64-pending.input || output > math.MaxInt64-pending.output {
					return errors.New("dsh_compaction_usage_invalid")
				}
				input += pending.input
				output += pending.output
				hasUsage = true
				tokens.cache(pending.cacheRead, pending.cacheWrite)
				delete(compactPending, receipt.InvocationID)
				delete(failedCompact, receipt.InvocationID)
				in, out := input, output
				turn.Usage, turn.Tokens = modelUsage{Input: &in, Output: &out}, tokens
				if project != nil {
					if err := project(nil); err != nil {
						return err
					}
				}
			}
		}
		if opts.Observe != nil {
			if err := opts.Observe(method, raw); err != nil {
				return err
			}
		}
		if method == "aibertai/model-settled" && len(pendingGoalThoughts) != 0 {
			var receipt struct {
				InvocationID string `json:"invocationId"`
			}
			if json.Unmarshal(raw, &receipt) == nil {
				if id, ok := pendingGoalThoughts[receipt.InvocationID]; ok {
					verified := turn.DSHThoughtProof != nil && turn.DSHThoughtProof.InvocationID == receipt.InvocationID
					for _, run := range turn.ToolRuns {
						verified = verified || run.DSHThoughtProof != nil && run.DSHThoughtProof.InvocationID == receipt.InvocationID
					}
					if !verified && id != "" && project != nil {
						if err := project(map[string]any{"sessionUpdate": "agent_message_reset", "streamAttemptId": id}); err != nil {
							return err
						}
					}
					delete(pendingGoalThoughts, receipt.InvocationID)
				}
			}
		}
		return nil
	}
	var deliveryTurn int64
	toolStarted := map[string]time.Time{}
	var previousTurnAnswer string
	var continuationStart bool
	event := func(raw json.RawMessage) error {
		var e dshEvent
		if json.Unmarshal(raw, &e) != nil || e.SessionID != session.DSHSessionID {
			return nil
		}
		if e.Event.Type == "todo/write" || e.Event.Type == "turn/start" {
			var frame struct{ Event dshConnectorHistoryEvent }
			if json.Unmarshal(raw, &frame) == nil && turn.applyDSHTodo(frame.Event) && project != nil {
				if err := project(turn.DSHTodos.update()); err != nil {
					return err
				}
			}
		}
		if e.Event.Type == "compaction/start" || e.Event.Type == "compaction/end" {
			var frame struct{ Event dshConnectorHistoryEvent }
			if json.Unmarshal(raw, &frame) == nil {
				if notice := turn.applyDSHCompaction(frame.Event, deliveryTurn); notice != nil && project != nil {
					if err := project(notice.update()); err != nil {
						return err
					}
				}
			}
		}
		if (e.Event.Type == "team/task" || e.Event.Type == "team/mode") && e.TeamBoard != nil && e.TeamBoard.SessionID == session.DSHSessionID {
			turn.DSHTeamBoard = e.TeamBoard
			if project != nil {
				if err := project(map[string]any{"sessionUpdate": "dsh_team_board", "board": e.TeamBoard}); err != nil {
					return err
				}
			}
		}
		if (e.Event.Type == "assistant/message" || e.Event.Type == "compaction/end") && e.Context.valid() {
			turn.DSHContext = e.Context
			if project != nil {
				if err := project(map[string]any{"sessionUpdate": "usage_update", "used": *e.Context.Used, "size": e.Context.Window, "_meta": map[string]any{"dshContext": e.Context}}); err != nil {
					return err
				}
			}
		}
		switch e.Event.Type {
		case "aibertai/compaction-usage":
			var frame struct {
				Event struct {
					Seq  *int64 `json:"seq"`
					Data struct {
						InvocationID string         `json:"invocationId"`
						OperationID  string         `json:"operationId"`
						Usage        *dshTokenUsage `json:"usage"`
					} `json:"data"`
				} `json:"event"`
			}
			if json.Unmarshal(raw, &frame) != nil || frame.Event.Seq == nil || *frame.Event.Seq < 0 || !validUUID(frame.Event.Data.InvocationID) || frame.Event.Data.OperationID != operationID || frame.Event.Data.Usage == nil || frame.Event.Data.Usage.Input == nil || frame.Event.Data.Usage.Output == nil || *frame.Event.Data.Usage.Input < 0 || *frame.Event.Data.Usage.Output < 0 {
				return errors.New("dsh_compaction_usage_invalid")
			}
			if _, exists := failedCompact[frame.Event.Data.InvocationID]; exists {
				return errors.New("dsh_compaction_usage_invalid")
			}
			v := dshFailedCompactionReceipt{InvocationID: frame.Event.Data.InvocationID, OperationID: frame.Event.Data.OperationID, UsageSeq: frame.Event.Seq}
			v.Usage = frame.Event.Data.Usage
			failedCompact[v.InvocationID] = v
		case "compaction/summary":
			var summary struct {
				Event struct {
					Seq  *int64 `json:"seq"`
					Data struct {
						LLMStreamCall bool           `json:"llmStreamCall"`
						InvocationID  string         `json:"invocationId"`
						Usage         *dshTokenUsage `json:"usage"`
					} `json:"data"`
				} `json:"event"`
			}
			if json.Unmarshal(raw, &summary) != nil {
				return errors.New("dsh_compaction_usage_invalid")
			}
			data := summary.Event.Data
			if !data.LLMStreamCall {
				break
			}
			if summary.Event.Seq == nil || *summary.Event.Seq < 0 || !validUUID(data.InvocationID) || data.Usage == nil || data.Usage.Input == nil || data.Usage.Output == nil || *data.Usage.Input < 0 || *data.Usage.Output < 0 {
				return errors.New("dsh_compaction_usage_invalid")
			}
			if compactSeen[data.InvocationID] {
				return errors.New("dsh_compaction_usage_invalid")
			}
			compactSeen[data.InvocationID] = true
			compactPending[data.InvocationID] = compactUsage{seq: *summary.Event.Seq, input: *data.Usage.Input, output: *data.Usage.Output, cacheRead: data.Usage.CacheRead, cacheWrite: data.Usage.CacheWrite}
		case "assistant/message":
			lastAttempt = e.Event.Data.InvocationID
			var thoughtIdentity struct {
				Event struct {
					Data struct {
						InvocationID string `json:"invocationId"`
					} `json:"data"`
				} `json:"event"`
			}
			_ = json.Unmarshal(raw, &thoughtIdentity)
			turn.dshCurrentInvocation = thoughtIdentity.Event.Data.InvocationID
			turn.dshStreamAttempt = ""
			if streamAwaitCommit {
				turn.dshStreamAttempt = streamID
			}
			turn.DSHThoughtProof = nil
			streamSettled = true
			answer = ""
			turn.DSHParts = nil
			hasThought := false
			for _, b := range e.Event.Data.Message.Content {
				hasThought = hasThought || b.Type == "reasoning"
			}
			for _, b := range e.Event.Data.Message.Content {
				if b.Type == "text" {
					answer += b.Text
				}
				if hasThought && b.Type == "text" {
					turn.DSHParts = append(turn.DSHParts, dshMessagePart{Type: "text", End: len(answer)})
				} else if b.Type == "reasoning" {
					turn.DSHParts = append(turn.DSHParts, dshMessagePart{Type: "thought", Text: b.Text})
				}
			}
			if e.Event.Data.Usage != nil {
				hasUsage = true
				input += e.Event.Data.Usage.Input
				output += e.Event.Data.Usage.Output
				tokens.cache(e.Event.Data.Usage.CacheRead, e.Event.Data.Usage.CacheWrite)
			}
			turn.Answer = answer
			turn.DSHStreamed = streamAwaitCommit && streamID != ""
			streamAwaitCommit = false
			if hasUsage {
				in, out := input, output
				turn.Usage, turn.Tokens = modelUsage{Input: &in, Output: &out}, tokens
			}
			if project != nil {
				meta := map[string]any{"dshParts": dshRenderedParts(turn.DSHParts, answer)}
				if turn.GoalCheckpoint != nil && hasThought && turn.dshStreamAttempt != "" && validUUID(turn.dshCurrentInvocation) {
					meta["dshProvisional"] = true
					pendingGoalThoughts[turn.dshCurrentInvocation] = streamID
				}
				if turn.dshStreamAttempt == "" && turn.GoalCheckpoint == nil && e.Event.Seq != nil && *e.Event.Seq >= 0 {
					meta["dshMessageSeq"] = *e.Event.Seq
				}
				return project(map[string]any{"sessionUpdate": "agent_message_settled", "content": map[string]string{"type": "text", "text": answer}, "streamAttemptId": turn.dshStreamAttempt, "_meta": meta})
			}
		case "assistant/attempt":
			lastAttempt = e.Event.Data.InvocationID
			if local != nil && ctx.Err() == nil && !opts.Query && turn.GoalCheckpoint == nil && streamAwaitCommit && streamText != "" {
				local.mu.Lock()
				partial := local.pending != nil && local.pending.CallID == lastAttempt && local.pending.OperationID == operationID && local.pending.Status >= 200 && local.pending.Status < 300 && local.pending.State == "dispatched"
				local.mu.Unlock()
				if partial {
					answer, turn.Answer = streamText, streamText
					turn.DSHParts, turn.DSHThoughtProof = nil, nil
					turn.DSHStreamed = true
					streamSettled, streamAwaitCommit = true, false
					if project != nil {
						return project(map[string]any{"sessionUpdate": "agent_message_settled", "content": map[string]string{"type": "text", "text": answer}, "streamAttemptId": streamID})
					}
					break
				}
			}
			turn.dshCurrentInvocation = ""
			turn.DSHThoughtProof = nil
			streamSettled = true
			streamAwaitCommit = false
			turn.Answer = ""
			turn.DSHParts = nil
			turn.DSHStreamed = false
			if project != nil {
				return project(map[string]any{"sessionUpdate": "agent_message_reset", "streamAttemptId": streamID})
			}
		case "user/message", "developer/message":
			if entry, ok := readDSHContextInjection(raw); ok {
				entry.AfterTools = len(turn.ToolRuns)
				if turn.addContextInjection(entry) && project != nil {
					return project(map[string]any{"sessionUpdate": "dsh_context", "entry": entry})
				}
				break
			}
			if notice, ok := readDSHMemberNotice(raw); ok {
				notice.AfterTools = len(turn.ToolRuns)
				notice.WokeParent = continuationStart
				notice.PreviousAnswer = previousTurnAnswer
				previousTurnAnswer = ""
				if turn.addMemberNotice(notice) {
					if notice.PreviousAnswer != "" {
						// The notice now owns the preceding turn's answer in history.
						answer, turn.Answer, turn.DSHParts = "", "", nil
						turn.DSHThoughtProof = nil
					}
					if project != nil {
						return project(map[string]any{"sessionUpdate": "dsh_member_notice", "notice": notice})
					}
				}
			}
		case "deliverables/presented":
			var frame struct {
				Event struct{ Data json.RawMessage }
			}
			if json.Unmarshal(raw, &frame) == nil {
				turn.recordDeliverables(frame.Event.Data, deliveryTurn)
			}
		case "turn/start":
			continuationStart = deliveryTurn != 0
			if continuationStart {
				previousTurnAnswer = answer
			}
			deliveryTurn = e.Event.Data.Turn
			if opts.Query {
				reason = ""
			}
		case "turn/end":
			reason = e.Event.Data.Reason.Kind
		case "step/start":
			continuationStart = continuationStart && e.Event.Data.Step == 1
			if e.Event.Data.Step > 0 && ctx.Err() == nil {
				turn.DSHStep = e.Event.Data.Step
				if project != nil {
					return project(map[string]any{"sessionUpdate": "dsh_step", "step": turn.DSHStep, "startedAt": turn.Started.UnixMilli()})
				}
			}
		case "tool/call", "tool/result":
			previous := len(turn.ToolRuns)
			update, err := projectDSHTool(turn, raw)
			if err != nil {
				return err
			}
			if e.Event.Type == "tool/call" && len(turn.ToolRuns) > previous {
				toolStarted[turn.ToolRuns[len(turn.ToolRuns)-1].DSHCallID] = time.Now()
			}
			if e.Event.Type == "tool/result" {
				for i := range turn.ToolRuns {
					run := &turn.ToolRuns[i]
					if started, ok := toolStarted[run.DSHCallID]; ok && !run.Pending {
						run.Report.DurationMS = time.Since(started).Milliseconds()
						delete(toolStarted, run.DSHCallID)
					}
				}
			}
			if e.Event.Type == "tool/call" && len(turn.ToolRuns) > previous {
				run := &turn.ToolRuns[len(turn.ToolRuns)-1]
				if tool, ok := hostToolViews[run.Report.Name]; ok {
					run.DisplayName, run.DisplayKind = tool.Title, tool.Kind
					if tool.Category != "" {
						run.Report.Category = tool.Category
					}
					if tool.ResourceID != "" {
						run.Report.ResourceID = tool.ResourceID
					}
					if update != nil && tool.Title != "" {
						update["title"] = tool.Title
					}
					if update != nil && tool.Kind != "" {
						update["kind"] = tool.Kind
					}
				}
			}
			if e.Event.Type == "tool/call" && len(turn.ToolRuns) > previous && (answer != "" || len(turn.DSHParts) != 0) {
				turn.ToolRuns[len(turn.ToolRuns)-1].Preamble = answer
				turn.ToolRuns[len(turn.ToolRuns)-1].dshSourceInvocation = turn.dshCurrentInvocation
				turn.ToolRuns[len(turn.ToolRuns)-1].dshStreamAttempt = turn.dshStreamAttempt
				turn.ToolRuns[len(turn.ToolRuns)-1].DSHParts = turn.DSHParts
				turn.ToolRuns[len(turn.ToolRuns)-1].DSHThoughtProof = turn.DSHThoughtProof
				turn.DSHThoughtProof = nil
				turn.dshCurrentInvocation = ""
				answer = ""
				turn.Answer = ""
				turn.DSHParts = nil
				turn.DSHStreamed = false
			}
			if update != nil && project != nil {
				return project(update)
			}
		}
		return nil
	}
	if root, ok := ctx.Value(dshSideEffectRootKey{}).(*os.Root); ok && root != nil {
		p.afterEvent = func(raw json.RawMessage) error {
			var event struct {
				SessionID string `json:"sessionId"`
				Event     struct {
					Type      string              `json:"type"`
					SurfaceOp dshHistorySurfaceOp `json:"surfaceOp"`
					Data      struct {
						CallID  string `json:"callId"`
						Message struct {
							Source struct {
								CallID string `json:"callId"`
							} `json:"source"`
						} `json:"message"`
					} `json:"data"`
				} `json:"event"`
			}
			if json.Unmarshal(raw, &event) != nil || event.SessionID != session.DSHSessionID || event.Event.Type != "tool/result" && event.Event.Type != "aibertai/workspace-result" {
				return nil
			}
			if event.Event.Type == "tool/result" && event.Event.SurfaceOp.Op == "replace" {
				return nil // A context-pruning notification carries no execution receipt.
			}
			call := event.Event.Data.Message.Source.CallID
			if event.Event.Type == "aibertai/workspace-result" {
				call = event.Event.Data.CallID
			}
			if call == "" {
				if event.Event.Type == "aibertai/workspace-result" {
					return nil
				}
				return errors.New("dsh_connector_reconciliation_required")
			}
			claims, err := dshConnectorClaims(root, session.ID)
			if err != nil {
				if event.Event.Type == "aibertai/workspace-result" {
					return nil
				}
				return err
			}
			for _, claim := range claims {
				if claim.State == "reconciled" || claim.State == "user_reviewed" || claim.CallID != call {
					continue
				}
				if claim.Kind == "workspace" && event.Event.Type == "tool/result" {
					return nil
				}
				if claim.Kind != "workspace" && event.Event.Type == "aibertai/workspace-result" {
					return nil
				}
				if claim.Kind == "native" && claim.State == "dispatch_claimed" && claim.OperationID == turn.Operation {
					var result struct {
						Event struct {
							Data struct {
								Message struct {
									Role, ToolCallID string
									IsError          bool
									Source           struct{ Kind, CallID string }
									Content          []struct{ Type, Text string }
								}
							}
						}
					}
					if json.Unmarshal(raw, &result) != nil || result.Event.Data.Message.Role != "tool" || result.Event.Data.Message.ToolCallID != call || result.Event.Data.Message.Source.Kind != "tool" || result.Event.Data.Message.Source.CallID != call || result.Event.Data.Message.IsError || len(result.Event.Data.Message.Content) != 1 || result.Event.Data.Message.Content[0].Type != "text" {
						return errors.New("dsh_connector_outcome_unknown")
					}
					text := result.Event.Data.Message.Content[0].Text
					if strings.Contains(text, "; moved to background job ") || strings.HasPrefix(text, "started background job ") {
						return errors.New("dsh_connector_outcome_unknown")
					}
					if err := dshConnectorRecordResult(root, session.ID, turn.Operation, call, text); err != nil {
						return errors.New("dsh_connector_outcome_unknown")
					}
					claim.State = "result_ready"
				}
				if claim.State != "result_ready" || claim.OperationID != turn.Operation {
					if event.Event.Type == "aibertai/workspace-result" {
						return nil
					}
					return errors.New("dsh_connector_reconciliation_required")
				}
				var run *workspaceToolRun
				for i := range turn.ToolRuns {
					if turn.ToolRuns[i].DSHCallID == call {
						run = &turn.ToolRuns[i]
						break
					}
				}
				if run == nil || run.Call == nil {
					if event.Event.Type == "aibertai/workspace-result" {
						return nil
					}
					return errors.New("dsh_connector_reconciliation_required")
				}
				if claim.Kind == "workspace" {
					historyOperation := turn.Operation
					if session.Goal != nil && session.Goal.Current && session.GoalRun == turn.Operation {
						historyOperation = session.Goal.DSHQueryOp
					}
					if err := dshWorkspaceReconcileBound(ctx, root, session.ID, turn.Operation, historyOperation, call, run.Call.Function.Name, json.RawMessage(run.Call.Function.Arguments), p.historyCall); err != nil && err.Error() != "dsh_workspace_reconciliation_required" {
						return err
					}
					return nil
				}
				return dshConnectorReconcile(root, session.ID, turn.Operation, call, func(sessionID, operationID, callID string) (json.RawMessage, error) {
					if session.Goal != nil && session.Goal.Current && session.GoalRun == operationID {
						operationID = session.Goal.DSHQueryOp
					}
					return dshMatchConnectorHistory(ctx, sessionID, operationID, callID, run.Call.Function.Name, json.RawMessage(run.Call.Function.Arguments), p.historyCall)
				})
			}
			return nil
		}
	}
	modelPrompt := turn.Prompt
	if turn.ModelInput != "" {
		modelPrompt = turn.ModelInput
	}
	params := map[string]any{"sessionId": session.DSHSessionID, "contentBlocks": []map[string]string{{"type": "text", "text": modelPrompt}}}
	if input, ok := ctx.Value(dshImageInputKey{}).(dshImageInput); ok {
		if cfg.Prepare == nil {
			return errors.New("model_vision_unavailable")
		}
		params["contentBlocks"] = input.blocks
	}
	method := "session/prompt"
	if opts.Query {
		method = "session/query"
		p.cancelSession = session.DSHSessionID
	}
	if !opts.SkipOperation && validUUID(operationID) {
		params["aibertaiOperationId"] = operationID
	} else if leased && opts.LeaseControl {
		params["aibertaiLeaseOperationId"] = leaseOperationID
	}
	if turn.DSHMemberID != "" {
		if !leased || opts.Query || !validUUID(operationID) {
			return errors.New("dsh_member_lease_unavailable")
		}
		method = "aibertai/member-prompt"
		params = map[string]any{"sessionId": session.DSHSessionID, "memberId": turn.DSHMemberID, "requestId": operationID, "operationId": operationID, "text": turn.Prompt, "standalone": true}
	}
	if turn.DSHHasImages {
		turn.dshImageAdmission = dshImageOutcomeUnknown
	}
	p.cancelSession = session.DSHSessionID
	receipt, err := p.call(ctx, method, params, event)
	if err != nil {
		if _, ok := ctx.Value(dshImageInputKey{}).(dshImageInput); ok && strings.Contains(err.Error(), "Unsupported or malformed image data.") {
			turn.dshImageAdmission = dshImageNotSent
			return errors.New("attachment_image_invalid")
		}
		return err
	}
	if turn.DSHHasImages {
		turn.dshImageAdmission = dshImageAdmitted
	}
	p.cancelSession = session.DSHSessionID
	waitMethod := "session/wait"
	waitParams := map[string]string{"sessionId": session.DSHSessionID}
	commandID := ""
	if opts.Query {
		var query struct {
			Kind      string `json:"kind"`
			Execution struct {
				CommandID string `json:"commandId"`
			} `json:"execution"`
		}
		if json.Unmarshal(receipt, &query) != nil {
			return errors.New("dsh_invalid_query_receipt")
		}
		switch query.Kind {
		case "prompt":
			if opts.OnCommand != nil {
				return errors.New("dsh_goal_command_unavailable")
			}
		case "command":
			commandID = query.Execution.CommandID
			if commandID == "" {
				return errors.New("dsh_invalid_query_receipt")
			}
			if opts.OnCommand != nil {
				if err := opts.OnCommand(commandID); err != nil {
					return err
				}
			}
			waitMethod = "aibertai/query-wait"
			waitParams["commandId"] = commandID
		default:
			return errors.New("dsh_invalid_query_receipt")
		}
	}
	waitResult, err := p.call(ctx, waitMethod, waitParams, event)
	if local != nil {
		local.mu.Lock()
		tooLarge := local.requestTooLarge
		local.mu.Unlock()
		if tooLarge {
			return errors.New("dsh_model_request_too_large")
		}
	}
	if err != nil {
		return err
	}
	memberStopped := false
	if turn.DSHMemberID != "" {
		var member struct {
			Reason, Answer, InvocationID, AbortReason string
			MemberStopped                             bool
		}
		if json.Unmarshal(waitResult, &member) != nil || member.Reason == "" {
			return errors.New("member_outcome_unverified")
		}
		reason, answer = member.Reason, member.Answer
		memberStopped = member.MemberStopped && reason == "aborted" && member.AbortReason == "user"
		if validUUID(member.InvocationID) {
			lastAttempt = member.InvocationID
		}
	}
	if commandID != "" {
		completion, ok := queryCompletions[commandID]
		if !ok {
			return errors.New("dsh_query_completion_missing")
		}
		if opts.VerifyCommand != nil && completion.FinishReason == "completed" {
			if err := opts.VerifyCommand(ctx, commandID, completion.FinalResponse, p.historyCall); err != nil {
				return err
			}
		}
		reason = completion.FinishReason
		if completion.FinalResponse != "" {
			answer = completion.FinalResponse
			turn.Answer = answer
			turn.DSHStreamed = streamText != "" && streamText == answer
			if project != nil {
				if err := project(nil); err != nil {
					return err
				}
			}
		}
	}

	if len(compactPending) != 0 {
		return errors.New("dsh_compaction_usage_invalid")
	}
	if local != nil {
		local.mu.Lock()
		pending := local.pending != nil
		local.mu.Unlock()
		if pending {
			if pendingModelFailure != nil {
				return pendingModelFailure
			}
			return errors.New("dsh_reconciliation_required")
		}
	}
	if denialConfirmed && commandID != "" && reason == "error" {
		terminalConfirmed = true
		for _, run := range turn.ToolRuns {
			if run.Pending {
				terminalConfirmed = false
				break
			}
		}
	}
	if memberStopped && local != nil && !slices.ContainsFunc(turn.ToolRuns, func(run workspaceToolRun) bool { return run.Pending }) {
		// Only the directed child stopped. Its exact user interruption was
		// verified by the SDK, and the gateway has no unsettled invocation.
		cancelConfirmed, turn.DSHCancelConfirmed = true, true
		turn.Answer = answer
		return context.Canceled
	}
	if ctx.Err() != nil && p.cancelAccepted && reason == "aborted" {
		settled := true
		for _, run := range turn.ToolRuns {
			if run.Pending {
				settled = false
				break
			}
		}
		if settled {
			cancelConfirmed = true
			turn.DSHCancelConfirmed = true
			return ctx.Err()
		}
	}
	turn.Answer = answer
	if hasUsage {
		turn.Usage, turn.Tokens = modelUsage{Input: &input, Output: &output}, tokens
	}
	if reason != "completed" {
		if (reason == "error" || reason == "max-tokens") && local != nil && !opts.Query && !slices.ContainsFunc(turn.ToolRuns, func(run workspaceToolRun) bool { return run.Pending }) {
			if code := local.archivedFailure(operationID, lastAttempt, reason); code != "" {
				turn.DSHModelFailureSettled = true
				return errors.New(code)
			}
		}
		return fmt.Errorf("dsh_turn_%s", reason)
	}
	return nil
}
