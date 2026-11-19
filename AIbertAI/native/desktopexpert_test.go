package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrivateExpertInstructionsAndSessionSelection(t *testing.T) {
	const id = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const skill = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	const connector = "c3bd985c-9f71-4b85-937a-c41b1b104d38"
	var pkg bytes.Buffer
	w := zip.NewWriter(&pkg)
	f, err := w.Create("fixture/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("Expert skill fixture")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(pkg.Bytes()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-account" {
			t.Error("expert request lost account authorization")
		}
		switch r.URL.Path {
		case "/api/v1/experts/" + id + "/manifest":
			_, _ = fmt.Fprintf(w, `{"expert_id":%q,"name":"Test expert","available":true,
			"prompt":"Follow the expert prompt","rules":[{"content":"Expert rule fixture"}],
			"skills":[{"id":%q,"package_sha256":%q}],
			"connectors":[{"id":%q,"required":true,"tool_allowlist":["read"],"tool_denylist":["write"]}]}`, id, skill, digest, connector)
		case "/api/v1/experts/" + id + "/skills/" + skill + "/package":
			if r.URL.Query().Get("sha256") != digest {
				t.Error("skill digest missing from request")
			}
			_, _ = w.Write(pkg.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	live := &acpLive{client: &client{server: server.URL, http: server.Client()}, auth: authorization{Access: "test-account"}, dir: t.TempDir(), cwd: "/tmp", sessions: map[string]acpSession{}, operations: map[string]string{}}
	host := &relaySessionHost{live: live}
	selected, err := host.resolveDesktopExpert(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	instructions, err := live.client.expertInstructions(t.Context(), live.auth, selected)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"Follow the expert prompt", "Expert rule fixture", "Expert skill fixture"} {
		if !strings.Contains(instructions, value) {
			t.Fatalf("missing %s", value)
		}
	}
	if len(selected.Connectors) != 1 || !selected.Connectors[0].allows("read") || selected.Connectors[0].allows("write") {
		t.Fatal("expert tool restrictions were lost")
	}
	session, err := live.createInWorkspace(connector, "/tmp", "", id, selected.expertSelection, nil, "", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}
	row, err := live.load(session)
	if err != nil || row.workspaceMetadata()["expertId"] != id {
		t.Fatal("session did not preserve expert", err)
	}
	reset, err := json.Marshal(map[string]any{"sessionId": session, "expertId": "", "_meta": map[string]int{"expectedRevision": 0}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := host.setDesktopSessionExpert(t.Context(), reset); err != nil {
		t.Fatal(err)
	}
	row, err = live.load(session)
	if err != nil || row.ExpertID != "" || row.ResourceRevision != 1 {
		t.Fatal("session expert was not cleared", err)
	}
}
