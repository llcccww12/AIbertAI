package main

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func checkDSHMemberView(t *testing.T, ctx context.Context, cfg *dshRuntimeConfig, row acpSession, warm bool) {
	t.Helper()
	host := &relaySessionHost{live: &acpLive{dsh: cfg, store: &desktopSessionStore{}, sessions: map[string]acpSession{row.ID: row}}}
	call := func(id string, from *int64) (map[string]json.RawMessage, error) {
		req, _ := json.Marshal(dshMemberRequest{SessionID: row.ID, MemberID: id, From: from})
		v, err := host.sessionMembers(ctx, req)
		if err != nil {
			return nil, err
		}
		return v.(map[string]json.RawMessage), nil
	}
	list, err := call("", nil)
	if err != nil {
		t.Fatal(err)
	}
	var members []struct{ ID, Name, ParentID string }
	var live bool
	if json.Unmarshal(list["live"], &live) != nil || live != warm || json.Unmarshal(list["members"], &members) != nil || len(members) != 1 || members[0].Name != "reviewer" || members[0].ParentID != row.ID {
		t.Fatalf("invalid member list: %s", list)
	}
	var from int64
	type entry struct {
		Type, Text, Tool, CallID, Source string
		Turn, Step, Time                 int64
	}
	var entries []entry
	var text strings.Builder
	for page := 0; page < 24; page++ {
		detail, err := call(members[0].ID, &from)
		if err != nil {
			t.Fatal(err)
		}
		text.Write(detail["entries"])
		var batch []entry
		if err := json.Unmarshal(detail["entries"], &batch); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, batch...)
		var more bool
		var next int64
		if json.Unmarshal(detail["more"], &more) != nil || json.Unmarshal(detail["next"], &next) != nil || more && next <= from {
			t.Fatal("invalid member cursor")
		}
		from = next
		if !more {
			break
		}
	}
	if !strings.Contains(text.String(), "worker-isolated-prompt") || !strings.Contains(text.String(), "team_task_update") || strings.Contains(text.String(), "Coordinate our isolated fixture.") {
		t.Fatalf("member execution missing or included root transcript: %s", text.String())
	}
	calls := map[string]entry{}
	matched := map[string]bool{}
	var tasks []string
	for _, item := range entries {
		if item.Time <= 0 {
			t.Fatal("missing event time")
		}
		switch item.Type {
		case "user/message":
			if item.Source == "user" {
				tasks = append(tasks, item.Text)
			}
			if item.Source != "user" && item.Source != "agent-message" {
				t.Fatalf("injected message rendered as task: %q", item.Source)
			}
		case "tool/call":
			if item.CallID == "" || item.Tool == "" || item.Turn < 1 || item.Step < 1 {
				t.Fatal("missing tool identity")
			}
			calls[item.CallID] = item
		case "tool/result":
			req, ok := calls[item.CallID]
			if !ok || matched[item.CallID] || req.Turn != item.Turn || req.Step != item.Step {
				t.Fatal("unpaired tool result")
			}
			matched[item.CallID] = true
		}
	}
	if !slices.Equal(tasks, []string{"worker-isolated-prompt"}) || len(matched) != len(calls) {
		t.Fatal("unexpected tasks or incomplete tool pairs")
	}
	if _, e := call(row.ID, nil); e == nil {
		t.Fatal("root session accepted as its own member")
	}
}
