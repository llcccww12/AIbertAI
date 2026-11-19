package main

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"unicode/utf16"
)

var errHistoryCursor = errors.New("invalid_session_history_cursor")

type historyQuery struct {
	SessionID  string `json:"sessionId"`
	Limit      int    `json:"limit"`
	Cursor     string `json:"cursor"`
	Direction  string `json:"direction"`
	Action     string `json:"action"`
	ToolCallID string `json:"toolCallId"`
}

var errHistoryToolOutput = errors.New("session_tool_output_unavailable")

func toolOutput(output string) map[string]any {
	length := 0
	for _, r := range output {
		length += utf16.RuneLen(r)
	}
	return map[string]any{"output": output, "length": length}
}

func (h *relaySessionHost) historyToolOutput(q historyQuery) (any, error) {
	if !validUUID(q.SessionID) || q.ToolCallID == "" || len(q.ToolCallID) > 256 {
		return nil, errHistoryToolOutput
	}
	if h.live != nil {
		if session, err := h.live.load(q.SessionID); err == nil {
			for index, turn := range session.Turns {
				for step, run := range turn.workspaceTools() {
					if session.toolHistoryID(turn, index, step) == q.ToolCallID && run.Report.Failed == 0 {
						return toolOutput(run.Output), nil
					}
				}
			}
			return nil, errHistoryToolOutput
		}
	}
	if h.local == nil && !h.shareDesktopSessions && h.sessionID != q.SessionID {
		return nil, errHistoryToolOutput
	}
	archive, err := readLegacySessionArchive(h.profile, q.SessionID)
	if err != nil {
		return nil, err
	}
	for index := len(archive.Replay) - 1; index >= 0; index-- {
		var row struct{ ToolCalls map[string]json.RawMessage }
		if json.Unmarshal(archive.Replay[index], &row) != nil {
			return nil, errInvalidSessionArchive
		}
		raw, ok := row.ToolCalls[q.ToolCallID]
		if !ok {
			continue
		}
		var call struct {
			ID, Status      string
			Output          *string
			OutputTruncated bool
			OutputLength    *int
		}
		if json.Unmarshal(raw, &call) != nil || call.ID != q.ToolCallID || call.Status != "completed" ||
			call.Output == nil || call.OutputTruncated {
			return nil, errHistoryToolOutput
		}
		result := toolOutput(*call.Output)
		if call.OutputLength != nil && *call.OutputLength != result["length"] {
			return nil, errHistoryToolOutput
		}
		return result, nil
	}
	return nil, errHistoryToolOutput
}

func (q *historyQuery) validate() bool {
	if !validUUID(q.SessionID) || q.Limit < 0 || q.Limit > 100 ||
		(q.Direction != "" && q.Direction != "older" && q.Direction != "newer") {
		return false
	}
	if q.Limit == 0 {
		q.Limit = 100
	}
	return true
}

// Cursors address the boundary between messages, in either direction.
// An archive's replay generation changes when the old runtime rebuilds it.
func (q historyQuery) window(generation string, total int) (start, end int, older, newer *string, err error) {
	prefix := q.SessionID + "/" + generation + "/"
	boundary := total
	if q.Cursor != "" {
		raw, ok := strings.CutPrefix(q.Cursor, prefix)
		value, parseErr := strconv.Atoi(raw)
		if !ok || parseErr != nil || value < 0 || value > total || raw != strconv.Itoa(value) {
			return 0, 0, nil, nil, errHistoryCursor
		}
		boundary = value
	}
	start, end = max(0, boundary-q.Limit), boundary
	if q.Cursor != "" && q.Direction == "newer" {
		start, end = boundary, min(total, boundary+q.Limit)
	}
	if start > 0 {
		older = new(prefix + strconv.Itoa(start))
	}
	if end < total {
		newer = new(prefix + strconv.Itoa(end))
	}
	return start, end, older, newer, nil
}
