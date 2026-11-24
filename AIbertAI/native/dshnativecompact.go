package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
)

// Bind the command observed on the private SDK connection to the already
// reserved platform operation before accepting its outcome.
func dshCompactOptions(session acpSession, turn *acpTurn, project func(map[string]any) error) dshSDKOptions {
	fail := func() error { return errors.New("dsh_compaction_unverified") }
	var command string
	opts := dshSDKOptions{Query: true}
	opts.Observe = func(method string, raw json.RawMessage) error {
		if method != "session.event" {
			return nil
		}
		var frame struct {
			SessionID string
			Event     struct {
				Type string
				Data struct {
					CommandID, Name, Args string
					Source                struct{ Kind string }
				}
			}
		}
		if json.Unmarshal(raw, &frame) != nil || frame.SessionID != session.DSHSessionID {
			return fail()
		}
		if frame.Event.Type != "command/run" {
			return nil
		}
		data := frame.Event.Data
		if command != "" || !strings.HasPrefix(data.CommandID, "cmd-") || len(data.CommandID) > 256 || data.Name != "compact" || strings.TrimSpace(data.Args) != "" || data.Source.Kind != "user" {
			return fail()
		}
		command = data.CommandID
		turn.DSHCompactID = command
		if project != nil {
			return project(nil)
		}
		return nil
	}
	opts.OnCommand = func(id string) error {
		if id == "" || id != command {
			return fail()
		}
		return nil
	}
	opts.VerifyCommand = func(ctx context.Context, id, text string, fetch dshConnectorHistoryFetch) error {
		if id != command || id == "" {
			return fail()
		}
		var runSeq, doneSeq, summarySeq, startSeq, endSeq int64 = -1, -1, -1, -1, -1
		var startID, summaryID, endID string
		var doneSource *int64
		var doneText, doneKind string
		countRun, countDone, countSummary, countStart, countEnd := 0, 0, 0, 0, 0
		_, err := dshRepairHistory(ctx, session.DSHSessionID, -1, fetch, func(_ int64, raw json.RawMessage) error {
			var e struct {
				Seq  *int64
				Type string
				Data struct {
					CommandID, Name, Args, Kind, Text, SourceCommandID, CompactionID string
					SourceEventSeq                                                   *int64
					Source                                                           struct{ Kind string }
				}
			}
			if json.Unmarshal(raw, &e) != nil || e.Seq == nil {
				return fail()
			}
			switch {
			case e.Type == "command/run" && e.Data.CommandID == id:
				countRun++
				runSeq = *e.Seq
				if e.Data.Name != "compact" || strings.TrimSpace(e.Data.Args) != "" || e.Data.Source.Kind != "user" {
					return fail()
				}
			case e.Type == "command/done" && e.Data.CommandID == id:
				countDone++
				doneSeq = *e.Seq
				doneSource = e.Data.SourceEventSeq
				doneKind = e.Data.Kind
				doneText = e.Data.Text
			case e.Type == "compaction/start" && e.Data.SourceCommandID == id:
				countStart++
				startSeq = *e.Seq
				startID = e.Data.CompactionID
			case e.Type == "compaction/end" && e.Data.SourceCommandID == id:
				countEnd++
				endSeq = *e.Seq
				endID = e.Data.CompactionID
			case e.Type == "compaction/summary" && e.Data.SourceCommandID == id:
				countSummary++
				summarySeq = *e.Seq
				summaryID = e.Data.CompactionID
			}
			return nil
		})
		if err != nil || countRun != 1 || countDone != 1 || doneSeq <= runSeq || doneKind != "success" || doneText != text {
			return fail()
		}
		if countSummary == 0 && countStart == 0 && countEnd == 0 && doneSource == nil && doneText == "No compactable history yet." {
			return nil
		}
		if countStart != 1 || countSummary != 1 || countEnd != 1 || !validUUID(startID) || startID != summaryID || startID != endID || doneSource == nil || *doneSource != summarySeq || startSeq <= runSeq || summarySeq <= startSeq || endSeq <= summarySeq || doneSeq <= endSeq {
			return fail()
		}
		return nil
	}
	return opts
}
