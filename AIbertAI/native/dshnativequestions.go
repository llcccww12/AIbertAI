package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// The workbench answers questions in display order, whereas DSH requires the
// original question IDs. Keep the original request alongside the display data.
type dshQuestion struct {
	ID          string              `json:"id"`
	Question    string              `json:"question"`
	Header      string              `json:"header,omitempty"`
	Detail      string              `json:"detail,omitempty"`
	Options     []dshQuestionOption `json:"options"`
	MultiSelect bool                `json:"multiSelect,omitempty"`
	Intent      json.RawMessage     `json:"intent,omitempty"`
	DSHNative   bool                `json:"_dshNative,omitempty"`
}

type dshQuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type dshQuestionRequest struct {
	RequestID string        `json:"requestId"`
	SessionID string        `json:"sessionId"`
	Questions []dshQuestion `json:"questions"`
}

type dshQuestionKey struct{}
type dshQuestionHandler func(context.Context, json.RawMessage) (any, error)
type pendingQuestion struct {
	sessionID string
	cancel    context.CancelFunc
}
type dshQuestionReply struct {
	requestID string
	rpcID     json.RawMessage
	result    any
	err       error
	pending   *pendingQuestion
}

func (h *relaySessionHost) dshQuestionHandler(ws *websocket.Conn, httpRequest *http.Request, session acpSession) dshQuestionHandler {
	return func(ctx context.Context, raw json.RawMessage) (any, error) {
		question, err := parseDSHQuestionRequest(raw, session.DSHSessionID)
		if err != nil || h.acpScope(httpRequest.WithContext(ctx), "session:control") != nil {
			return nil, errors.New("dsh_question_unavailable")
		}
		if h.interactions != nil {
			return h.askDesktopDSHQuestion(ctx, ws, question)
		}
		waiters, ok := ctx.Value(acpPermissionContextKey{}).(*sync.Map)
		if !ok {
			return nil, errors.New("dsh_question_unavailable")
		}
		id := "dsh-question-" + question.RequestID
		waiter := &acpPermissionWaiter{sessionID: session.ID, reply: make(chan acpPermissionReply, 1)}
		if _, loaded := waiters.LoadOrStore(id, waiter); loaded {
			return nil, errors.New("dsh_question_unavailable")
		}
		defer waiters.Delete(id)
		defer func() {
			// A cancelled or finished SDK request must not leave a stale card.
			notifyCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			_ = wsjson.Write(notifyCtx, ws, map[string]any{"jsonrpc": "2.0", "method": "_monkeyai/session/request_cancelled", "params": map[string]any{"sessionId": session.ID, "requestId": question.RequestID}})
		}()
		if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": id, "method": "_monkeyai/session/question", "params": map[string]any{"sessionId": session.ID, "requestId": question.RequestID, "questions": question.Questions}}); err != nil {
			return nil, errors.New("dsh_question_unavailable")
		}
		viewerDone, _ := ctx.Value(acpViewerDoneKey{}).(<-chan struct{})
		select {
		case reply := <-waiter.reply:
			if len(reply.Error) != 0 || h.acpScope(httpRequest.WithContext(ctx), "session:control") != nil {
				return nil, errors.New("dsh_question_unavailable")
			}
			return dshQuestionAnswer(question, reply.Result)
		case <-viewerDone:
			return nil, errors.New("dsh_question_unavailable")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func parseDSHQuestionRequest(raw json.RawMessage, sessionID string) (dshQuestionRequest, error) {
	var request dshQuestionRequest
	if json.Unmarshal(raw, &request) != nil || request.SessionID != sessionID || request.RequestID == "" || len(request.Questions) == 0 {
		return request, errors.New("dsh_invalid_question")
	}
	seen := make(map[string]bool, len(request.Questions))
	for i := range request.Questions {
		q := &request.Questions[i]
		if seen[q.ID] {
			return request, errors.New("dsh_invalid_question")
		}
		seen[q.ID] = true
		q.DSHNative = true
		if q.Options == nil {
			q.Options = []dshQuestionOption{}
		}
	}
	return request, nil
}

func dshQuestionAnswer(request dshQuestionRequest, raw json.RawMessage) (any, error) {
	var response struct {
		Cancelled bool `json:"cancelled"`
		Meta      struct {
			Abandoned bool `json:"abandoned"`
		} `json:"_meta"`
		Answers []struct {
			Question string   `json:"question"`
			Selected []string `json:"selected"`
			Custom   bool     `json:"custom"`
		} `json:"answers"`
	}
	if json.Unmarshal(raw, &response) != nil || response.Meta.Abandoned {
		return nil, errors.New("dsh_invalid_question_answer")
	}
	if response.Cancelled {
		if len(response.Answers) != 0 {
			return nil, errors.New("dsh_invalid_question_answer")
		}
		return map[string]bool{"cancelled": true}, nil
	}
	if len(response.Answers) != len(request.Questions) {
		return nil, errors.New("dsh_invalid_question_answer")
	}
	answers := make([]map[string]any, 0, len(request.Questions))
	for i, q := range request.Questions {
		a := response.Answers[i]
		if a.Question != q.Question || !q.MultiSelect && len(a.Selected) > 1 || a.Custom && len(a.Selected) == 0 {
			return nil, errors.New("dsh_invalid_question_answer")
		}
		var intent struct {
			Kind string `json:"kind"`
		}
		if len(q.Intent) != 0 && json.Unmarshal(q.Intent, &intent) != nil {
			return nil, errors.New("dsh_invalid_question_answer")
		}
		if intent.Kind == "approval" && len(a.Selected) != 1 {
			return nil, errors.New("dsh_invalid_question_answer")
		}
		selected := a.Selected
		if selected == nil {
			selected = []string{}
		}
		answer := map[string]any{"id": q.ID}
		if a.Custom {
			if intent.Kind == "approval" || strings.TrimSpace(selected[len(selected)-1]) == "" {
				return nil, errors.New("dsh_invalid_question_answer")
			}
			answer["custom"] = selected[len(selected)-1]
			selected = selected[:len(selected)-1]
		}
		seen := map[string]bool{}
		for _, value := range selected {
			valid := false
			for _, option := range q.Options {
				valid = valid || value == option.Label
			}
			if !valid || seen[value] {
				return nil, errors.New("dsh_invalid_question_answer")
			}
			seen[value] = true
		}
		answer["selected"] = selected
		answers = append(answers, answer)
	}
	return map[string]any{"answers": answers}, nil
}
