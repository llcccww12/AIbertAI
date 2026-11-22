package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

// Pages are recomputed under normal session authorization. Every follow-up
// names the digest of the first page, so changing live output cannot be mixed.
func writeRelayHistoryResult(w http.ResponseWriter, auth relayAuthorization, page any, query historyQuery) {
	if !query.PageFragments {
		if query.PageOffset != nil || query.PageHash != "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		writeRelayResult(w, auth, page)
		return
	}
	data, err := json.Marshal(page)
	if err != nil || len(data) > maxDesktopLargeRecordBytes {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "session_read_too_large"}})
		return
	}
	hash := desktopHash(data)
	offset := 0
	if query.PageOffset != nil {
		offset = *query.PageOffset
		if !validDesktopHash(query.PageHash) || query.PageHash != hash {
			w.WriteHeader(http.StatusConflict)
			return
		}
	} else if query.PageHash != "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if offset < 0 || offset >= len(data) || offset%(512<<10) != 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	end := min(offset+(512<<10), len(data))
	writeRelayResult(w, auth, map[string]any{"_aibertaiFragment": 1, "sessionId": query.SessionID, "offset": offset, "total": len(data), "sha256": hash, "pageLimit": query.Limit, "data": base64.StdEncoding.EncodeToString(data[offset:end])})
}

// fitRelayDSHHistoryQuery keeps the record cursor meaningful while choosing
// the largest contiguous edge of a requested page that fits the byte budget.
// Messages are measured one at a time, avoiding a huge aggregate allocation.
func fitRelayDSHHistoryQuery(session acpSession, q historyQuery, auth *relayAuthorization) (historyQuery, error) {
	start, end, _, _, err := q.window("source", session.historySequence())
	if err != nil {
		return q, err
	}
	type position struct {
		turn     int
		response bool
	}
	positions := make([]position, 0, end-start)
	positionIndex := 0
	for i, turn := range session.Turns {
		if turn.DSHMemberID != "" {
			continue
		}
		if positionIndex >= start && positionIndex < end {
			positions = append(positions, position{turn: i})
		}
		positionIndex++
		if turn.hasHistoryResponse() {
			if positionIndex >= start && positionIndex < end {
				positions = append(positions, position{turn: i, response: true})
			}
			positionIndex++
		}
	}
	metadata := session.workspaceMetadata()
	model := session.Model
	if model == "" {
		model = modelID
	}
	metadata["id"], metadata["model"] = session.ID, model
	metadata["createdAt"], metadata["updatedAt"] = session.Created, session.Updated
	// A cursor at the largest position bounds both cursor encodings; per-item
	// accounting below includes separators and each message's tool call map.
	cursor := fmt.Sprintf("%s/source/%d", session.ID, session.historySequence())
	base, err := json.Marshal(map[string]any{"session": metadata, "messages": []any{}, "toolCalls": map[string]any{}, "eventSeq": session.historySequence(), "olderCursor": cursor, "newerCursor": cursor})
	if err != nil {
		return q, err
	}
	used := len(base)
	accepted := 0
	newer := q.Cursor != "" && q.Direction == "newer"
	for j := 0; j < len(positions); j++ {
		index := j
		if !newer {
			index = len(positions) - 1 - j
		}
		p := positions[index]
		calls := map[string]any{}
		var msg map[string]any
		if p.response {
			msg = session.responseMessage(p.turn, calls)
		} else {
			msg = session.promptMessage(p.turn, auth)
		}
		encoded, err := json.Marshal(msg)
		if err != nil {
			return q, err
		}
		callJSON, err := json.Marshal(calls)
		if err != nil {
			return q, err
		}
		size := len(encoded) + len(callJSON) + 2
		if used+size > maxDesktopLargeRecordBytes {
			break
		}
		used += size
		accepted++
	}
	if accepted > 0 && accepted < q.Limit {
		q.Limit = accepted
	}
	if accepted == 0 && len(positions) > 0 {
		q.Limit = 1
	}
	return q, nil
}
