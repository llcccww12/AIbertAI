package main

import (
	"encoding/json"
	"slices"
	"unicode/utf8"
)

// dshContextEntry is one context injection the SDK added to model history in a
// turn (runtime-context, skill-catalog, time-context, agent instructions, ...).
// DSH shows these as "已注入上下文" rows; the transcript keeps a bounded copy.
type dshContextEntry struct {
	ID         string          `json:"id"`
	Time       int64           `json:"time"`
	Source     json.RawMessage `json:"source"`
	Text       string          `json:"text"`
	Truncated  bool            `json:"truncated,omitempty"`
	AfterTools int             `json:"afterTools"`
}

const (
	dshContextTextLimit   = 4 << 10
	dshContextSourceLimit = 2 << 10
	dshContextTurnLimit   = 8
)

// readDSHContextInjection mirrors DSH's chat projection: an appended
// user/message whose source is not the user is injected context, as is every
// developer/message. Member notices, compaction checkpoints and empty retry
// markers have their own presentation.
func readDSHContextInjection(raw json.RawMessage) (dshContextEntry, bool) {
	type message struct {
		ID      string
		Source  json.RawMessage
		Content []struct{ Type, Text string }
	}
	var frame struct {
		Event struct {
			Type      string
			Time      int64
			SurfaceOp dshHistorySurfaceOp
			Data      struct {
				ID      string
				Source  json.RawMessage
				Content []struct{ Type, Text string }
				Message *message
			}
		}
	}
	if json.Unmarshal(raw, &frame) != nil {
		return dshContextEntry{}, false
	}
	e := frame.Event
	m := message{ID: e.Data.ID, Source: e.Data.Source, Content: e.Data.Content}
	switch e.Type {
	case "user/message":
		if e.SurfaceOp.Op != "append" {
			return dshContextEntry{}, false
		}
	case "developer/message":
		if e.Data.Message == nil {
			return dshContextEntry{}, false
		}
		m = *e.Data.Message
	default:
		return dshContextEntry{}, false
	}
	var source struct{ Kind string }
	if m.ID == "" || json.Unmarshal(m.Source, &source) != nil || source.Kind == "" {
		return dshContextEntry{}, false
	}
	switch source.Kind {
	case "user", "compact-checkpoint", "subagent-settled", "agent-message":
		return dshContextEntry{}, false
	case "retry":
		if len(m.Content) == 0 {
			return dshContextEntry{}, false
		}
	}
	entry := dshContextEntry{ID: m.ID, Time: e.Time, Source: m.Source}
	if len(entry.Source) > dshContextSourceLimit {
		entry.Source, _ = json.Marshal(map[string]string{"kind": source.Kind})
	}
	for _, b := range m.Content {
		if b.Type != "text" || b.Text == "" {
			continue
		}
		if entry.Text != "" {
			entry.Text += "\n"
		}
		entry.Text += b.Text
	}
	if len(entry.Text) > dshContextTextLimit {
		cut := dshContextTextLimit
		for cut > 0 && !utf8.RuneStart(entry.Text[cut]) {
			cut--
		}
		entry.Text, entry.Truncated = entry.Text[:cut], true
	}
	return entry, true
}

func (turn *acpTurn) addContextInjection(entry dshContextEntry) bool {
	if len(turn.DSHInjected) >= dshContextTurnLimit || slices.ContainsFunc(turn.DSHInjected, func(prior dshContextEntry) bool { return prior.ID == entry.ID }) {
		return false
	}
	turn.DSHInjected = append(slices.Clip(turn.DSHInjected), entry)
	return true
}
