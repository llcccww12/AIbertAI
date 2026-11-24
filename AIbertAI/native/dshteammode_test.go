package main

import (
	"context"
	"encoding/json"
	"testing"
)

func TestDSHAgentTeamCreationPreference(t *testing.T) {
	const op = "f22f14e9-a855-4761-b684-1cdb91b0cb30"
	dir := t.TempDir()
	live := &acpLive{dir: dir, cwd: dir, sessions: map[string]acpSession{}, operations: map[string]string{}}
	host := &relaySessionHost{live: live}
	off, allowed := false, true
	id, err := live.createSessionWithTeam(op, dir, "", modelID, expertSelection{}, nil, "", desktopMediaModels{}, true, &off, &allowed)
	if err != nil {
		t.Fatal(err)
	}
	read := func() desktopAgentTeamMode {
		t.Helper()
		v, err := host.sessionAgentTeam(t.Context(), json.RawMessage(`{"sessionId":"`+id+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		return v.(desktopAgentTeamMode)
	}
	if mode := read(); mode.Enabled || !mode.Available {
		t.Fatalf("explicit off lost: %+v", mode)
	}
	if _, err := host.sessionAgentTeam(t.Context(), json.RawMessage(`{"sessionId":"`+id+`","enabled":true}`)); err != nil || !read().Enabled {
		t.Fatalf("empty session did not accept on: %v", err)
	}
	row, err := live.load(id)
	if err != nil || row.workspaceMetadata()["agentTeamEnabled"] != true {
		t.Fatalf("team mode metadata did not persist: %v", err)
	}
	row.State = "running"
	if err := live.save(row); err != nil {
		t.Fatal(err)
	}
	if _, err := host.sessionAgentTeam(t.Context(), json.RawMessage(`{"sessionId":"`+id+`","enabled":false}`)); err == nil || err.Error() != "session_busy" {
		t.Fatalf("running session accepted mutation: %v", err)
	}
	if !read().Enabled {
		t.Fatal("busy session lost its committed selection")
	}
	for _, tc := range []struct {
		preset []string
		want   bool
	}{{nil, true}, {[]string{}, false}, {[]string{"tool-subagent"}, false}, {[]string{"tool-agent-team"}, true}} {
		if (desktopExpert{DSHPlugins: tc.preset}).allowsAgentTeam() != tc.want {
			t.Fatalf("wrong availability for preset %v", tc.preset)
		}
	}
}

func testDSHRestoredAgentTeamMode(t *testing.T, ctx context.Context, cfg *dshRuntimeConfig, row acpSession) {
	t.Helper()
	live := &acpLive{dsh: cfg, cwd: row.CWD, dir: t.TempDir(), sessions: map[string]acpSession{}}
	if err := live.save(row); err != nil {
		t.Fatal(err)
	}
	host := &relaySessionHost{live: live}
	mode := func(extra string, want bool) {
		t.Helper()
		value, err := host.sessionAgentTeam(ctx, json.RawMessage(`{"sessionId":"`+row.ID+`"`+extra+`}`))
		if err != nil || value != (desktopAgentTeamMode{Enabled: want, Available: true}) {
			t.Fatalf("SDK agent team %q returned %#v, %v", extra, value, err)
		}
	}
	// Reuse the leased root after a successful model turn.
	mode("", true)
	mode(`,"enabled":false`, false)
	if err := cfg.LeasePool.close(ctx); err != nil {
		t.Fatal(err)
	}
	cfg.LeasePool = &dshLeasePool{}
	// Remove host UI state: an old session must read its SDK journal after a restart.
	if err := live.save(row); err != nil {
		t.Fatal(err)
	}
	mode("", false)
	mode(`,"enabled":true`, true)
}
