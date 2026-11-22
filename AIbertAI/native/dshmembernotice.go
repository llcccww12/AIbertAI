package main

import (
	"encoding/json"
	"slices"
	"strings"
)

type dshMemberNotice struct {
	ID             string `json:"id"`
	MemberID       string `json:"memberId"`
	Kind           string `json:"kind"`
	StopReason     string `json:"stopReason,omitempty"`
	Text           string `json:"text"`
	Time           int64  `json:"time"`
	AfterTools     int    `json:"afterTools"`
	WokeParent     bool   `json:"wokeParent,omitempty"`
	PreviousAnswer string `json:"previousAnswer,omitempty"`
}

// Only an appended SDK message consumed by the parent establishes delivery.
// A member catalog status or model-authored description is not a receipt.
func readDSHMemberNotice(raw json.RawMessage) (dshMemberNotice, bool) {
	var frame struct {
		Event struct {
			Type      string
			Time      int64
			SurfaceOp dshHistorySurfaceOp
			Data      struct {
				ID, Role string
				Source   struct{ Kind, SenderSessionID, StopReason string }
				Content  []struct{ Type, Text string }
			}
		}
	}
	if json.Unmarshal(raw, &frame) != nil {
		return dshMemberNotice{}, false
	}
	e := frame.Event
	s := e.Data.Source
	if e.Type != "user/message" || e.SurfaceOp.Op != "append" || e.Data.Role != "user" || e.Data.ID == "" || !validUUID(s.SenderSessionID) || e.Time <= 0 || e.Time > 8640000000000000 || (s.Kind != "subagent-settled" && s.Kind != "agent-message") {
		return dshMemberNotice{}, false
	}
	var body strings.Builder
	for _, b := range e.Data.Content {
		if b.Type == "text" {
			if body.Len() > 0 {
				body.WriteByte('\n')
			}
			body.WriteString(b.Text)
		}
	}
	return dshMemberNotice{ID: e.Data.ID, MemberID: s.SenderSessionID, Kind: s.Kind, StopReason: s.StopReason, Time: e.Time, Text: body.String()}, true
}

func (turn *acpTurn) addMemberNotice(n dshMemberNotice) bool {
	if slices.ContainsFunc(turn.DSHMemberNotices, func(prior dshMemberNotice) bool { return prior.ID == n.ID }) {
		return false
	}
	turn.DSHMemberNotices = append(slices.Clip(turn.DSHMemberNotices), n)
	return true
}
