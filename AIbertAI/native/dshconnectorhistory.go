package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

const dshConnectorHistoryMaxEvents = 16384
const dshConnectorHistoryMaxBytes = 32 << 20

// The fetcher must call the owning SDK child's aibertai/history RPC. That RPC
// flushes and reads back the validated persistent log, unlike notifications.
type dshConnectorHistoryFetch func(context.Context, map[string]any) (json.RawMessage, error)
type dshConnectorHistoryEvent struct {
	Seq             *int64              `json:"seq"`
	Type            string              `json:"type"`
	Data            json.RawMessage     `json:"data"`
	SourceEventSeqs []int64             `json:"sourceEventSeqs"`
	SurfaceOp       dshHistorySurfaceOp `json:"surfaceOp"`
}

type dshHistorySurfaceOp struct {
	Op       string `json:"op"`
	StartSeq *int64 `json:"startSeq"`
	EndSeq   *int64 `json:"endSeq"`
}

func (s *dshHistorySurfaceOp) UnmarshalJSON(raw []byte) error {
	if bytes.Equal(raw, []byte(`"append"`)) {
		*s = dshHistorySurfaceOp{Op: "append"}
		return nil
	}
	type replacement dshHistorySurfaceOp
	var r replacement
	if json.Unmarshal(raw, &r) != nil || r.Op != "replace" || r.StartSeq == nil || r.EndSeq == nil {
		return errDSHInvalidHistory
	}
	*s = dshHistorySurfaceOp(r)
	return nil
}

// The SDK pruner appends a price marker followed by a replacement of one
// existing result. All data except message.content is copied from its source.
func dshValidPrunedResult(events []dshConnectorHistoryEvent, i int) bool {
	e := events[i]
	if i < 2 || events[i-1].Type != "compaction/prune" || len(e.SourceEventSeqs) != 1 {
		return false
	}
	source := e.SourceEventSeqs[0]
	if source < 0 || source >= int64(i-1) || events[source].Type != "tool/result" ||
		e.SurfaceOp.StartSeq == nil || e.SurfaceOp.EndSeq == nil || *e.SurfaceOp.StartSeq != source || *e.SurfaceOp.EndSeq != source {
		return false
	}
	var price struct {
		ShadowedSeqs  []int64
		ShadowedRange struct{ Start, End *int64 }
	}
	if json.Unmarshal(events[i-1].Data, &price) != nil || len(price.ShadowedSeqs) != 1 || price.ShadowedSeqs[0] != source ||
		price.ShadowedRange.Start == nil || price.ShadowedRange.End == nil || *price.ShadowedRange.Start != source || *price.ShadowedRange.End != source {
		return false
	}
	identity := func(raw json.RawMessage) map[string]any {
		var data map[string]any
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		if dec.Decode(&data) != nil {
			return nil
		}
		msg, ok := data["message"].(map[string]any)
		if !ok {
			return nil
		}
		if _, ok := msg["content"].([]any); !ok {
			return nil
		}
		delete(msg, "content")
		return data
	}
	before := identity(events[source].Data)
	return before != nil && reflect.DeepEqual(before, identity(e.Data))
}

// Reject duplicate keys and invalid UTF-8; preserve numeric precision.
func strictDSHJSON(raw []byte) ([]byte, error) {
	if !utf8.Valid(raw) {
		return nil, errors.New("invalid_json")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var read func(int) (any, error)
	read = func(depth int) (any, error) {
		if depth > 128 {
			return nil, errors.New("json_depth_exceeded")
		}
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return token, nil
		}
		switch delim {
		case '{':
			obj := map[string]any{}
			for dec.More() {
				k, e := dec.Token()
				if e != nil {
					return nil, e
				}
				key, ok := k.(string)
				if !ok {
					return nil, errors.New("invalid_key")
				}
				if _, exists := obj[key]; exists {
					return nil, errors.New("duplicate_key")
				}
				v, e := read(depth + 1)
				if e != nil {
					return nil, e
				}
				obj[key] = v
			}
			end, e := dec.Token()
			if e != nil || end != json.Delim('}') {
				return nil, errors.New("invalid_object")
			}
			return obj, nil
		case '[':
			arr := []any{}
			for dec.More() {
				v, e := read(depth + 1)
				if e != nil {
					return nil, e
				}
				arr = append(arr, v)
			}
			end, e := dec.Token()
			if e != nil || end != json.Delim(']') {
				return nil, errors.New("invalid_array")
			}
			return arr, nil
		default:
			return nil, errors.New("invalid_delimiter")
		}
	}
	value, err := read(0)
	if err != nil {
		return nil, err
	}
	if _, err = dec.Token(); err != io.EOF {
		return nil, errors.New("trailing_json")
	}
	return json.Marshal(value)
}

// Only a complete fixed-ceiling durable history can uniquely bind operation,
// admitted user message, turn, tool call, exact arguments and tool result.
func dshMatchConnectorHistory(ctx context.Context, session, operation, call, name string, args json.RawMessage, fetch dshConnectorHistoryFetch) (json.RawMessage, error) {
	return dshMatchToolHistory(ctx, session, operation, call, name, args, fetch, false)
}

func dshMatchToolHistory(ctx context.Context, session, operation, call, name string, args json.RawMessage, fetch dshConnectorHistoryFetch, unavailableApproval bool) (json.RawMessage, error) {
	fail := func() (json.RawMessage, error) { return nil, errors.New("dsh_connector_history_unverified") }
	limit := 64 << 10
	if name == "write" || name == "edit" {
		limit = 512 << 10
	}
	if !validUUID(session) || !validUUID(operation) || call == "" || len(call) > 1024 || name == "" || fetch == nil || len(args) > limit {
		return fail()
	}
	canonicalArgs, err := strictDSHJSON(args)
	if err != nil || len(canonicalArgs) == 0 || canonicalArgs[0] != '{' {
		return fail()
	}
	var events []dshConnectorHistoryEvent
	var rawEvents []json.RawMessage
	_, err = dshRepairHistory(ctx, session, -1, fetch, func(_ int64, rawEvent json.RawMessage) error {
		var event dshConnectorHistoryEvent
		if json.Unmarshal(rawEvent, &event) != nil {
			return errDSHInvalidHistory
		}
		events = append(events, event)
		rawEvents = append(rawEvents, rawEvent)
		return nil
	})
	if err != nil {
		if errors.Is(err, errDSHInvalidHistory) || errors.Is(err, errDSHHistoryTooLarge) {
			return fail()
		}
		return nil, err
	}
	type message struct {
		ID     string `json:"id"`
		Role   string `json:"role"`
		Source struct {
			Kind      string `json:"kind"`
			Operation string `json:"aibertaiOperationId"`
			GoalID    string `json:"goalId"`
			Revision  *int64 `json:"revision"`
			Round     *int64 `json:"round"`
		} `json:"source"`
	}
	var admission json.RawMessage
	var admissionID string
	admissionSeq := int64(-1)
	admissionCount := 0
	var binding struct {
		OperationID   string `json:"operationId"`
		CommandID     string `json:"commandId"`
		CommandRunSeq *int64 `json:"commandRunSeq"`
		GoalID        string `json:"goalId"`
		GoalRevision  *int64 `json:"goalRevision"`
		GoalSeq       *int64 `json:"goalSeq"`
	}
	bindingCount := 0
	var bindingSeq int64
	for _, event := range events {
		if event.Type == "aibertai/query-binding" {
			var candidate struct {
				OperationID string `json:"operationId"`
			}
			if json.Unmarshal(event.Data, &candidate) != nil {
				return fail()
			}
			if candidate.OperationID == operation {
				bindingCount++
				if json.Unmarshal(event.Data, &binding) != nil {
					return fail()
				}
				bindingSeq = *event.Seq
			}
		}
		if event.Type != "agent/inbox/spliced" {
			continue
		}
		var splice struct {
			Inserted []json.RawMessage `json:"inserted"`
		}
		if json.Unmarshal(event.Data, &splice) != nil {
			return fail()
		}
		for _, raw := range splice.Inserted {
			var m message
			if json.Unmarshal(raw, &m) != nil {
				return fail()
			}
			if m.Source.Operation == operation {
				if m.Source.Kind != "user" || m.Role != "user" || m.ID == "" {
					return fail()
				}
				admissionCount++
				admissionID = m.ID
				admission = raw
				admissionSeq = *event.Seq
			}
		}
	}
	goalBinding := admissionCount == 0 && bindingCount == 1
	if !goalBinding && (admissionCount != 1 || bindingCount != 0) {
		return fail()
	}
	var admissionCanonical []byte
	if goalBinding {
		if binding.CommandID == "" || !strings.HasPrefix(binding.GoalID, "goal-") || !validUUID(strings.TrimPrefix(binding.GoalID, "goal-")) || binding.GoalRevision == nil || *binding.GoalRevision < 1 || binding.CommandRunSeq == nil || binding.GoalSeq == nil || *binding.CommandRunSeq < 0 || *binding.GoalSeq <= *binding.CommandRunSeq || bindingSeq <= *binding.GoalSeq || *binding.GoalSeq >= int64(len(events)) {
			return fail()
		}
		command := events[*binding.CommandRunSeq]
		goal := events[*binding.GoalSeq]
		var run struct{ CommandID, Name string }
		var change struct {
			Operation string
			Goal      struct {
				ID       string
				Revision *int64
				Phase    string
			}
		}
		if command.Type != "command/run" || json.Unmarshal(command.Data, &run) != nil || run.CommandID != binding.CommandID || run.Name != "goal" || goal.Type != "goal/change" || json.Unmarshal(goal.Data, &change) != nil || (change.Operation != "create" && change.Operation != "resume") || change.Goal.ID != binding.GoalID || change.Goal.Revision == nil || *change.Goal.Revision != *binding.GoalRevision || change.Goal.Phase != "active" {
			return fail()
		}
		var commandDone int
		for _, event := range events {
			if event.Type == "command/done" {
				var done struct{ CommandID, Kind string }
				if json.Unmarshal(event.Data, &done) != nil {
					return fail()
				}
				if done.CommandID == binding.CommandID {
					if done.Kind != "success" || *event.Seq <= *binding.GoalSeq || *event.Seq >= bindingSeq {
						return fail()
					}
					commandDone++
				}
			}
		}
		if commandDone != 1 {
			return fail()
		}
	} else {
		admissionCanonical, err = strictDSHJSON(admission)
		if err != nil {
			return fail()
		}
	}
	var activeTurn, activeStep, lastTurn, lastStep int64
	var turnOpen, stepOpen bool
	targetTurn, userSeq := int64(-1), int64(-1)
	userCount := 0
	seenTurns := map[int64]bool{}
	seenSteps := map[int64]map[int64]bool{}
	humanMessages := map[int64]int{}
	goalTurns := map[int64]int64{}
	for i, event := range events {
		switch event.Type {
		case "turn/start":
			var d struct{ Turn *int64 }
			if json.Unmarshal(event.Data, &d) != nil || d.Turn == nil || *d.Turn <= lastTurn || turnOpen || seenTurns[*d.Turn] {
				return fail()
			}
			activeTurn = *d.Turn
			lastTurn, lastStep = activeTurn, 0
			seenTurns[activeTurn] = true
			seenSteps[activeTurn] = map[int64]bool{}
			turnOpen = true
			stepOpen = false
		case "turn/end":
			var d struct{ Turn *int64 }
			if json.Unmarshal(event.Data, &d) != nil || d.Turn == nil || !turnOpen || stepOpen || *d.Turn != activeTurn {
				return fail()
			}
			turnOpen = false
			stepOpen = false
		case "step/start":
			var d struct{ Turn, Step *int64 }
			if json.Unmarshal(event.Data, &d) != nil || d.Turn == nil || d.Step == nil || !turnOpen || *d.Turn != activeTurn || *d.Step <= lastStep || stepOpen || seenSteps[activeTurn][*d.Step] {
				return fail()
			}
			activeStep = *d.Step
			lastStep = activeStep
			seenSteps[activeTurn][activeStep] = true
			stepOpen = true
		case "step/end":
			var d struct{ Turn, Step *int64 }
			if json.Unmarshal(event.Data, &d) != nil || d.Turn == nil || d.Step == nil || !turnOpen || !stepOpen || *d.Turn != activeTurn || *d.Step != activeStep {
				return fail()
			}
			stepOpen = false
		case "tool/call", "tool/result":
			if event.Type == "tool/result" && event.SurfaceOp.Op == "replace" {
				if !dshValidPrunedResult(events, i) {
					return fail()
				}
				continue
			}
			var d struct{ Turn, Step *int64 }
			if json.Unmarshal(event.Data, &d) != nil || d.Turn == nil || d.Step == nil || !turnOpen || !stepOpen || *d.Turn != activeTurn || *d.Step != activeStep {
				return fail()
			}
		case "user/message":
			var m message
			if json.Unmarshal(event.Data, &m) != nil {
				return fail()
			}
			if m.Source.Kind == "user" {
				if !turnOpen || !stepOpen || m.Role != "user" || m.ID == "" {
					return fail()
				}
				humanMessages[activeTurn]++
			}
			if goalBinding && *event.Seq > *binding.GoalSeq && m.Source.Kind == "goal" && m.Source.GoalID == binding.GoalID {
				if !turnOpen || !stepOpen || m.Role != "user" || m.ID == "" || m.Source.Revision == nil || *m.Source.Revision != *binding.GoalRevision || m.Source.Round == nil || *m.Source.Round < 1 {
					return fail()
				}
				if _, duplicate := goalTurns[activeTurn]; duplicate {
					return fail()
				}
				goalTurns[activeTurn] = *event.Seq
			}
			if m.Source.Operation == operation || m.ID == admissionID {
				if !turnOpen || !stepOpen || *event.Seq <= admissionSeq || m.ID != admissionID || m.Source.Kind != "user" || m.Role != "user" || m.Source.Operation != operation {
					return fail()
				}
				got, e := strictDSHJSON(event.Data)
				if e != nil || !bytes.Equal(got, admissionCanonical) {
					return fail()
				}
				userCount++
				targetTurn = activeTurn
				userSeq = *event.Seq
			}
		}
	}
	if goalBinding {
		matches := 0
		for _, event := range events {
			if event.Type != "tool/call" {
				continue
			}
			var d struct {
				Turn   *int64
				CallID string `json:"callId"`
			}
			if json.Unmarshal(event.Data, &d) != nil || d.Turn == nil {
				return fail()
			}
			if seq, ok := goalTurns[*d.Turn]; ok && d.CallID == call {
				targetTurn, userSeq = *d.Turn, seq
				matches++
			}
		}
		if matches != 1 || targetTurn < 1 || humanMessages[targetTurn] != 0 {
			return fail()
		}
	} else if userCount != 1 || targetTurn < 1 || humanMessages[targetTurn] != 1 {
		return fail()
	}
	callSeq, callStep := int64(-1), int64(-1)
	callCount, resultCount := 0, 0
	resultIndex := -1
	for i, event := range events {
		switch event.Type {
		case "tool/call":
			var d struct {
				Turn, Step      *int64
				CallID          string `json:"callId"`
				Name, Arguments string
			}
			if json.Unmarshal(event.Data, &d) != nil || d.Turn == nil || d.Step == nil {
				return fail()
			}
			if *d.Turn != targetTurn || d.CallID != call {
				continue
			}
			callCount++
			if *event.Seq <= userSeq || *d.Step < 1 || d.Name != name {
				return fail()
			}
			got, e := strictDSHJSON([]byte(d.Arguments))
			if e != nil || !bytes.Equal(got, canonicalArgs) {
				return fail()
			}
			callSeq, callStep = *event.Seq, *d.Step
		case "tool/result":
			// Only the original execution result may settle a host side effect.
			if event.SurfaceOp.Op == "replace" {
				continue
			}
			var d struct {
				Turn, Step *int64
				Message    struct {
					Role, ToolCallID string
					IsError          bool `json:"isError"`
					Source           struct{ Kind, CallID string }
				}
			}
			if json.Unmarshal(event.Data, &d) != nil || d.Turn == nil || d.Step == nil {
				return fail()
			}
			if *d.Turn != targetTurn || d.Message.Source.CallID != call && d.Message.ToolCallID != call {
				continue
			}
			resultCount++
			if d.Message.Role != "tool" || d.Message.IsError != unavailableApproval || d.Message.Source.Kind != "tool" || d.Message.Source.CallID != call || d.Message.ToolCallID != call {
				return fail()
			}
			resultIndex = i
		}
	}
	if callCount != 1 || resultCount != 1 || resultIndex < 0 {
		return fail()
	}
	if goalBinding {
		for _, event := range events {
			if *event.Seq <= *binding.GoalSeq || *event.Seq >= callSeq {
				continue
			}
			if event.Type == "goal/change" {
				var change struct {
					Operation string
					Goal      struct {
						ID       string
						Revision *int64
						Phase    string
					}
					Cleared struct{ ID string }
				}
				if json.Unmarshal(event.Data, &change) != nil {
					return fail()
				}
				if change.Cleared.ID == binding.GoalID || change.Goal.ID == binding.GoalID && (change.Goal.Revision == nil || *change.Goal.Revision != *binding.GoalRevision || change.Goal.Phase != "active") {
					return fail()
				}
			}
			if event.Type == "aibertai/query-binding" && *event.Seq != bindingSeq {
				var other struct {
					GoalID string `json:"goalId"`
				}
				if json.Unmarshal(event.Data, &other) != nil || other.GoalID == binding.GoalID {
					return fail()
				}
			}
		}
	}
	result := events[resultIndex]
	var d struct{ Turn, Step *int64 }
	if json.Unmarshal(result.Data, &d) != nil || d.Step == nil || *d.Step != callStep || *result.Seq <= callSeq || len(result.SourceEventSeqs) != 1 || result.SourceEventSeqs[0] != callSeq {
		return fail()
	}
	if unavailableApproval && !dshAbortedApprovalTail(events, resultIndex, callSeq, targetTurn, callStep, call) {
		return fail()
	}
	return json.Marshal(map[string]any{"sessionId": session, "event": rawEvents[resultIndex]})
}
