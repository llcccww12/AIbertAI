package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopDSHSessionTrash(t *testing.T) {
	const serverID = "11111111-1111-4111-8111-111111111111"
	const machineID = "22222222-2222-4222-8222-222222222222"
	for _, sdk := range []bool{false, true} {
		name := "rollback"
		if sdk {
			name = "real-sdk"
		}
		t.Run(name, func(t *testing.T) {
			node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
			if sdk && (node == "" || entry == "") {
				t.Skip("patched SDK build not configured")
			}
			profile := t.TempDir()
			live, err := openEmployeeSessions(profile, serverID, machineID, "")
			if err != nil {
				t.Fatal(err)
			}
			defer live.store.Close()
			defer live.workspaces.root.Close()
			root := live.workspaces.root
			host := &relaySessionHost{profile: profile, live: live}
			newRow := func() acpSession {
				t.Helper()
				op, err := newUUID()
				if err != nil {
					t.Fatal(err)
				}
				id, err := live.createSession(op, "", "", serverID, expertSelection{}, nil, "", desktopMediaModels{}, true)
				if err != nil {
					t.Fatal(err)
				}
				return live.sessions[id]
			}
			if sdk {
				client, err := newBackendClient("https://fixture.invalid", false)
				if err != nil {
					t.Fatal(err)
				}
				client.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
					reply := httptest.NewRecorder()
					switch r.URL.Path {
					case "/api/v1/models":
						fmt.Fprintf(reply, `{"total_count":1,"models":[{"id":%q,"kind":"text","protocol":"openai_chat_completions"}]}`, serverID)
					case "/api/v1/api-keys":
						fmt.Fprintf(reply, `{"id":%q,"api_key":"fixture"}`, machineID)
					case "/api/v1/api-keys/" + machineID:
					case "/v1/chat/completions":
						reply.Header().Set("Content-Type", "text/event-stream")
						fmt.Fprintf(reply, "data: %s\n\ndata: %s\n\ndata: [DONE]\n\n",
							`{"id":"fixture","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"saved"},"finish_reason":null}]}`,
							`{"id":"fixture","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`)
					default:
						http.NotFound(reply, r)
					}
					return reply.Result(), nil
				})
				gw := &dshGateway{client: client, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
				base := filepath.Join(root.Name(), "dsh")
				cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(base, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(base, "gateway"), gw, func(context.Context, acpSession) (selectedModelConfig, error) {
					return selectedModelConfig{ID: serverID, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096}, nil
				})
				if err != nil {
					t.Fatal(err)
				}
				live.dsh = cfg
			}
			target, keep := newRow(), newRow()
			query := func(row acpSession, resume bool) error {
				op, err := newUUID()
				if err != nil {
					return err
				}
				prompt := "remember this session"
				if resume {
					prompt = "/goal"
				}
				return runDSHInteractiveTurn(t.Context(), live.dsh, row, &acpTurn{Operation: op, Prompt: prompt}, resume, nil)
			}
			write := func(path string) {
				t.Helper()
				if err := root.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := root.WriteFile(path, []byte("fixture"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			for _, row := range []acpSession{target, keep} {
				if sdk {
					if err := query(row, false); err != nil {
						t.Fatal(err)
					}
				} else {
					write(filepath.Join("dsh", "home", "sessions", "--fixture-project--", row.ID, "session.v4.jsonl.zstd"))
					write(filepath.Join("dsh", "home", "storages", "session_projcache", "sessions", row.ID+".json"))
				}
				write(filepath.Join("dsh", "gateway", row.ID, "receipt.json"))
			}
			if !sdk {
				conflict := filepath.Join("deleted-dsh", "sessions", "--fixture-project--", target.ID)
				write(conflict)
				if _, err := host.trashDesktopSession(target.ID, false); err == nil {
					t.Fatal("trash ignored occupied destination")
				}
				if _, err := live.load(target.ID); err != nil {
					t.Fatal(err)
				}
				for _, path := range []string{filepath.Join("dsh", "home", "storages", "session_projcache", "sessions", target.ID+".json"), filepath.Join("dsh", "gateway", target.ID, "receipt.json")} {
					if _, err := root.Stat(path); err != nil {
						t.Fatalf("rollback lost %s: %v", path, err)
					}
				}
				if err := root.Remove(conflict); err != nil {
					t.Fatal(err)
				}
			}
			_, cacheErr := root.Stat(filepath.Join("dsh", "home", "storages", "session_projcache", "sessions", target.ID+".json"))
			if cacheErr != nil && !errors.Is(cacheErr, os.ErrNotExist) {
				t.Fatal(cacheErr)
			}
			if found, err := host.trashDesktopSession(target.ID, false); err != nil || !found {
				t.Fatalf("trash returned %v %v", found, err)
			}
			if _, err := root.Stat(filepath.Join("deleted-dsh", "gateway", target.ID, "receipt.json")); err != nil {
				t.Fatal(err)
			}
			if _, err := root.Stat(filepath.Join("deleted-dsh", "projection-cache", target.ID+".json")); cacheErr == nil && err != nil {
				t.Fatal(err)
			}
			if _, err := root.Stat(filepath.Join("dsh", "gateway", keep.ID, "receipt.json")); err != nil {
				t.Fatal(err)
			}
			archives, err := filepath.Glob(filepath.Join(root.Name(), "deleted-dsh", "sessions", "*", target.ID, "*.jsonl*"))
			if err != nil || len(archives) == 0 {
				t.Fatalf("no archived SDK generation: %v", err)
			}
			if sdk {
				if err := query(keep, true); err != nil {
					t.Fatalf("other SDK session cannot resume: %v", err)
				}
				if err := query(target, true); err == nil || !strings.Contains(err.Error(), "dsh rpc session/resume:") {
					t.Fatalf("deleted SDK session resume: %v", err)
				}
			}
		})
	}
}
