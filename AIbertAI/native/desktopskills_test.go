package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSlashSkillReachesModelForOneTurn(t *testing.T) {
	const id = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	var revoked atomic.Bool
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/skills":
			fmt.Fprintf(w, `{"skills":[{"id":%q,"name":"summarize"}],"total_count":1}`, id)
		case "/api/v1/skills/" + id + "/content":
			if r.Header.Get("Authorization") != "Bearer account" {
				t.Error("skill request did not use the current account")
			}
			if revoked.Load() {
				http.NotFound(w, r)
				return
			}
			fmt.Fprint(w, `{"name":"summarize","content":"Use the configured summary format."}`)
		case "/api/v1/models":
			fmt.Fprintf(w, `{"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}],"total_count":1}`, id)
		case "/v1/chat/completions":
			current := calls.Add(1)
			var body struct {
				Messages []struct{ Role, Content string }
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Messages) == 0 {
				t.Errorf("invalid model request: %v", err)
				http.Error(w, "bad model request", 400)
				return
			}
			if current == 1 {
				if body.Messages[0].Role != "system" || !strings.Contains(body.Messages[0].Content, "Use the configured summary format.") ||
					body.Messages[len(body.Messages)-1].Content != "Summarize this text" {
					t.Errorf("skill was not applied to the user's task: %+v", body.Messages)
				}
			} else if body.Messages[0].Role != "user" {
				t.Error("skill leaked into a subsequent ordinary turn")
			}
			fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"fixture completed"}}]}`)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	host := &relaySessionHost{
		live: &acpLive{auth: authorization{Access: "account"}, key: "model-key",
			client: &client{server: server.URL, http: server.Client()}, workspaces: &desktopWorkspaces{}},
		local: &desktopRuntimeAccess{address: "fixture.invalid", protocol: "fixture"},
	}
	request := httptest.NewRequest(http.MethodGet, "http://fixture.invalid/acp", nil)
	request.Header.Set("Sec-WebSocket-Protocol", "monkeyai.v1, fixture")
	run := func(prompt string) (acpTurn, error) {
		turn := acpTurn{Model: id, Prompt: prompt}
		err := host.runACPModelTurn(t.Context(), nil, request, acpSession{ID: id}, &turn, nil)
		return turn, err
	}
	turn, err := run("/SUMMARIZE Summarize this text")
	if err != nil || turn.Answer != "fixture completed" || turn.Prompt != "/SUMMARIZE Summarize this text" {
		t.Fatalf("skill turn failed: %q, %v", turn.Answer, err)
	}
	if _, err := run("/workspace/input.txt is the source"); err != nil {
		t.Fatalf("ordinary path was treated as a skill: %v", err)
	}
	revoked.Store(true)
	if _, err := run("/summarize Try again"); !errors.Is(err, errSkillCommandUnavailable) || calls.Load() != 2 {
		t.Fatalf("revoked skill reached the model: %v", err)
	}
}

func TestExecuteSkillScriptUsesPinnedPackageAndWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a Bash fixture")
	}
	const id = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	t.Setenv("AIBERTAI_TEST_SCRIPT_SECRET", "private")
	archive := func(extra string) []byte {
		t.Helper()
		var b bytes.Buffer
		z := zip.NewWriter(&b)
		for name, body := range map[string]string{
			"fixture/SKILL.md":         "Test script",
			"fixture/data":             "package-ok",
			"fixture/scripts/check.sh": "set -eu\ntest -z \"${AIBERTAI_TEST_SCRIPT_SECRET+x}\"\ncat work.txt \"$AIBERTAI_SKILL_DIR/data\"\nprintf ':%s' \"$1\"",
			extra:                      "ignored",
		} {
			w, err := z.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte(body)); err != nil {
				t.Fatal(err)
			}
		}
		if err := z.Close(); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "work.txt"), []byte("workspace-ok:"), 0600); err != nil {
		t.Fatal(err)
	}
	var requestCount atomic.Int32
	for _, entry := range []string{"fixture/extra", "fixture/../outside"} {
		payload := archive(entry)
		sha := fmt.Sprintf("%x", sha256.Sum256(payload))
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount.Add(1)
			if r.URL.Path != "/api/v1/skills/"+id+"/package" || r.URL.Query().Get("sha256") != sha || r.Header.Get("Authorization") != "Bearer account" {
				t.Error("package download lacked the expected identity or revision")
			}
			_, _ = w.Write(payload)
		}))
		c := &client{server: server.URL, http: server.Client()}
		call := skillScriptCall{Path: "scripts/check.sh", Args: []string{"one; echo invalid"}}
		out, err := c.executeSkillScript(t.Context(), authorization{Access: "account"}, desktopSkillPackage{ID: id, SHA: sha}, call, cwd)
		if entry == "fixture/extra" {
			if err != nil || !strings.HasSuffix(out, "workspace-ok:package-ok:one; echo invalid") {
				t.Errorf("script output %q, error %v", out, err)
			}
		} else if !errors.Is(err, errSkillPackageInvalid) {
			t.Errorf("unsafe package returned %q, %v", out, err)
		}
		call.Path = "../check.sh"
		if _, err := c.executeSkillScript(t.Context(), authorization{Access: "account"}, desktopSkillPackage{ID: id, SHA: sha}, call, cwd); !errors.Is(err, errSkillPackageInvalid) {
			t.Errorf("unsafe script path: %v", err)
		}
		server.Close()
	}
	if requestCount.Load() != 2 {
		t.Fatalf("unexpected package requests: %d", requestCount.Load())
	}
}
