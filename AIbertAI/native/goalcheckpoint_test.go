package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestGoalCheckpointRecoversToolResultWithoutReplayingAction(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		name := "recorded"
		if uncertain {
			name = "uncertain"
		}
		t.Run(name, func(t *testing.T) {
			fakeTool := func(id string) string {
				return `{"usage":{"prompt_tokens":10,"completion_tokens":3},"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","reasoning_content":"reasoning", "tool_calls":[{"id":"` + id + `","type":"function","function":{"name":"write_text_file","arguments":"{\"path\":\"report.txt\",\"content\":\"report\"}"}}]}}]}`
			}
			c, err := newClient("https://fixture.example")
			if err != nil {
				t.Fatal(err)
			}
			var onDisk []byte
			g := &desktopGoalRun{remaining: 1000, active: func() bool { return true }, save: func(checkpoint goalCheckpoint) error {
				onDisk, err = json.Marshal(checkpoint)
				return err
			}}
			var executed, calls int
			reply := func(body string) *http.Response {
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
			}
			c.http.Transport = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return reply(fakeTool("first")), nil
				}
				return nil, errors.New("transport interrupted")
			})
			run := func() (string, error) {
				answer, _, _, err := c.completeWorkspaceToolTurn(t.Context(), "openai_chat_completions", "fixture", "", "model",
					nil, "write a report", nil, false, true, 8, nil, &workspaceToolPolicy{CanWrite: true, Goal: g}, 36000, nil,
					func(_ toolConversationMessage, _ toolArguments, _ int) (string, error) {
						executed++
						if uncertain {
							return "", errors.New("media_generation_already_submitted")
						}
						return "recorded result", nil
					})
				return answer, err
			}
			if _, err := run(); err == nil || executed != 1 {
				t.Fatalf("expected one action before interruption, got %d: %v", executed, err)
			}
			saved := &goalCheckpoint{}
			if err := json.Unmarshal(onDisk, saved); err != nil || (saved.Pending != nil) != uncertain || saved.Usage.Input == nil {
				t.Fatalf("wrong checkpoint: %+v %v", saved, err)
			}
			g.checkpoint, g.outcome = saved, ""
			calls = 0
			c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				data, _ := io.ReadAll(req.Body)
				if calls == 1 {
					needle := "recorded result"
					if uncertain {
						needle = "its outcome is unknown"
					}
					if !strings.Contains(string(data), needle) || !strings.Contains(string(data), "reasoning_content") {
						t.Fatalf("missing persisted tool result or replay context: %s", data)
					}
					return reply(fakeTool("second")), nil
				}
				return reply(`{"usage":{"prompt_tokens":11,"completion_tokens":2},"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"Done\n[GOAL_COMPLETE]"}}]}`), nil
			})
			answer, err := run()
			if err != nil || answer != "Done" || executed != 1 || calls != 2 {
				t.Fatalf("resume: answer=%q executions=%d calls=%d err=%v", answer, executed, calls, err)
			}
		})
	}
}
