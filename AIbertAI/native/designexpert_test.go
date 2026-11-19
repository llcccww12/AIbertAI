package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDesignExpertInstructionsForModelAPIs(t *testing.T) {
	const canvas = "e04d45f0-0496-46f1-b3f1-681c8bedf4e6"
	const userPrompt = "Build a prototype."
	instructions := designExpertInstructions(json.RawMessage(`{"canvasId":"` + canvas + `","designType":"prototype","extra":"untrusted instructions"}`))
	if !strings.Contains(instructions, "AIbertAI's Design Expert") || !strings.Contains(instructions, "keyboard focus") ||
		!strings.Contains(instructions, canvas+"/design-context.json") || strings.Contains(instructions, "untrusted instructions") {
		t.Fatalf("unexpected design brief: %s", instructions)
	}
	if designExpertInstructions(json.RawMessage(`{"canvasId":"../../invalid"}`)) != "" {
		t.Fatal("invalid canvas ID activated design instructions")
	}
	for _, route := range []struct{ name, path, field, reply string }{
		{"openai_chat_completions", "/v1/chat/completions", "messages", `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}]}`},
		{"openai_responses", "/v1/responses", "input", `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]}]}`},
	} {
		for _, design := range []bool{true, false} {
			c, err := newClient("https://private.example.test")
			if err != nil {
				t.Fatal(err)
			}
			var policy *workspaceToolPolicy
			if design {
				policy = &workspaceToolPolicy{Instructions: instructions, DesignTools: []desktopDesignTool{{Name: designSelectionTool, Schema: json.RawMessage(`{"type":"object"}`)}}}
			}
			c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				var fields map[string]json.RawMessage
				if err := json.NewDecoder(req.Body).Decode(&fields); err != nil {
					t.Fatal(err)
				}
				var messages []toolConversationMessage
				if err := json.Unmarshal(fields[route.field], &messages); err != nil {
					t.Fatal(err)
				}
				if req.URL.Path != route.path || len(messages) == 0 || messages[len(messages)-1].Role != "user" ||
					messages[len(messages)-1].Content == nil || *messages[len(messages)-1].Content != userPrompt {
					t.Fatal(req.URL.Path, messages)
				}
				if design {
					if len(messages) != 2 || messages[0].Role != "system" || messages[0].Content == nil ||
						*messages[0].Content != instructions || !strings.Contains(string(fields["tools"]), designSelectionTool) {
						t.Fatal("model did not receive expert instructions and its selection tool")
					}
				} else if len(messages) != 1 || strings.Contains(string(fields["tools"]), designSelectionTool) {
					t.Fatal("ordinary conversation received design-only behavior")
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(route.reply))}, nil
			})
			answer, _, _, err := c.completeWorkspaceToolTurn(t.Context(), route.name, "fixture", "session", "model-fixture", nil, userPrompt,
				nil, false, false, 2, nil, policy, defaultModelInputBudget, nil, nil)
			if err != nil || answer != "ok" {
				t.Fatalf("%s (design=%v): %q %v", route.name, design, answer, err)
			}
		}
	}
}

func TestDesignHostDependenciesUseDesignDispatch(t *testing.T) {
	for _, name := range []string{designProductionTool, designPreparePPTXTool, "canvas_capture_preview", "canvas_update_node"} {
		tool := modelToolUse{ID: "call", Type: "function"}
		tool.Function.Name = name
		tool.Function.Arguments = "{}"
		inventory := map[string]bool{name: true}
		native, err := parseModelToolArguments(tool, nil, true, nil, inventory)
		if err != nil || !native.Design {
			t.Fatalf("%s did not reach the canvas host: %v", name, err)
		}
		if _, err := parseModelToolArguments(tool, nil, false, nil, inventory); err == nil {
			t.Fatalf("%s accepted without workspace write consent", name)
		}
		if _, err := parseModelToolArguments(tool, nil, true, nil, nil); err == nil {
			t.Fatalf("%s accepted outside the current design inventory", name)
		}
	}
}
