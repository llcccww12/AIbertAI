package main

import (
	"encoding/json"
	"strings"
)

// A candidate remains transient until the gateway has accepted its committed
// model receipt. It is associated with declared tool IDs, never a tool index.
type dshGoalThoughtCall struct{ id, name, arguments string }

type dshGoalThoughtCandidate struct {
	proof         dshGoalThoughtProof
	text          string
	parts         []dshMessagePart
	calls         []dshGoalThoughtCall
	turn, step    int64
	input, output int64
	commandID     string
}

type dshGoalThoughtTracker struct {
	commandID                                string
	commandSeq, goalSeq, doneSeq, bindingSeq int64
	goalID, query                            string
	revision, turn, step                     int64
	target, turnOpen, stepOpen               bool
	goalSeen, human, preTargetActivity       bool
	invocation                               string
	seen                                     map[string]bool
	candidates                               map[string]dshGoalThoughtCandidate
	boundSeen, invalid                       bool
}

func (s *dshGoalThoughtTracker) event(raw json.RawMessage, row acpSession) {
	var e struct {
		Event struct {
			Type string          `json:"type"`
			Seq  *int64          `json:"seq"`
			Data json.RawMessage `json:"data"`
		} `json:"event"`
	}
	if row.Goal == nil || json.Unmarshal(raw, &e) != nil || e.Event.Seq == nil || *e.Event.Seq < 0 {
		return
	}
	if s.invalid {
		return
	}
	seq := *e.Event.Seq
	switch e.Event.Type {
	case "command/run":
		var d struct{ CommandID, Name string }
		if json.Unmarshal(e.Event.Data, &d) == nil && d.Name == "goal" && d.CommandID != "" {
			s.commandID, s.commandSeq, s.goalSeq, s.doneSeq, s.bindingSeq = d.CommandID, seq, -1, -1, -1
			s.query, s.target, s.turnOpen, s.stepOpen = "", false, false, false
		}
	case "goal/change":
		var d struct {
			Operation string
			Goal      struct {
				ID, Phase string
				Revision  int64
			}
		}
		if json.Unmarshal(e.Event.Data, &d) != nil {
			s.query, s.target = "", false
			return
		}
		if s.query != "" && (d.Goal.ID != s.goalID || d.Goal.Revision < s.revision || d.Goal.Revision > row.Goal.DSHRevision || (d.Operation != "pause" && d.Operation != "block" && d.Operation != "complete")) {
			s.invalid, s.candidates = true, nil
			return
		}
		if s.commandID != "" && s.bindingSeq < 0 && (d.Operation == "create" || d.Operation == "resume") && d.Goal.ID == row.Goal.DSHGoalID && d.Goal.Phase == "active" && d.Goal.Revision > 0 && d.Goal.Revision <= row.Goal.DSHRevision {
			s.goalSeq, s.goalID, s.revision = seq, d.Goal.ID, d.Goal.Revision
		}
	case "command/done":
		var d struct{ CommandID, Kind string }
		if json.Unmarshal(e.Event.Data, &d) == nil && d.CommandID == s.commandID && d.Kind == "success" && s.goalSeq > s.commandSeq {
			s.doneSeq = seq
		}
	case "aibertai/query-binding":
		var b dshGoalQueryBinding
		if json.Unmarshal(e.Event.Data, &b) != nil {
			s.invalid = true
			s.candidates = nil
			return
		}
		if b.OperationID == row.Goal.DSHQueryOp || row.Goal.DSHCommandID != "" && b.CommandID == row.Goal.DSHCommandID {
			if s.boundSeen || !validUUID(row.Goal.DSHQueryOp) || b.OperationID != row.Goal.DSHQueryOp || b.CommandID != s.commandID || b.CommandRunSeq != s.commandSeq || b.GoalSeq != s.goalSeq || b.GoalID != s.goalID || b.GoalRevision != s.revision || s.doneSeq <= s.goalSeq || seq <= s.doneSeq {
				s.invalid, s.candidates = true, nil
				return
			}
			s.boundSeen, s.bindingSeq, s.query = true, seq, b.OperationID
		}
	case "turn/start":
		var d struct{ Turn int64 }
		if json.Unmarshal(e.Event.Data, &d) == nil && s.bindingSeq >= 0 && seq > s.bindingSeq && d.Turn > s.turn && !s.turnOpen && !s.stepOpen {
			s.turn, s.step, s.target, s.turnOpen, s.stepOpen, s.invocation = d.Turn, 0, false, true, false, ""
			s.goalSeen, s.human, s.preTargetActivity = false, false, false
		} else {
			s.target, s.turnOpen, s.stepOpen, s.invocation = false, false, false, ""
		}
	case "turn/end":
		var d struct{ Turn int64 }
		if json.Unmarshal(e.Event.Data, &d) != nil || d.Turn != s.turn || !s.turnOpen || s.stepOpen {
			s.query = ""
		}
		s.target, s.turnOpen, s.stepOpen, s.invocation = false, false, false, ""
	case "step/start":
		var d struct{ Turn, Step int64 }
		if json.Unmarshal(e.Event.Data, &d) == nil && s.turnOpen && s.turn > 0 && d.Turn == s.turn && d.Step > s.step && !s.stepOpen {
			s.step, s.stepOpen, s.invocation = d.Step, true, ""
		} else {
			s.target, s.stepOpen, s.invocation = false, false, ""
		}
	case "step/end":
		var d struct{ Turn, Step int64 }
		if json.Unmarshal(e.Event.Data, &d) != nil || !s.stepOpen || d.Turn != s.turn || d.Step != s.step {
			s.target = false
		}
		s.stepOpen, s.invocation = false, ""
	case "user/message":
		var d struct {
			Role   string
			Source struct {
				Kind, GoalID string
				Revision     int64
				Round        int
			}
		}
		if json.Unmarshal(e.Event.Data, &d) != nil {
			s.target = false
			return
		}
		if d.Source.Kind == "goal" {
			s.target = !s.goalSeen && !s.human && !s.preTargetActivity && s.stepOpen && d.Role == "user" && d.Source.GoalID == s.goalID && d.Source.Revision == s.revision && d.Source.Round > 0 && d.Source.Round <= row.Goal.RoundsStarted
			s.goalSeen = true
		} else if d.Source.Kind == "user" {
			s.human, s.target = true, false
		} else if s.target && d.Source.Kind != "runtime-context" && d.Source.Kind != "skill-catalog" && d.Source.Kind != "time-context" && d.Source.Kind != "tool-goal" {
			s.target = false
		}
	case "tool/call", "tool/result":
		if s.turnOpen && !s.target {
			s.preTargetActivity = true
		}
	case "aibertai/invocation":
		if s.turnOpen && !s.target {
			s.preTargetActivity = true
		}
		var d struct{ InvocationID, Purpose string }
		if json.Unmarshal(e.Event.Data, &d) != nil || !s.target || !s.stepOpen || d.Purpose != "conversation" || !validUUID(d.InvocationID) || s.invocation != "" {
			return
		}
		if s.seen == nil {
			s.seen = make(map[string]bool)
		}
		if s.seen[d.InvocationID] {
			s.invocation = ""
			return
		}
		s.seen[d.InvocationID], s.invocation = true, d.InvocationID
	case "assistant/message":
		if s.turnOpen && !s.target {
			s.preTargetActivity = true
		}
		var d struct {
			Turn, Step                     int64
			InvocationID, InvocationFinish string
			Usage                          *struct {
				Input  *int64 `json:"inputTokens"`
				Output *int64 `json:"outputTokens"`
			}
			Message struct {
				ID, Role string
				Source   struct{ Kind string }
				Content  []struct{ Type, Text, ID, Name, Arguments string }
			}
		}
		if json.Unmarshal(e.Event.Data, &d) != nil || !s.target || !s.stepOpen || d.Turn != s.turn || d.Step != s.step || d.InvocationID != s.invocation || d.Message.ID == "" || d.Message.Role != "assistant" || d.Message.Source.Kind != "model" || s.query == "" || d.Usage == nil || d.Usage.Input == nil || d.Usage.Output == nil || *d.Usage.Input < 0 || *d.Usage.Output < 0 {
			return
		}
		candidate := dshGoalThoughtCandidate{proof: dshGoalThoughtProof{GoalID: row.Goal.ID, QueryOperation: s.query, InvocationID: d.InvocationID, MessageSeq: seq}, turn: d.Turn, step: d.Step, input: *d.Usage.Input, output: *d.Usage.Output, commandID: s.commandID}
		var text strings.Builder
		hasThought := false
		for _, block := range d.Message.Content {
			if block.Type == "reasoning" {
				hasThought = true
			}
		}
		for _, block := range d.Message.Content {
			switch block.Type {
			case "reasoning":
				candidate.parts = append(candidate.parts, dshMessagePart{Type: "thought", Text: block.Text})
			case "text":
				text.WriteString(block.Text)
				if hasThought {
					candidate.parts = append(candidate.parts, dshMessagePart{Type: "text", End: text.Len()})
				}
			case "tool-call":
				if block.ID == "" {
					return
				}
				candidate.calls = append(candidate.calls, dshGoalThoughtCall{id: block.ID, name: block.Name, arguments: block.Arguments})
			}
		}
		if !hasThought || (len(candidate.calls) > 0 && d.InvocationFinish != "tool-calls") || (len(candidate.calls) == 0 && d.InvocationFinish != "stop") {
			return
		}
		candidate.text = text.String()
		candidate.proof.ContentSHA256 = dshGoalThoughtDigest(candidate.text, candidate.parts)
		if s.candidates == nil {
			s.candidates = make(map[string]dshGoalThoughtCandidate)
		}
		if _, duplicate := s.candidates[d.InvocationID]; duplicate {
			delete(s.candidates, d.InvocationID)
			return
		}
		s.candidates[d.InvocationID] = candidate
	}
}
