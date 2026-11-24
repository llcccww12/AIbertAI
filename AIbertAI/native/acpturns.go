package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const maxACPToolCalls = 4
const maxACPDeckToolCalls = 96

var (
	errSessionReportPending = errors.New("session_report_pending")
	errGoalSessionBusy      = errors.New("goal_session_busy")
)

type dshLegacyToolCalls map[string]json.RawMessage

// dshGoalThoughtProof identifies the durable SDK model message whose
// committed receipt was checked before its reasoning entered the checkpoint.
type dshGoalThoughtProof struct {
	GoalID         string `json:"goalId"`
	QueryOperation string `json:"queryOperation"`
	InvocationID   string `json:"invocationId"`
	MessageSeq     int64  `json:"messageSeq"`
	ContentSHA256  string `json:"contentSha256"`
}

func dshGoalThoughtDigest(text string, parts []dshMessagePart) string {
	b, _ := json.Marshal(struct {
		Text  string           `json:"text"`
		Parts []dshMessagePart `json:"parts"`
	}{text, parts})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func validDSHGoalThoughtProof(proof *dshGoalThoughtProof, goalID, text string, parts []dshMessagePart) bool {
	return proof != nil && validUUID(goalID) && proof.GoalID == goalID && validUUID(proof.QueryOperation) && validUUID(proof.InvocationID) && proof.MessageSeq >= 0 && len(parts) > 0 && proof.ContentSHA256 == dshGoalThoughtDigest(text, parts)
}

type dshMessagePart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	End  int    `json:"end,omitempty"`
}

// Text stays in Answer/Preamble; offsets retain its position among reasoning
// blocks without duplicating the entire reply in the session snapshot.
func dshRenderedParts(parts []dshMessagePart, text string) []map[string]string {
	if len(parts) == 0 {
		return []map[string]string{{"type": "text", "text": text}}
	}
	out := make([]map[string]string, 0, len(parts))
	pos := 0
	for _, part := range parts {
		switch part.Type {
		case "thought":
			out = append(out, map[string]string{"type": "thought", "text": part.Text})
		case "text":
			if part.End < pos || part.End > len(text) {
				return []map[string]string{{"type": "text", "text": text}}
			}
			out = append(out, map[string]string{"type": "text", "text": text[pos:part.End]})
			pos = part.End
		default:
			return []map[string]string{{"type": "text", "text": text}}
		}
	}
	if pos != len(text) {
		return []map[string]string{{"type": "text", "text": text}}
	}
	return out
}

type acpTurn struct {
	DSHMemberNotices       []dshMemberNotice `json:"dshMemberNotices,omitempty"`
	DSHInjected            []dshContextEntry `json:"dshInjected,omitempty"`
	DSHMemberID            string            `json:"dshMemberId,omitempty"`
	goalOutcome            string
	dshImageAdmission      dshImageAdmission
	Operation              string                `json:"operationId"`
	ClientID               string                `json:"clientMessageId,omitempty"`
	Model                  string                `json:"model,omitempty"`
	CanvasNodes            []acpCanvasNode       `json:"canvasNodes,omitempty"`
	Attachments            []acpAttachment       `json:"attachments,omitempty"`
	Started                time.Time             `json:"startedAt"`
	Ended                  time.Time             `json:"endedAt,omitzero"`
	Prompt                 string                `json:"prompt"`
	ReportText             bool                  `json:"reportText,omitempty"`
	Resources              []turnResource        `json:"resources,omitempty"`
	ModelInput             string                `json:"modelInput,omitempty"`
	DSHHasImages           bool                  `json:"dshHasImages,omitempty"`
	DSHInputNotAdmitted    bool                  `json:"dshInputNotAdmitted,omitempty"`
	ResponseItems          []json.RawMessage     `json:"responseItems,omitempty"`
	LegacyPrompt           json.RawMessage       `json:"legacyPrompt,omitempty"`
	LegacyResponse         json.RawMessage       `json:"legacyResponse,omitempty"`
	LegacyToolCalls        dshLegacyToolCalls    `json:"legacyToolCalls,omitempty"`
	Answer                 string                `json:"answer,omitempty"`
	DSHParts               []dshMessagePart      `json:"dshParts,omitempty"`
	DSHDeliverables        []dshPresentedFile    `json:"dshDeliverables,omitempty"`
	Tool                   *reportedTool         `json:"tool,omitempty"`
	Call                   *modelToolUse         `json:"modelToolCall,omitempty"`
	Output                 string                `json:"toolOutput,omitempty"`
	ToolRuns               []workspaceToolRun    `json:"toolRuns,omitempty"`
	GoalCheckpoint         *goalCheckpoint       `json:"goalCheckpoint,omitempty"`
	Usage                  modelUsage            `json:"usage"`
	Tokens                 turnTokenFacts        `json:"tokens,omitzero"`
	Permissions            reportedPermissions   `json:"permissions,omitzero"`
	Failed                 bool                  `json:"failed,omitempty"`
	FailureCode            string                `json:"failureCode,omitempty"`
	Interrupted            bool                  `json:"interrupted,omitempty"`
	DSHCompactID           string                `json:"dshCompactCommandId,omitempty"`
	DSHCompactions         []dshCompactionNotice `json:"dshCompactions,omitempty"`
	DSHTodos               *dshTodoSnapshot      `json:"dshTodos,omitempty"`
	DSHTeamBoard           *dshTeamBoard         `json:"dshTeamBoard,omitempty"`
	DSHContext             *dshContextUsage      `json:"dshContext,omitempty"`
	DSHStreamed            bool                  `json:"dshStreamed,omitempty"`
	DSHStep                int                   `json:"dshStep,omitempty"`
	DSHUserReviewed        bool                  `json:"dshUserReviewed,omitempty"`
	DSHReconciled          bool                  `json:"dshReconciled,omitempty"`
	DSHProjectionSeq       *int64                `json:"dshProjectionSeq,omitempty"`
	DSHModelFailureSettled bool                  `json:"dshModelFailureSettled,omitempty"`
	DSHStopObserved        bool                  `json:"dshStopObserved,omitempty"`
	DSHCancelConfirmed     bool                  `json:"dshCancelConfirmed,omitempty"`
	Acked                  bool                  `json:"acked,omitempty"`

	dshCurrentInvocation string
	dshStreamAttempt     string
	DSHGoalID            string               `json:"dshGoalId,omitempty"`
	DSHThoughtProof      *dshGoalThoughtProof `json:"dshThoughtProof,omitempty"`
}

// Older sessions recorded a single tool directly on the turn.
func (t acpTurn) workspaceTools() []workspaceToolRun {
	if t.GoalCheckpoint != nil {
		return t.GoalCheckpoint.Runs
	}
	if len(t.ToolRuns) != 0 || t.Tool == nil {
		return t.ToolRuns
	}
	return []workspaceToolRun{{Call: t.Call, Report: *t.Tool, Output: t.Output, ResponseItems: t.ResponseItems}}
}

func (s acpSession) modelHistory() []toolConversationMessage {
	messages := make([]toolConversationMessage, 0, 2*len(s.Turns))
	compacted := 0
	if s.ContextSummary != nil && s.ContextSummary.Through > 0 && s.ContextSummary.Through <= len(s.Turns) {
		compacted = s.ContextSummary.Through
		user, assistant := "Summarize the conversation so far.", s.ContextSummary.Text
		messages = append(messages, toolConversationMessage{Role: "user", Content: &user}, toolConversationMessage{Role: "assistant", Content: &assistant})
	}
	for _, turn := range s.Turns[compacted:] {
		if turn.DSHMemberID == "" && turn.Answer != "" {
			input := turn.Prompt
			if turn.ModelInput != "" {
				input = turn.ModelInput
			}
			messages = append(messages, toolConversationMessage{Role: "user", Content: &input})
			for _, run := range turn.workspaceTools() {
				if run.Call != nil && run.Report.Failed == 0 {
					assistant := toolConversationMessage{Role: "assistant", ToolCalls: []modelToolUse{*run.Call}, ResponseItems: run.ResponseItems}
					if run.Preamble != "" {
						assistant.Content = &run.Preamble
					}
					messages = append(messages, assistant,
						toolConversationMessage{Role: "tool", Content: &run.Output, ToolCallID: run.Call.ID})
				}
			}
			messages = append(messages, toolConversationMessage{Role: "assistant", Content: &turn.Answer})
		}
	}
	return messages
}

func (l *acpLive) beginTurn(id, op, clientID, prompt string, attachments []acpAttachment, cancel context.CancelFunc) (acpSession, int, bool, error) {
	return l.reserveTurn(id, op, clientID, prompt, attachments, cancel, false, nil)
}

func (l *acpLive) reserveTurn(id, op, clientID, prompt string, attachments []acpAttachment, cancel context.CancelFunc, background bool, queued *desktopQueuedMessage) (acpSession, int, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	session, ok := l.sessions[id]
	if !ok || l.restarting || l.compacting[id] || l.workspaces == nil && session.CWD != l.cwd || !validUUID(op) {
		return session, 0, false, errors.New("session_unavailable_for_prompt")
	}
	if background {
		if session.GoalRun != op || session.Goal == nil || !session.Goal.Current || session.Goal.Status != "active" || l.activeTurns[id] != nil {
			return session, 0, false, errors.New("stale_goal")
		}
		if n := len(session.Turns); n > 0 && session.Turns[n-1].Operation == op && session.Turns[n-1].Ended.IsZero() {
			session.State = "running"
			if err := l.save(session); err != nil {
				return session, 0, false, errors.New("session_write_failed")
			}
			l.trackTurn(id, cancel)
			return session, n - 1, false, nil
		}
	} else if session.GoalRun != "" {
		return session, 0, false, errGoalSessionBusy
	}
	if session.State == "running" || !background && queued == nil && l.queueWorkers[id] != nil {
		return session, 0, false, errors.New("session_unavailable_for_prompt")
	}
	var memberID string
	var canvasNodes []acpCanvasNode
	fromQueue := queued != nil
	if fromQueue {
		if len(session.Queue.Messages) == 0 || session.Queue.Messages[0].ID != clientID ||
			session.Queue.Messages[0].DSHMemberID != queued.DSHMemberID || session.Queue.Messages[0].editID != "" || session.Queue.Messages[0].Text != prompt ||
			!slices.Equal(session.Queue.Messages[0].Attachments, attachments) {
			return session, 0, false, errSessionQueueChanged
		}
		canvasNodes = queued.CanvasNodes
		memberID = queued.DSHMemberID
	}
	for index, turn := range session.Turns {
		if turn.Operation != op {
			continue
		}
		if turn.DSHMemberID != memberID || turn.Prompt != prompt || turn.ClientID != clientID || !slices.Equal(turn.Attachments, attachments) || !slices.Equal(turn.CanvasNodes, canvasNodes) || turn.Ended.IsZero() {
			return session, 0, false, errors.New("session_operation_conflict")
		}
		if !turn.Acked {
			session.State = "running"
			if err := l.save(session); err != nil {
				return session, 0, false, errors.New("session_write_failed")
			}
		}
		return session, index, true, nil
	}
	// DSH never locks a conversation because an earlier turn failed: the agent
	// loop records the failure, returns to idle and admits the next message.
	// The guarantees this gate used to provide are enforced where they belong:
	//   - the failed turn's SDK child is discarded, so no old reply can run on;
	//   - a resumed turn reconciles any pending model invocation before it
	//     prompts (runDSHSessionWithProfile), and fails that attempt only;
	//   - on resume the SDK closes every unanswered tool call with an
	//     "outcome unknown, verify before retrying" result (core/session repair),
	//     so the model, not a lock, decides what to re-check;
	//   - a replayed side-effecting call is still refused by its durable claim.
	// The failed turn stays on disk unchanged, and the explicit recheck action
	// remains available for billing receipts.
	//
	// One case still waits: the last turn's own model call is unsettled in the
	// gateway journal and promptACP could not settle it just now. That call
	// cannot be dispatched past, and keeping it the last turn is what lets the
	// "重新核验" receipt check (and settleStuckDSHModelTurn) resolve it.
	if n := len(session.Turns); !background && session.Engine == "dsh" && n > 0 {
		if last := session.Turns[n-1]; (dshPendingModelFailure(last) || dshStuckModelOnlyTurn(last)) && l.dshModelCallPending(id) {
			return session, 0, false, errors.New("dsh_reconciliation_required")
		}
	}
	if n := len(session.Turns); n > 0 && !session.Turns[n-1].Acked {
		return session, 0, false, errSessionReportPending
	}
	started := time.Now().UTC()
	if started.Before(session.Updated) {
		started = session.Updated
	}
	session.State, session.Updated = "running", started
	if session.ConnectorPreferences == nil {
		prefs := l.connectorPrefs
		session.ConnectorPreferences = &prefs
	}
	index := len(session.Turns)
	turn := acpTurn{Operation: op, ClientID: clientID, DSHMemberID: memberID, Model: session.Model, Started: started, Prompt: prompt, Attachments: attachments, CanvasNodes: canvasNodes}
	if background {
		turn.GoalCheckpoint = &goalCheckpoint{}
	}
	session.Turns = append(slices.Clone(session.Turns), turn)
	if fromQueue {
		session.Queue.Messages = session.Queue.Messages[1:]
		session.advanceQueue()
	}
	if err := l.save(session); err != nil {
		return session, 0, false, errors.New("session_write_failed")
	}
	l.trackTurn(id, cancel)
	return session, index, false, nil
}

func (l *acpLive) trackTurn(id string, cancel context.CancelFunc) {
	if cancel != nil {
		if l.activeTurns == nil {
			l.activeTurns = make(map[string]context.CancelFunc)
		}
		l.activeTurns[id] = cancel
	}
}

func (h *relaySessionHost) submitACPTurn(ctx context.Context, id string, index int, turn acpTurn, register bool) error {
	live := h.live
	auth, err := live.authorization(ctx)
	if err != nil {
		return err
	}
	if register {
		if err := live.client.registerSession(ctx, auth, h.reportMachine(), id, turn.Model, turn.reportTitle(), turn.Started); err != nil {
			return err
		}
	}
	stop := "complete"
	if turn.Failed {
		stop = "error"
	} else if turn.Interrupted {
		stop = "interrupted"
	}
	tools := turn.workspaceTools()
	facts := make([]reportedTool, 0, len(tools))
	for _, tool := range tools {
		if !tool.Pending {
			facts = append(facts, tool.Report)
		}
	}
	input := reportedInput{Kind: "prompt"}
	if turn.ReportText {
		input = turnReportInput(turn.Prompt)
	}
	live.mu.Lock()
	cwd := live.sessions[id].CWD
	live.mu.Unlock()
	return live.client.reportTurn(ctx, auth, h.reportMachine(), id, turn.Model, index, index-1, turn.Started, turn.Ended, stop, turn.Usage, input, turn.reportFacts(cwd), facts...)
}

func (t acpTurn) reportFacts(cwd string) turnReportFacts {
	facts := turnReportFacts{Tokens: t.Tokens, Permissions: &t.Permissions, Resources: t.Resources, SkillEvents: t.skillEvents()}
	if t.DSHContext.valid() {
		window := t.DSHContext.Window
		facts.ContextUsed, facts.ContextWindow = t.DSHContext.Used, &window
	}
	if files := t.mutatedFiles(cwd); len(files) != 0 {
		facts.Files = &reportedFiles{}
		for _, file := range files {
			switch file["operation"] {
			case "created":
				facts.Files.Created++
			case "deleted":
				facts.Files.Deleted++
			default:
				facts.Files.Updated++
			}
		}
	}
	for _, compaction := range t.DSHCompactions {
		if compaction.Status == "completed" {
			facts.Compactions++
		}
	}
	return facts
}

func (h *relaySessionHost) retryPendingReport(ctx context.Context, id string) error {
	live := h.live
	live.mu.Lock()
	session, ok := live.sessions[id]
	index := len(session.Turns) - 1
	if !ok || index < 0 || session.State == "running" || session.Turns[index].Ended.IsZero() {
		live.mu.Unlock()
		return errSessionReportPending
	}
	turn := session.Turns[index]
	if turn.Acked {
		live.mu.Unlock()
		return nil
	}
	// Reserve the session while replaying its durable facts after a disconnect.
	session.State = "running"
	if err := live.save(session); err != nil {
		live.mu.Unlock()
		return err
	}
	live.mu.Unlock()
	reportErr := h.submitACPTurn(ctx, id, index+1, turn, index == 0)
	live.mu.Lock()
	defer live.mu.Unlock()
	session = live.sessions[id]
	turn.Acked = reportErr == nil
	session.State = completedState(turn)
	session.Turns = slices.Clone(session.Turns)
	session.Turns[index] = turn
	return errors.Join(reportErr, live.save(session))
}

func (h *relaySessionHost) promptACP(ctx context.Context, ws *websocket.Conn, request *http.Request, id, op, clientID, prompt string, accepted func(acpSession) error, queued *desktopQueuedMessage, files ...acpAttachment) (any, error) {
	if op == "" {
		var err error
		op, err = newUUID()
		if err != nil {
			return nil, errors.New("session_operation_unavailable")
		}
	}
	live := h.live
	turnCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	_, background := ctx.Value(goalRuntimeContext{}).(*goalRuntime)
	if !background {
		if err := h.preflightOwnedModel(turnCtx, id, op); err != nil {
			return nil, err
		}
		// Settle a stale earlier failure when its evidence allows. This only
		// tidies receipts; the next message is admitted either way.
		_ = h.settleStuckDSHModelTurn(ctx, id)
		_ = h.settleStaleCanvasTurn(id)
	}
	session, index, retry, err := live.reserveTurn(id, op, clientID, prompt, files, cancel, background, queued)
	if errors.Is(err, errSessionReportPending) && h.retryPendingReport(ctx, id) == nil {
		session, index, retry, err = live.reserveTurn(id, op, clientID, prompt, files, cancel, background, queued)
	}
	if err != nil {
		return nil, err
	}
	turn := session.Turns[index]
	attemptStarted := time.Now()
	var previousActiveTime time.Duration
	if background && turn.GoalCheckpoint != nil {
		previousActiveTime = turn.GoalCheckpoint.ActiveTime
		turn.Permissions = turn.GoalCheckpoint.Permissions
	}
	currentGoalAtStart := session.Goal
	register := index == 0 && retry
	if !retry {
		inputPreparationFailed := false
		// At this point the user message and cancellable turn are durable.
		if clientID != "" {
			err = notifyACPTurnState(ctx, ws, session, op, false)
			if err == nil && turn.DSHMemberID == "" {
				err = writeTurnNotification(ctx, ws, "_monkeyai/session/message",
					map[string]any{"sessionId": id, "eventSeq": session.historySequence(), "message": session.promptMessage(index, nil)})
			}
		}
		if err == nil && accepted != nil {
			err = accepted(session)
		}
		if err == nil {
			turn.ReportText = turn.DSHMemberID == "" && h.collectTurnText(turnCtx)
			live.touchDevice(turnCtx)
		}
		if err == nil && index == 0 {
			var auth authorization
			auth, err = live.authorization(turnCtx)
			if err == nil {
				err = live.client.registerSession(turnCtx, auth, h.reportMachine(), id, turn.Model, turn.reportTitle(), turn.Started)
			}
			register = err != nil
		}
		if err == nil {
			if clientID != "" {
				turnCtx = context.WithValue(turnCtx, acpAnswerKey{}, session.responseID(index))
			}
			if session.Engine == "dsh" {
				if background && (len(files) != 0 || len(turn.CanvasNodes) != 0) {
					err = errors.New("dsh_input_unsupported")
				} else {
					turnCtx, err = h.bindDSHRules(turnCtx)
					if err == nil && session.ExpertID != "" && session.ExpertID != designExpertID {
						var releaseExpert func()
						turnCtx, releaseExpert, err = h.bindDSHExpert(turnCtx, session)
						defer releaseExpert()
					}
					if err == nil {
						turn.recordDSHResources(turnCtx, session)
					}
					var modelPrompt string
					if err == nil && !background {
						if len(files) > 0 && h.local != nil {
							var auth authorization
							if err == nil {
								auth, err = live.authorization(turnCtx)
							}
							var selected selectedModelConfig
							if err == nil {
								selected, err = live.client.modelConfig(turnCtx, auth, turn.Model)
							}
							var input dshImageInput
							var hasImage bool
							if err == nil {
								input, hasImage, err = h.prepareDSHImageInput(turnCtx, id, turn.Prompt, files, selected.SupportsVision)
								inputPreparationFailed = err != nil
							}
							if err == nil {
								var text strings.Builder
								for _, block := range input.blocks {
									if block["type"] == "text" {
										text.WriteString(block["text"])
									}
								}
								modelPrompt = text.String()
								if hasImage {
									turn.DSHHasImages = true
									turnCtx = context.WithValue(turnCtx, dshImageInputKey{}, input)
								}
							}
						} else {
							modelPrompt, _, err = h.attachmentPrompt(turnCtx, id, turn.Prompt, files)
						}
						if err == nil && modelPrompt != turn.Prompt {
							turn.ModelInput = modelPrompt
						}
					}
					if err == nil && !background {
						var bridges []dshHostToolBridge
						var instructions string
						bridges, instructions, err = h.dshTurnHostTools(turnCtx, ws, request, session, &turn)
						inputPreparationFailed = errors.Is(err, errDSHCanvasUnavailable)
						if err == nil && instructions != "" {
							turnCtx = context.WithValue(turnCtx, dshExpertInstructionsKey{}, instructions)
						}
						if err == nil && len(bridges) > 0 {
							turnCtx = context.WithValue(turnCtx, dshHostToolBridgesKey{}, bridges)
						}
						if err == nil && len(turn.CanvasNodes) > 0 {
							previous := modelPrompt
							modelPrompt += "\n\nThe user attached references to these canvas nodes. Use canvas_get_selection to inspect the attached selection when needed:"
							for _, node := range turn.CanvasNodes {
								modelPrompt += fmt.Sprintf("\n- %q (%s, node ID %q)", node.Title, node.NodeType, node.ID)
							}
							turn.ModelInput = modelPrompt
							if input, ok := turnCtx.Value(dshImageInputKey{}).(dshImageInput); ok {
								input.blocks = append(input.blocks, map[string]string{"type": "text", "text": strings.TrimPrefix(modelPrompt, previous)})
								turnCtx = context.WithValue(turnCtx, dshImageInputKey{}, input)
							}
						}
					}
					if err == nil {
						if live.store != nil && live.store.Root != nil {
							turnCtx = context.WithValue(turnCtx, dshSideEffectRootKey{}, live.store.Root)
						}
						turnCtx = context.WithValue(turnCtx, dshNativeApprovalKey{}, h.dshNativeApproval(turnCtx, ws, request, session))
						if !background || h.interactions != nil {
							turnCtx = context.WithValue(turnCtx, dshQuestionKey{}, h.dshQuestionHandler(ws, request, session))
						}
						allowWrites := live.workspaces != nil && h.acpScope(request.WithContext(turnCtx), "files:write") == nil
						if allowWrites {
							turnCtx = context.WithValue(turnCtx, dshBoundaryStateKey{}, &dshBoundaryState{})
							turnCtx = context.WithValue(turnCtx, dshWorkspaceAuthorizeKey{}, func(ctx context.Context) error {
								return h.acpScope(request.WithContext(ctx), "files:write")
							})
						}
						if !background && allowWrites {
							turnCtx = context.WithValue(turnCtx, dshWorkspaceBridgeKey{}, h.dshWorkspaceBridge(turnCtx, ws, session, &turn))
							if h.local != nil {
								turnCtx = context.WithValue(turnCtx, dshTrustedLaunchKey{}, true)
							}
						}
						project := func(update map[string]any) error {
							live.mu.Lock()
							current := live.sessions[id]
							if turnCtx.Err() != nil && update != nil {
								switch update["sessionUpdate"] {
								case "agent_message_start", "agent_message_chunk", "agent_thought_chunk":
									// A provisional frame may have raced cancellation after
									// the SDK reader checked its context. Restore the last
									// published state; durable settlement remains admissible.
									previous := current.Turns[index]
									turn.Answer, turn.DSHParts, turn.DSHStreamed = previous.Answer, previous.DSHParts, previous.DSHStreamed
									turn.DSHThoughtProof, turn.dshCurrentInvocation = previous.DSHThoughtProof, previous.dshCurrentInvocation
									live.mu.Unlock()
									return nil
								}
							}
							current.Turns = slices.Clone(current.Turns)
							current.Turns[index] = turn
							saveErr := live.save(current)
							live.mu.Unlock()
							if saveErr != nil {
								return saveErr
							}
							if turn.DSHMemberID == "" && update != nil && (turnCtx.Err() == nil || update["sessionUpdate"] == "agent_message_reset") {
								// A cancelled turn context must not be used for a WebSocket
								// write: a racing cancellation can close the connection.
								notifyCtx := ctx
								if answerID, ok := turnCtx.Value(acpAnswerKey{}).(string); ok {
									notifyCtx = context.WithValue(notifyCtx, acpAnswerKey{}, answerID)
								}
								_ = sendACPToolUpdate(notifyCtx, ws, id, update)
							}
							h.notifyDSHHistory(ctx, current)
							return nil
						}
						if turn.DSHHasImages || turn.ModelInput != "" {
							err = project(nil)
							if err != nil {
								// The SDK has not received the input yet. Drop the
								// oversized projection so the failed turn itself can be saved.
								turn.ModelInput = ""
								inputPreparationFailed = true
								if err.Error() == "desktop_session_snapshot_too_large" {
									err = errors.New("attachment_unavailable_or_too_large")
								}
							}
						}
						if err == nil {
							if background {
								err = h.runDSHGoalTurn(turnCtx, ws, request, session, &turn, index > 0, allowWrites, project)
							} else {
								resume := false
								for _, prior := range session.Turns[:index] {
									if !prior.DSHInputNotAdmitted {
										resume = true
										break
									}
								}
								err = runDSHInteractiveTurn(turnCtx, live.dsh, session, &turn, resume, project)
							}
						}
					}
				}
			} else {
				err = h.runACPModelTurn(turnCtx, ws, request, session, &turn, files)
			}
		}
		notAdmitted := err != nil && (inputPreparationFailed || turn.DSHHasImages && turn.dshImageAdmission == dshImageNotSent)
		if notAdmitted {
			// A local failure or an explicit SDK admission refusal is not an
			// unknown dispatch. Preserve the failure without retaining an image.
			turn.DSHHasImages = false
			turn.DSHInputNotAdmitted = true
		}
		if session.Engine != "dsh" && len(turn.Answer) > 128<<10 {
			turn.Answer = ""
			err = errors.New("session_answer_too_large")
		}
		// An unresolved write stays on disk as evidence, but it must not fail an
		// otherwise completed turn: the tool result already told the model.
		unsettledWrite := session.Engine == "dsh" && live.store != nil && dshUnsettledWorkspaceWrite(live.store.Root, id, turn)
		memberStopped := turn.DSHMemberID != "" && turn.DSHCancelConfirmed && errors.Is(err, context.Canceled)
		turn.Interrupted = err != nil && !notAdmitted && ctx.Err() == nil && (turnCtx.Err() == context.Canceled || memberStopped)
		if session.Engine == "dsh" && turn.Interrupted && !turn.DSHCancelConfirmed {
			turn.DSHCancelConfirmed = !background && h.confirmStoppedDSHReply(ctx, session, &turn)
			if !turn.DSHCancelConfirmed {
				turn.Interrupted = false
				turn.FailureCode = "dsh_reconciliation_required"
			}
		}
		turn.Failed, turn.Ended = err != nil && !turn.Interrupted, time.Now().UTC()
		if turn.DSHModelFailureSettled && (live.store == nil || live.store.Root == nil || dshConnectorAssertSettled(live.store.Root, id) != nil) {
			turn.DSHModelFailureSettled = false
			turn.FailureCode = "dsh_connector_reconciliation_required"
		}
		if err != nil && session.Engine == "dsh" && (dshPendingWrite(turn) || unsettledWrite) {
			turn.Interrupted = false
			turn.Failed = true
			turn.FailureCode = "dsh_write_outcome_unknown"
		}
		if err != nil && session.Engine == "dsh" && strings.Contains(err.Error(), "dsh_canvas_outcome_unknown") {
			turn.Interrupted = false
			turn.Failed = true
			turn.FailureCode = "dsh_canvas_outcome_unknown"
		}
		if turn.Failed {
			if turn.FailureCode == "" {
				turn.FailureCode = safeDesktopTurnFailure(err)
			}
			plannedGoalStop := false
			if background && session.Engine == "dsh" {
				if current, loadErr := live.load(id); loadErr == nil && current.Goal != nil && !current.Goal.hasPendingDSHInvocation() && !dshPendingWrite(turn) {
					plannedGoalStop = current.Goal.Status == "budget_limited" || current.Goal.Status == "clearing" || current.Goal.Status == "paused" && current.Goal.Reason == "user_paused"
				}
			}
			if ctx.Err() == nil && !plannedGoalStop {
				slog.Warn("桌面对话执行失败", "session_id", id, "operation_id", op, "reason", turn.FailureCode)
			}
		}
		if turn.Ended.Before(turn.Started) {
			turn.Ended = turn.Started
		}
		live.mu.Lock()
		delete(live.activeTurns, id)
		current := live.sessions[id]
		if background {
			saved := *current.Turns[index].GoalCheckpoint
			saved.ActiveTime = previousActiveTime + time.Since(attemptStarted)
			current.Turns = slices.Clone(current.Turns)
			current.Turns[index].GoalCheckpoint = &saved
			turn.GoalCheckpoint = &saved
			if ctx.Err() != nil {
				// Leave the unfinished turn with the saved checkpoint for startup recovery.
				saveErr := live.save(current)
				live.mu.Unlock()
				return nil, errors.Join(ctx.Err(), saveErr)
			}
		}
		session.Mode, session.desktopSessionPresentation = current.Mode, current.desktopSessionPresentation
		session.Goal, session.LastGoalRequest = current.Goal, current.LastGoalRequest
		session.GoalRun = current.GoalRun
		session.Queue = current.Queue
		session.Turns = slices.Clone(session.Turns)
		paused := background && current.Goal.Current && (current.Goal.Status == "paused" || turn.Interrupted || turn.Failed)
		if paused {
			saved := current.Turns[index].GoalCheckpoint
			goal := *current.Goal
			session.State = "idle"
			if current.Goal.Status != "paused" && current.Goal.Status != "budget_limited" && current.Goal.Status != "blocked" && current.Goal.Status != "clearing" {
				goal.Status = "paused"
				goal.Reason = turn.goalOutcome
				if goal.Reason == "" && turn.Interrupted {
					goal.Reason = "user_paused"
				}
				if goal.Reason == "" {
					goal.Reason = "execution_failure"
				}
			}
			if saved != nil && saved.Pending != nil {
				goal.Reason, goal.Uncertain = "resume_confirmation", true
			}
			if session.Engine == "dsh" && errors.Is(err, errDSHGoalApprovalUnavailable) {
				goal.Status, goal.Reason = "paused", "approval_unavailable"
			}
			if session.Engine == "dsh" && dshPendingWrite(turn) {
				goal.Status, goal.Reason, goal.Uncertain = "paused", "resume_confirmation", true
			} else if session.Engine == "dsh" && goal.Status != "clearing" && goal.hasPendingDSHInvocation() {
				goal.Status, goal.Reason, goal.Uncertain = "paused", "usage_unknown", true
			}
			goal.Version++
			session.Goal, session.Turns[index] = &goal, current.Turns[index]
		} else {
			settleDesktopGoal(&session, currentGoalAtStart, turn)
			if background {
				turn.GoalCheckpoint, session.GoalRun = nil, ""
			}
			session.Turns[index] = turn
		}
		writeErr := live.save(session)
		if writeErr == nil && !background && session.Engine == "dsh" && turn.Failed && turn.FailureCode == "dsh_write_outcome_unknown" {
			if live.dshStoppedTurns == nil {
				live.dshStoppedTurns = make(map[string]string)
			}
			live.dshStoppedTurns[id] = op
		}
		live.mu.Unlock()
		if writeErr != nil {
			return nil, errors.New("session_write_failed")
		}
		if paused {
			_ = notifyACPReply(ctx, ws, session, index, background)
			_ = notifyACPTurnState(ctx, ws, session, op, true)
			return map[string]string{"stopReason": "cancelled"}, nil
		}
	}
	if !turn.Acked {
		reportErr := h.submitACPTurn(ctx, id, index+1, turn, register)
		turn.Acked = reportErr == nil
		if reportErr != nil {
			slog.Warn("桌面对话回合上报失败，稍后补传", "session_id", id, "operation_id", op, "reason", safeDesktopTurnFailure(reportErr))
		}
		session.State, session.Updated = completedState(turn), turn.Ended
		live.mu.Lock()
		current := live.sessions[id]
		session.Mode, session.desktopSessionPresentation = current.Mode, current.desktopSessionPresentation
		session.Goal, session.LastGoalRequest = current.Goal, current.LastGoalRequest
		session.Queue = current.Queue
		session.Turns = slices.Clone(session.Turns)
		session.Turns[index] = turn
		writeErr := live.save(session)
		live.mu.Unlock()
		if writeErr != nil {
			return nil, errors.New("session_write_failed")
		}
	}
	h.notifyDSHTurnCheckpoint(ctx, session, index, op)
	if clientID != "" {
		_ = notifyACPReply(ctx, ws, session, index, background)
		_ = notifyACPTurnState(ctx, ws, session, op, true)
	}
	if turn.Failed {
		if dshTerminalModelFailure(turn.FailureCode) || turn.FailureCode == "dsh_model_outcome_unknown" || turn.FailureCode == "dsh_model_response_interrupted" || turn.FailureCode == "dsh_connector_outcome_unknown" || turn.FailureCode == "dsh_canvas_outcome_unknown" {
			return nil, errors.New(turn.FailureCode)
		}
		if errors.Is(err, errSkillCommandUnavailable) {
			return nil, errSkillCommandUnavailable
		}
		return nil, errors.New("session_turn_failed")
	}
	if turn.Interrupted {
		return map[string]string{"stopReason": "cancelled"}, nil
	}
	if clientID == "" && !(session.Engine == "dsh" && turn.DSHStreamed && !retry) {
		if err := sendACPText(ctx, ws, id, "agent_message_chunk", turn.Answer); err != nil {
			return nil, err
		}
	}
	return map[string]string{"stopReason": "end_turn"}, nil
}

func (h *relaySessionHost) notifyDSHTurnCheckpoint(ctx context.Context, session acpSession, index int, op string) {
	h.notifyDSHHistory(ctx, session)
	if session.Engine == "dsh" && h.local != nil && h.goals != nil {
		if session.Turns[index].hasHistoryResponse() {
			calls := map[string]any{}
			message := session.responseMessage(index, calls)
			h.goals.notify(ctx, "_monkeyai/session/message", map[string]any{"sessionId": session.ID, "eventSeq": session.historySequence(), "message": message, "toolCalls": calls, "checkpoint": true})
		}
		h.goals.notify(ctx, "_monkeyai/session/state", acpTurnStatePayload(session, op, true))
	}
}

func (h *relaySessionHost) notifyDSHHistory(ctx context.Context, session acpSession) {
	if session.Engine == "dsh" && h.local != nil && h.goals != nil {
		h.goals.notify(ctx, "_monkeyai/session/history_changed", map[string]any{"sessionId": session.ID, "eventSeq": session.historySequence()})
	}
}

func notifyACPReply(ctx context.Context, ws *websocket.Conn, session acpSession, index int, checkpoint bool) error {
	if !session.Turns[index].hasHistoryResponse() {
		return nil
	}
	calls := map[string]any{}
	message := session.responseMessage(index, calls)
	return writeTurnNotification(ctx, ws, "_monkeyai/session/message",
		map[string]any{"sessionId": session.ID, "eventSeq": session.historySequence(), "message": message, "toolCalls": calls, "checkpoint": checkpoint})
}

func notifyACPTurnState(ctx context.Context, ws *websocket.Conn, session acpSession, operation string, finished bool) error {
	return writeTurnNotification(ctx, ws, "_monkeyai/session/state", acpTurnStatePayload(session, operation, finished))
}

func acpTurnStatePayload(session acpSession, operation string, finished bool) map[string]any {
	title, source := session.sessionTitle()
	state := map[string]any{"status": session.State, "executionId": operation, "title": title, "titleSource": source}
	if finished {
		state["executionId"] = nil
		if reason := session.lastTurnFailure(); reason != "" {
			state["lastError"] = reason
		} else {
			state["lastError"] = ""
		}
	}
	session.addDSHTodoMetadata(state)
	state["goal"] = goalView(session)
	return map[string]any{
		"sessionId": session.ID, "eventSeq": session.historySequence(), "state": state, "finish": finished,
	}
}

func (s acpSession) lastTurnFailure() string {
	if s.State == "error" && len(s.Turns) != 0 {
		return s.Turns[len(s.Turns)-1].FailureCode
	}
	return ""
}

func (h *relaySessionHost) runACPModelTurn(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn, files []acpAttachment) error {
	live := h.live
	id := session.ID
	task, skillInstructions := turn.Prompt, ""
	var skillPackage *desktopSkillPackage
	if live.workspaces != nil {
		var err error
		task, skillInstructions, skillPackage, err = h.desktopSkillInstructions(ctx, task)
		if err != nil {
			return err
		}
	}
	modelPrompt, attachments, err := h.attachmentPrompt(ctx, id, task, files)
	if err != nil {
		return err
	}
	if modelPrompt != turn.Prompt {
		turn.ModelInput = modelPrompt
	}
	if len(turn.CanvasNodes) != 0 {
		modelPrompt += "\n\nThe user attached references to these canvas nodes. Use canvas_get_selection to inspect the attached selection when needed:"
		for _, node := range turn.CanvasNodes {
			modelPrompt += fmt.Sprintf("\n- %q (%s, node ID %q)", node.Title, node.NodeType, node.ID)
		}
	}
	approved := append([]string{h.approvedFile}, slices.Sorted(maps.Keys(attachments))...)
	limit, allowWrite := 1, false
	var connectors map[string]connectorTool
	localMCP := map[string]*desktopMCP{}
	defer func() {
		for _, server := range localMCP {
			server.close()
		}
	}()
	var workspace *workspaceToolPolicy
	var design json.RawMessage
	if live.workspaces != nil {
		approved = nil
		limit = maxACPToolCalls
		allowWrite = h.acpScope(request.WithContext(ctx), "files:write") == nil
		workspace = &workspaceToolPolicy{CanWrite: allowWrite, SkillScript: skillPackage != nil && h.local != nil && allowWrite}
		prefs := connectorPreferences{}.normalized()
		if session.ConnectorPreferences != nil {
			prefs = session.ConnectorPreferences.normalized()
			prefs.Enabled = maps.Clone(prefs.Enabled)
		}
		var expert desktopExpert
		if session.ExpertID != "" && session.ExpertID != designExpertID {
			expert, err = h.resolveDesktopExpert(ctx, session.ExpertID)
			if err != nil {
				return err
			}
			auth, err := live.authorization(ctx)
			if err != nil {
				return err
			}
			workspace.Instructions, err = live.client.expertInstructions(ctx, auth, expert)
			if err != nil {
				return err
			}
			for _, link := range expert.Connectors {
				if validUUID(link.ID) && link.Required {
					prefs.Enabled["mcp/server/"+link.ID] = true
				}
			}
		}
		imported := h.runtimeResources()
		var importedExpert importedExpertRuntime
		if imported != nil {
			importedExpert = imported.Experts[session.ExpertID]
		}
		designExpertSelected := session.ExpertID == "" || session.ExpertID == designExpertID
		if (designExpertSelected || len(importedExpert.DesignTools) != 0) && h.local != nil && h.designBridge != nil &&
			h.acpScope(request.WithContext(ctx), "files:read") == nil {
			root, rootErr := live.workspaces.sessionRoot(session)
			if rootErr == nil {
				design = workspaceDesign(root, id)
				root.Close()
			} else if len(importedExpert.DesignTools) != 0 && allowWrite {
				return rootErr
			}
			if design == nil && len(importedExpert.DesignTools) != 0 && allowWrite {
				if _, err := h.designBridge.ensureCanvas(ctx, session); err != nil {
					return err
				}
				root, err := live.workspaces.sessionRoot(session)
				if err != nil {
					return err
				}
				design = workspaceDesign(root, id)
				root.Close()
				if design == nil {
					return errors.New("design_canvas_unavailable")
				}
			}
			if design != nil || (designExpertSelected && rootErr == nil) {
				if design != nil {
					limit = 12
					var task struct{ DesignType string }
					if json.Unmarshal(design, &task) == nil && task.DesignType == "slide_deck" {
						// Each editable slide needs separate create, write and commit calls.
						limit = maxACPDeckToolCalls
						workspace.SlideDeck = true
					}
				}
				workspace.DesignTools, err = h.designBridge.tools(ctx, session, design)
				if err != nil {
					return err
				}
				if designExpertSelected {
					workspace.Instructions = designExpertInstructions(design) +
						"\nWhen the user asks for an image, check the selected image model and call media_generate_image (or media_edit_image for edits). Report the generated result, or explain the tool error. Do not replace an image request with HTML."
				} else {
					workspace.DesignTools = slices.DeleteFunc(workspace.DesignTools, func(tool desktopDesignTool) bool {
						return !slices.Contains(importedExpert.DesignTools, tool.Name)
					})
				}
			}
		}
		if skillInstructions != "" {
			workspace.Instructions = strings.TrimSpace(workspace.Instructions + "\n\n" + skillInstructions)
		}
		workspace.Preferences = prefs.builtinTools()
		if prefs.hasEnabledConnector() && h.acpScope(request.WithContext(ctx), "resources:manage") == nil {
			auth, err := live.authorization(ctx)
			if err != nil {
				return err
			}
			connectors, err = live.client.activeConnectorTools(ctx, auth, prefs)
			if err != nil {
				return err
			}
			for _, link := range expert.Connectors {
				for alias, tool := range connectors {
					if tool.ConnectorID == link.ID && !link.allows(tool.Name) {
						delete(connectors, alias)
					}
				}
			}
		}
		if h.local != nil && imported != nil {
			profile := imported.Experts[session.ExpertID]
			workspace.Preferences = workspace.Preferences.intersect(profile.Builtin)
			for alias, tool := range connectors {
				if !imported.Filters[tool.ConnectorID].allows(tool.Name) {
					delete(connectors, alias)
				}
			}
			if connectors == nil {
				connectors = make(map[string]connectorTool)
			}
			for _, connectorID := range slices.Sorted(maps.Keys(imported.Local)) {
				entry := imported.Local[connectorID]
				enabled, set := prefs.Enabled["mcp/"+entry.Name]
				selected := enabled || !set && (entry.Enabled == nil || *entry.Enabled)
				if !selected && !slices.Contains(profile.LocalTools, connectorID) {
					continue
				}
				executable, err := localMCPExecutable(entry.Command)
				if err != nil {
					return err
				}
				callID := turn.Operation + "-local-mcp-" + connectorID
				permission, err := h.requestACPToolApproval(ctx, ws, id, callID, "启动本机连接器 "+entry.Name, entry.Name, "execute",
					map[string]any{"command": executable, "arguments": entry.Args})
				turn.Permissions.Asked += permission.Asked
				turn.Permissions.Allowed += permission.Allowed
				turn.Permissions.Denied += permission.Denied
				status := "failed"
				if err == nil {
					server, startErr := openLocalMCP(ctx, entry, executable, session.CWD)
					if startErr == nil {
						localMCP[connectorID] = server
						tools, listErr := server.tools(connectorID, entry, prefs.Tools[entry.Name])
						if listErr == nil {
							maps.Copy(connectors, tools)
							status = "completed"
						}
					}
				}
				_ = sendACPToolUpdate(ctx, ws, id, map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": callID, "status": status})
				if err != nil && permission.Denied == 0 {
					// An explicit denial may still allow a text-only answer.
					return err
				}
			}
		}
	}
	budget := live.modelBudget
	if live.store == nil {
		budget = defaultModelInputBudget
	}
	selected, protocol := turn.Model, live.protocol
	if selected == "" {
		selected = modelID
	}
	var modelOptions map[string]any
	stream := turn.ClientID != ""
	if h.local != nil || live.cloudWorkspace != "" || selected != modelID {
		auth, authErr := live.authorization(ctx)
		if authErr != nil {
			return authErr
		}
		config, modelErr := live.client.modelConfig(ctx, auth, selected)
		if modelErr != nil {
			return modelErr
		}
		protocol, budget = config.Protocol, config.inputBudget()
		modelOptions = session.Thinking.requestFields(config)
		stream = stream && config.StreamMode != "off"
	}
	var goalRun *desktopGoalRun
	if goal := session.Goal; h.local != nil && workspace != nil && goal != nil && goal.Current && goal.Status == "active" {
		goalRun = &desktopGoalRun{remaining: goal.Limits.Tokens - goal.Usage.Tokens, active: func() bool {
			current, err := live.load(id)
			return err == nil && current.Goal != nil && current.Goal.ID == goal.ID &&
				current.Goal.Current && current.Goal.Status == "active"
		}}
		if _, background := ctx.Value(goalRuntimeContext{}).(*goalRuntime); background {
			limit = maxBackgroundGoalTools
			goalRun.checkpoint = turn.GoalCheckpoint
			attemptStarted := time.Now()
			goalRun.save = func(checkpoint goalCheckpoint) error {
				checkpoint.ActiveTime = turn.GoalCheckpoint.ActiveTime + time.Since(attemptStarted)
				checkpoint.Permissions = turn.Permissions
				row, err := live.checkpointGoal(id, turn.Operation, checkpoint)
				if err == nil && checkpoint.Pending == nil && ctx.Err() == nil {
					_ = notifyACPReply(ctx, ws, row, len(row.Turns)-1, true)
					_ = notifyACPTurnState(ctx, ws, row, turn.Operation, false)
				}
				return err
			}
		}
		workspace.Goal = goalRun
		workspace.Instructions += "\nThe user's current goal is: " + goal.Objective +
			"\nWork toward this goal. End your answer with a separate line [GOAL_COMPLETE] only when it is achieved. Otherwise report progress and any blockers."
	}
	var delta func(string) error
	if stream {
		sentBytes := 0
		delta = func(text string) error {
			sentBytes += len(text)
			if sentBytes > 128<<10 {
				return errors.New("session_answer_too_large")
			}
			return sendACPToolUpdate(ctx, ws, id, map[string]any{
				"sessionUpdate": "agent_message_chunk",
				"content":       map[string]string{"type": "text", "text": text},
			})
		}
	}
	turn.Answer, turn.Usage, turn.ToolRuns, err = live.client.completeWorkspaceToolTurn(ctx, protocol, h.modelKey(), id, selected, approved, modelPrompt,
		session.modelHistory(), false, allowWrite, limit, connectors, workspace, budget, modelOptions, func(message toolConversationMessage, args toolArguments, step int) (output string, toolErr error) {
			call := message.ToolCalls[0]
			call.ID = fmt.Sprintf("%s-tool-%d", turn.Operation, step+1)
			defer func() {
				update := map[string]any{"sessionUpdate": "tool_call_update", "toolCallId": call.ID,
					"status": "completed", "rawOutput": output}
				if toolErr != nil {
					update["status"], update["rawOutput"] = "failed", toolErr.Error()
				}
				_ = sendACPToolUpdate(ctx, ws, id, update)
			}()
			var permissions reportedPermissions
			var permissionErr error
			if args.SkillScript != nil {
				if skillPackage == nil || !workspace.SkillScript {
					return "", errRelayAccess
				}
				permissions, permissionErr = h.requestACPToolApproval(ctx, ws, id, call.ID, "运行技能脚本 "+skillPackage.Name+" / "+args.SkillScript.Path, call.Function.Name, "execute", args.JSON)
			} else if tool := args.Connector; tool != nil {
				title := tool.ConnectorName + " / " + tool.Name
				permissions, permissionErr = h.requestACPToolApproval(ctx, ws, id, call.ID, title, tool.Name, "other",
					map[string]any{"connector": tool.ConnectorName, "tool": tool.Name, "arguments": args.JSON})
			} else if args.Design {
				permissions, permissionErr = h.requestACPToolApproval(ctx, ws, id, call.ID, call.Function.Name, call.Function.Name, workspaceToolKind(call.Function.Name), args.JSON)
			} else if args.Compat != nil {
				if args.Compat.Patch != nil {
					if h.acpScope(request.WithContext(ctx), "files:read") != nil ||
						h.acpScope(request.WithContext(ctx), "files:write") != nil {
						return "", errRelayAccess
					}
					plan, err := h.prepareToolWorkspacePatch(session, turn.Operation, args.Compat.Patch)
					if err != nil {
						return "", err
					}
					args.Compat.PatchPlan = &plan
				}
				permissions, permissionErr = h.requestACPToolApproval(ctx, ws, id, call.ID, workspaceCompatTitle(call.Function.Name), call.Function.Name, workspaceToolKind(call.Function.Name), args.JSON)
			} else {
				permissions, permissionErr = h.approveACPTool(ctx, ws, id, call, args.Workspace)
			}
			turn.Permissions.Asked += permissions.Asked
			turn.Permissions.Allowed += permissions.Allowed
			turn.Permissions.Denied += permissions.Denied
			if permissionErr != nil {
				return "", permissionErr
			}
			if goalRun != nil && !goalRun.active() {
				return "", errors.New("tool_permission_cancelled")
			}
			if args.SkillScript != nil {
				root, err := live.workspaces.sessionRoot(session)
				if err != nil {
					return "", err
				}
				defer root.Close()
				auth, err := live.authorization(ctx)
				if err != nil {
					return "", err
				}
				return live.client.executeSkillScript(ctx, auth, *skillPackage, *args.SkillScript, root.Name())
			}
			if args.Design {
				if call.Function.Name == designSelectionTool && len(turn.CanvasNodes) != 0 {
					nodeIDs := make([]string, 0, len(turn.CanvasNodes))
					for _, node := range turn.CanvasNodes {
						nodeIDs = append(nodeIDs, node.ID)
					}
					args.JSON, _ = json.Marshal(map[string][]string{"node_ids": nodeIDs})
				}
				return h.designBridge.call(ctx, session, design, call.Function.Name, args.JSON)
			}
			if args.Connector != nil {
				if client := localMCP[args.Connector.ConnectorID]; client != nil {
					return client.invoke(*args.Connector, args.JSON)
				}
			}
			return h.executeACPModelTool(ctx, request, session, turn.Operation, step, call.Function.Name, args, attachments)
		}, delta)
	if goalRun != nil {
		turn.goalOutcome = goalRun.outcome
	}
	if goalRun == nil && workspace != nil && workspace.SlideDeck && ctx.Err() == nil &&
		slices.ContainsFunc(turn.ToolRuns, func(run workspaceToolRun) bool {
			return run.Report.Name == designCreateHTMLTool || run.Report.Name == designCommitHTMLTool
		}) {
		result, statusErr := h.designBridge.call(ctx, session, design, designReadTool, json.RawMessage(`{}`))
		var canvas struct {
			Document struct {
				SlideCount, DraftSlideCount int
			}
		}
		if statusErr == nil && json.Unmarshal([]byte(result), &canvas) == nil && canvas.Document.DraftSlideCount > 0 {
			progress := fmt.Sprintf("画布实际已提交 %d/%d 页，另有 %d 页待提交。整套 PPT 预览和导出需等这些页面提交后才能使用。",
				canvas.Document.SlideCount-canvas.Document.DraftSlideCount, canvas.Document.SlideCount, canvas.Document.DraftSlideCount)
			turn.Answer = strings.TrimSpace(turn.Answer + "\n\n" + progress)
		}
	}
	return err
}

// A lost result after a mutating tool started cannot prove whether it wrote.
func dshPendingWrite(turn acpTurn) bool {
	for _, run := range turn.ToolRuns {
		if run.Pending && run.Call != nil && (run.Call.Function.Name == "write" || run.Call.Function.Name == "edit") {
			return true
		}
	}
	return false
}

func safeDesktopTurnFailure(err error) string {
	if strings.Contains(err.Error(), "dsh_canvas_outcome_unknown") {
		return "dsh_canvas_outcome_unknown"
	}
	if strings.Contains(err.Error(), "dsh_connector_outcome_unknown") || strings.Contains(err.Error(), "dsh_connector_dispatch_uncertain") {
		return "dsh_connector_outcome_unknown"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "backend_request_timeout"
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return "backend_response_interrupted"
	}
	message := err.Error()
	if knowledgeErrorCode.MatchString(message) {
		return message
	}
	for _, prefix := range []string{"POST /v1/chat/completions: HTTP ", "POST /v1/responses: HTTP "} {
		if len(message) == len(prefix)+3 && message[:len(prefix)] == prefix {
			return message
		}
	}
	return "session_turn_failed"
}

func (h *relaySessionHost) executeACPModelTool(ctx context.Context, request *http.Request, session acpSession, operation string, step int, name string, args toolArguments, attachments map[string]string) (string, error) {
	if tool := args.Connector; tool != nil {
		if err := h.acpScope(request.WithContext(ctx), "resources:manage"); err != nil {
			return "", errRelayAccess
		}
		auth, err := h.live.authorization(ctx)
		if err != nil {
			return "", err
		}
		return h.live.client.callConnector(ctx, auth, session.ID, operation, step, *tool, args.JSON)
	}
	if builtinWorkspaceWrite(name) {
		if err := h.acpScope(request.WithContext(ctx), "files:write"); err != nil {
			return "", errRelayAccess
		}
	}
	if args.Compat != nil {
		if err := h.acpScope(request.WithContext(ctx), "files:read"); err != nil {
			return "", errRelayAccess
		}
		return h.executeWorkspaceCompat(ctx, session, name, *args.Compat, attachments)
	}
	if name == workspaceWriteTool {
		return h.writeToolWorkspaceFile(ctx, session, args.Workspace.Path, *args.Workspace.Content)
	}
	if data, exists := attachments[args.Workspace.Path]; exists {
		return data, nil
	}
	return h.readToolWorkspaceFile(session, args.Workspace.Path)
}

func (h *relaySessionHost) readToolWorkspaceFile(row acpSession, path string) (string, error) {
	if h.approvedFile != "" && path == h.approvedFile && row.Workspace != "selected" {
		return readApprovedToolFile(h.root, h.approvedFile, path)
	}
	if h.live.workspaces == nil {
		return "", errWorkspacePath
	}
	root, err := h.live.workspaces.sessionRoot(row)
	if err != nil {
		return "", err
	}
	defer root.Close()
	return readWorkspaceText(root, path)
}

func workspaceToolKind(name string) string {
	if name == "write" || name == "edit" || builtinWorkspaceWrite(name) || designWriteTool(name) {
		return "edit"
	}
	if name == legacyGrepTool || name == legacyGlobTool || name == directoryTool {
		return "search"
	}
	return "read"
}

func workspaceCompatTitle(name string) string {
	switch name {
	case legacyWriteTool:
		return "写入工作区文件"
	case legacyEditTool:
		return "修改工作区文件"
	case legacyPatchTool:
		return "应用工作区补丁"
	case legacyGrepTool:
		return "搜索工作区内容"
	case legacyGlobTool, directoryTool:
		return "查找工作区文件"
	default:
		return "读取工作区文件"
	}
}

type acpAnswerKey struct{}
type dshNativeApprovalWaitKey struct{}

func acpApprovalWaitContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if enabled, _ := ctx.Value(dshNativeApprovalWaitKey{}).(bool); enabled {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, 30*time.Second)
}

func sendACPToolUpdate(ctx context.Context, ws *websocket.Conn, id string, update map[string]any) error {
	params := map[string]any{"sessionId": id, "update": update}
	if answerID, ok := ctx.Value(acpAnswerKey{}).(string); ok {
		params["_meta"] = map[string]string{"messageId": answerID}
	}
	return writeTurnNotification(ctx, ws, "session/update", params)
}

func (h *relaySessionHost) approveACPTool(ctx context.Context, ws *websocket.Conn, sessionID string, call modelToolUse, args workspaceToolArguments) (reportedPermissions, error) {
	title := "读取会话文件"
	input := map[string]string{"path": args.Path}
	if args.Content != nil {
		title = "写入工作区文件"
		input["content"] = *args.Content
	}
	return h.requestACPToolApproval(ctx, ws, sessionID, call.ID, title, call.Function.Name, workspaceToolKind(call.Function.Name), input)
}

func (h *relaySessionHost) requestACPToolApproval(ctx context.Context, ws *websocket.Conn, sessionID, callID, title, name, kind string, input any) (reportedPermissions, error) {
	return h.requestToolApproval(ctx, ws, sessionID, callID, title, name, kind, input, true, false)
}

// DSH has already published its tool/call. The permission request carries
// the host's approval details without replacing the model's tool arguments.
func (h *relaySessionHost) requestDSHToolApproval(ctx context.Context, ws *websocket.Conn, sessionID, callID, title, name, kind string, input any, allowBypass bool) (reportedPermissions, error) {
	return h.requestToolApproval(ctx, ws, sessionID, callID, title, name, kind, input, allowBypass, true)
}

func (h *relaySessionHost) requestToolApproval(ctx context.Context, ws *websocket.Conn, sessionID, callID, title, name, kind string, input any, allowBypass, callProjected bool) (reportedPermissions, error) {
	pending := map[string]any{"toolCallId": callID, "title": title, "name": name,
		"kind": kind, "status": "pending", "rawInput": input}
	publishCall := func(ctx context.Context) error {
		if callProjected {
			return nil
		}
		update := maps.Clone(pending)
		update["sessionUpdate"] = "tool_call"
		return sendACPToolUpdate(ctx, ws, sessionID, update)
	}

	if allowBypass && h.local != nil && h.live != nil {
		row, err := h.live.load(sessionID)
		if err == nil && row.permissionMode() == modeFullAccess {
			return reportedPermissions{}, publishCall(ctx)
		}
	}
	if callProjected && h.interactions != nil {
		ctx = context.WithValue(ctx, dshNativeApprovalWaitKey{}, true)
	}
	ctx, cancel := acpApprovalWaitContext(ctx)
	defer cancel()
	permissionID := "permission-" + callID
	if err := publishCall(ctx); err != nil {
		return reportedPermissions{}, errors.New("tool_approval_unavailable")
	}
	permission := map[string]any{
		"jsonrpc": "2.0", "id": permissionID, "method": "session/request_permission",
		"params": map[string]any{"sessionId": sessionID, "toolCall": pending,
			"options": []map[string]string{
				{"optionId": "allow", "kind": "allow_once", "name": "允许一次"},
				{"optionId": "deny", "kind": "reject_once", "name": "拒绝"},
			},
		},
	}
	if callProjected && h.interactions != nil {
		reply, err := h.interactions.ask(ctx, ws, "session/request_permission", permission["params"])
		counts := reportedPermissions{Asked: 1}
		if err != nil {
			return counts, err
		}
		return applyACPPermissionReply(counts, reply)
	}
	if g, ok := ctx.Value(goalRuntimeContext{}).(*goalRuntime); ok {
		choice, err := g.permission(ctx, permissionID, permission)
		switch choice {
		case "allow":
			return reportedPermissions{Asked: 1, Allowed: 1}, nil
		case "deny":
			return reportedPermissions{Asked: 1, Denied: 1}, errors.New("tool_permission_denied")
		}
		if err == nil {
			err = errors.New("tool_permission_cancelled")
		}
		return reportedPermissions{}, err
	}
	permissions, ok := ctx.Value(acpPermissionContextKey{}).(*sync.Map)
	if !ok {
		return reportedPermissions{}, errors.New("tool_approval_unavailable")
	}
	waiter := &acpPermissionWaiter{sessionID: sessionID, reply: make(chan acpPermissionReply, 1)}
	permissions.Store(permissionID, waiter)
	defer permissions.Delete(permissionID)
	if err := wsjson.Write(ctx, ws, permission); err != nil {
		return reportedPermissions{}, errors.New("tool_approval_unavailable")
	}
	counts := reportedPermissions{Asked: 1}
	viewerDone, _ := ctx.Value(acpViewerDoneKey{}).(<-chan struct{})
	select {
	case reply := <-waiter.reply:
		return applyACPPermissionReply(counts, reply)
	case <-viewerDone:
		return counts, errors.New("tool_approval_unavailable")
	case <-ctx.Done():
		return counts, errors.New("tool_approval_unavailable")
	}
}

func applyACPPermissionReply(counts reportedPermissions, reply acpPermissionReply) (reportedPermissions, error) {
	var result struct {
		Outcome struct{ Outcome, OptionID string }
	}
	if len(reply.Error) != 0 || json.Unmarshal(reply.Result, &result) != nil || result.Outcome.Outcome != "selected" {
		return counts, errors.New("tool_permission_cancelled")
	}
	switch result.Outcome.OptionID {
	case "allow":
		counts.Allowed = 1
		return counts, nil
	case "deny":
		counts.Denied = 1
		return counts, errors.New("tool_permission_denied")
	default:
		return counts, errors.New("invalid_tool_permission")
	}
}

func (session acpSession) historySequence() int {
	count := 0
	for _, turn := range session.Turns {
		if turn.DSHMemberID != "" {
			continue
		}
		count++
		if turn.hasHistoryResponse() {
			count++
		}
	}
	return count
}

func (t acpTurn) hasModelFailureRecord() bool {
	return t.Failed && !t.DSHReconciled && strings.HasPrefix(t.FailureCode, "dsh_model_")
}

func (t acpTurn) hasHistoryResponse() bool {
	if t.DSHMemberID != "" {
		return false
	}
	return t.hasModelFailureRecord() || t.Answer != "" || len(t.DSHMemberNotices) != 0 || len(t.DSHCompactions) != 0 || t.DSHTeamBoard.visible() || len(t.DSHParts) > 0 || len(t.DSHDeliverables) > 0 || len(t.workspaceTools()) > 0 || t.GoalCheckpoint != nil && t.GoalCheckpoint.Progress != "" || t.Interrupted && t.DSHCancelConfirmed && (!t.Failed || t.DSHReconciled)
}

func (session acpSession) toolHistoryID(turn acpTurn, index, step int) string {
	if len(turn.ToolRuns) == 0 && turn.GoalCheckpoint == nil {
		return fmt.Sprintf("%s-%d-tool", session.ID, index+1)
	}
	return fmt.Sprintf("%s-tool-%d", turn.Operation, step+1)
}

func (session acpSession) promptMessage(index int, auth *relayAuthorization) map[string]any {
	turn := session.Turns[index]
	if len(turn.LegacyPrompt) != 0 {
		var original map[string]any
		if json.Unmarshal(turn.LegacyPrompt, &original) == nil && original != nil {
			return original
		}
	}
	id := fmt.Sprintf("%s-%d", session.ID, index+1)
	userParts := []map[string]any{{"type": "text", "text": turn.Prompt}}
	for _, file := range turn.Attachments {
		part := map[string]any{"type": "file", "name": file.Name}
		if file.CanvasNodeID != "" {
			part["canvasNodeId"] = file.CanvasNodeID
		}
		if file.URI != "" {
			if auth == nil {
				part["uri"] = file.URI
				part["employeeAttachment"] = true
			} else if path := historyFilePath(file.URI, file.Name); path != "" {
				part["fileRef"] = historyRelayRef(*auth, session.ID, path)
			}
		} else {
			part["fileRef"] = file.Ref
		}
		userParts = append(userParts, part)
	}
	for _, node := range turn.CanvasNodes {
		userParts = append(userParts, map[string]any{"type": "canvas_node", "id": node.ID, "title": node.Title, "nodeType": node.NodeType})
	}
	user := map[string]any{"id": id + "-prompt", "role": "user", "at": turn.Started.UnixMilli(), "parts": userParts}
	if turn.ClientID != "" {
		user["queueMessageId"] = turn.ClientID
	}
	return user
}

func (session acpSession) responseID(index int) string {
	return fmt.Sprintf("%s-%d-response", session.ID, index+1)
}

func (session acpSession) responseMessage(index int, calls map[string]any) map[string]any {
	turn := session.Turns[index]
	if len(turn.LegacyResponse) != 0 {
		var original map[string]any
		if json.Unmarshal(turn.LegacyResponse, &original) == nil && original != nil {
			for id, call := range turn.LegacyToolCalls {
				calls[id] = call
			}
			return original
		}
	}
	parts := []map[string]string{}
	goalTurn := turn.GoalCheckpoint != nil || turn.DSHGoalID != "" || session.Engine == "dsh" && session.Goal != nil && turn.Operation == session.Goal.ID
	notices := turn.DSHCompactions
	appendCompactions := func(afterTools int) {
		for len(notices) > 0 && notices[0].AfterTools <= afterTools {
			parts = append(parts, notices[0].part(turn))
			notices = notices[1:]
		}
	}
	runs := turn.workspaceTools()
	for step, run := range runs {
		appendCompactions(step)
		if len(run.DSHParts) != 0 && (!goalTurn || validDSHGoalThoughtProof(run.DSHThoughtProof, turn.DSHGoalID, run.Preamble, run.DSHParts)) {
			parts = append(parts, dshRenderedParts(run.DSHParts, run.Preamble)...)
		} else if run.Preamble != "" {
			parts = append(parts, map[string]string{"type": "text", "text": run.Preamble})
		}
		callID, state := session.toolHistoryID(turn, index, step), "completed"
		if run.Pending {
			state = "pending"
			if run.DSHCallID != "" && session.State == "running" && turn.Ended.IsZero() {
				state = "in_progress"
			}
		} else if run.Report.Failed > 0 {
			state = "failed"
		}
		kind := workspaceToolKind(run.Report.Name)
		if run.Report.Category == "connector" {
			kind = "other"
		}
		title := run.Report.Name
		if run.DisplayName != "" {
			title = run.DisplayName
		}
		if run.DisplayKind != "" {
			kind = run.DisplayKind
		}
		calls[callID] = map[string]any{"id": callID, "name": run.Report.Name, "title": title, "kind": kind, "status": state,
			"output": run.Output, "isError": run.Report.Failed > 0}
		if turn.Failed && dshRecoverableWorkspaceFailure(turn.FailureCode) && step == len(turn.workspaceTools())-1 && run.Call != nil && (run.Call.Function.Name == "write" || run.Call.Function.Name == "edit") {
			if turn.DSHUserReviewed {
				calls[callID].(map[string]any)["userReviewed"] = true
			} else if !turn.DSHReconciled {
				calls[callID].(map[string]any)["outcomeUnknown"] = true
			}
		}
		if turn.Failed && ((turn.FailureCode == "dsh_connector_outcome_unknown" && run.Report.Category == "connector") || (turn.FailureCode == "dsh_canvas_outcome_unknown" && run.Report.Category == "canvas" && !turn.DSHReconciled)) && step == len(turn.workspaceTools())-1 {
			calls[callID].(map[string]any)["outcomeUnknown"] = true
		}
		if run.DSHCallID != "" && run.Call != nil {
			var input any
			if json.Unmarshal([]byte(run.Call.Function.Arguments), &input) != nil {
				input = run.Call.Function.Arguments
			}
			calls[callID].(map[string]any)["rawInput"] = input
			calls[callID].(map[string]any)["dshCallId"] = run.DSHCallID
			if len(run.DSHResultMeta) > 0 {
				calls[callID].(map[string]any)["dshResultMeta"] = run.DSHResultMeta
			}
		}
		parts = append(parts, map[string]string{"type": "tool_call", "toolCallId": callID})
	}
	appendCompactions(len(runs))
	progress := turn.Answer
	if turn.GoalCheckpoint != nil {
		progress = turn.GoalCheckpoint.Progress
	}
	progressParts, proof := turn.DSHParts, turn.DSHThoughtProof
	if turn.GoalCheckpoint != nil {
		progressParts, proof = turn.GoalCheckpoint.DSHParts, turn.GoalCheckpoint.DSHThoughtProof
	}
	if len(progressParts) != 0 && (!goalTurn || validDSHGoalThoughtProof(proof, turn.DSHGoalID, progress, progressParts)) {
		parts = append(parts, dshRenderedParts(progressParts, progress)...)
	} else if progress != "" {
		parts = append(parts, map[string]string{"type": "text", "text": progress})
	}
	if session.Engine == "dsh" && turn.hasModelFailureRecord() {
		parts = append(parts, map[string]string{"type": "error", "reason": turn.FailureCode})
	}
	message := map[string]any{"id": session.responseID(index), "role": "assistant", "at": turn.Started.UnixMilli(), "parts": parts}
	if session.Engine == "dsh" {
		message["dshExecution"] = true
		if len(turn.DSHMemberNotices) != 0 {
			message["dshMemberNotices"] = turn.DSHMemberNotices
		}
		if len(turn.DSHInjected) != 0 {
			message["dshInjected"] = turn.DSHInjected
		}
		if turn.DSHTeamBoard.visible() && !slices.ContainsFunc(session.Turns[index+1:], func(later acpTurn) bool { return later.DSHTeamBoard != nil }) {
			message["dshTeamBoard"] = turn.DSHTeamBoard
		}
		if turn.Failed && !turn.DSHReconciled {
			message["dshFailed"] = true
			if turn.DSHStopObserved {
				message["dshStopAccountingPending"] = true
			}
		}
		if !turn.Ended.IsZero() {
			if files := turn.presentedFiles(session.CWD); len(files) > 0 {
				message["dshDeliverables"] = files
			}
		}
		// The client lists files changed in this conversation from the message.
		if files := turn.mutatedFiles(session.CWD); len(files) > 0 {
			message["files"] = files
		}
		if !turn.Started.IsZero() {
			message["startedAt"] = turn.Started.UnixMilli()
		}
		if turn.Ended.IsZero() && turn.DSHStep > 0 {
			message["dshStep"] = turn.DSHStep
		}
	}
	if turn.Interrupted && turn.DSHCancelConfirmed && (!turn.Failed || turn.DSHReconciled) {
		message["interrupted"] = true
	}
	if !turn.Ended.IsZero() {
		message["at"], message["endedAt"] = turn.Ended.UnixMilli(), turn.Ended.UnixMilli()
	}
	return message
}

func (session acpSession) historyWindow(query historyQuery, auth *relayAuthorization) (map[string]any, error) {
	count := session.historySequence()
	start, end, older, newer, err := query.window("source", count)
	if err != nil {
		return nil, err
	}
	messages := []map[string]any{}
	calls := map[string]any{}
	position := 0
	for index, turn := range session.Turns {
		if turn.DSHMemberID != "" {
			continue
		}
		userPosition := position
		position++
		if turn.hasHistoryResponse() {
			position++
		}
		if position <= start || userPosition >= end {
			continue
		}
		if userPosition >= start {
			messages = append(messages, session.promptMessage(index, auth))
		}
		if !turn.hasHistoryResponse() || userPosition+1 >= end {
			continue
		}
		messages = append(messages, session.responseMessage(index, calls))
	}
	model := session.Model
	if model == "" {
		model = modelID
	}
	metadata := session.workspaceMetadata()
	metadata["id"], metadata["model"] = session.ID, model
	metadata["createdAt"], metadata["updatedAt"] = session.Created, session.Updated
	return map[string]any{"session": metadata, "messages": messages,
		"toolCalls": calls, "eventSeq": count, "olderCursor": older, "newerCursor": newer}, nil
}
