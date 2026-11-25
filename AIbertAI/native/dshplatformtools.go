package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync"

	"github.com/coder/websocket"
)

const (
	dshCreateSkillTool     = "create_platform_skill"
	dshCreateConnectorTool = "create_platform_connector"
	dshCreateLocalMCPTool  = "create_local_mcp"
)

const dshPlatformCreateInstructions = "When the user asks you to create a reusable skill, connect a remote MCP service, or add a local MCP server, use create_platform_skill, create_platform_connector or create_local_mcp. Each call shows the user an approval card and saves the result to their account on the platform, where it is listed in Settings. Never put API keys, tokens or passwords in these calls: tell the user to add credentials in Settings → Connectors. A new skill is available to the skill tool right after it is saved; a new connector must be enabled in the conversation's connector menu and is available from the next message."

func dshPlatformCreateTools() []dshHostTool {
	schema := func(s string) json.RawMessage { return json.RawMessage(s) }
	return []dshHostTool{
		{Name: dshCreateSkillTool, Title: "创建技能", Kind: "other", Category: "builtin",
			Description: "Save a reusable skill (Markdown instructions) to the user's platform account. name: letters, digits, '-' or '_', at most 64 characters, starting with a letter or digit. description: when to use the skill, one or two sentences. content: the full Markdown instructions.",
			InputSchema: schema(`{"type":"object","properties":{"name":{"type":"string","pattern":"^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$"},"description":{"type":"string","minLength":1,"maxLength":1024},"content":{"type":"string","minLength":1,"maxLength":131072}},"required":["name","description","content"],"additionalProperties":false}`)},
		{Name: dshCreateConnectorTool, Title: "创建连接器", Kind: "other", Category: "builtin",
			Description: "Save a remote MCP service (streamable HTTP) as a personal connector on the user's platform account. url must be the full http:// or https:// MCP endpoint. Credentials are added by the user in Settings, never here.",
			InputSchema: schema(`{"type":"object","properties":{"name":{"type":"string","minLength":1,"maxLength":200},"description":{"type":"string","maxLength":2000},"url":{"type":"string","minLength":1,"maxLength":2048}},"required":["name","url"],"additionalProperties":false}`)},
		{Name: dshCreateLocalMCPTool, Title: "添加本机 MCP", Kind: "other", Category: "builtin",
			Description: "Add a local MCP server that runs on this device over stdio, for example command 'npx' with args ['-y','@modelcontextprotocol/server-filesystem','/path']. name: no spaces, '/', '\\' or '__'. Environment variables and secrets are added by the user in Settings, never here.",
			InputSchema: schema(`{"type":"object","properties":{"name":{"type":"string","minLength":1,"maxLength":128},"command":{"type":"string","minLength":1,"maxLength":4096},"args":{"type":"array","items":{"type":"string","maxLength":8192},"maxItems":128}},"required":["name","command"],"additionalProperties":false}`)},
	}
}

// dshPlatformCreateBridge saves skills and connectors only after the user
// approves the exact arguments of a pending, persisted call in this turn.
func (h *relaySessionHost) dshPlatformCreateBridge(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn) func(json.RawMessage) (any, error) {
	var mu sync.Mutex
	seen := map[string]bool{}
	return func(raw json.RawMessage) (any, error) {
		mu.Lock()
		defer mu.Unlock()
		var req dshMutation
		if json.Unmarshal(raw, &req) != nil || req.SessionID != session.DSHSessionID || req.CallID == "" || len(req.CallID) > 1024 || seen[req.CallID] || len(req.Arguments) > 160<<10 {
			return nil, errors.New("dsh_platform_request_invalid")
		}
		seen[req.CallID] = true
		var input struct {
			Name, Description, Content, URL, Command string
			Args                                     []string
		}
		decoder := json.NewDecoder(bytes.NewReader(req.Arguments))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF {
			return nil, errors.New("dsh_platform_request_invalid")
		}
		var title string
		var approval map[string]any
		switch req.Name {
		case dshCreateSkillTool:
			if _, err := personalSkillPackage(input.Name, input.Description, input.Content); err != nil {
				return nil, err
			}
			title, approval = "创建技能 "+input.Name, map[string]any{"name": input.Name, "description": input.Description, "content": input.Content}
		case dshCreateConnectorTool:
			body, err := remoteConnectorInput(input.Name, input.Description, input.URL)
			if err != nil {
				return nil, err
			}
			title, approval = "创建连接器 "+input.Name, body
		case dshCreateLocalMCPTool:
			if _, err := localConnectorInput(legacyMCP{Name: input.Name, Transport: "stdio", Command: input.Command, Args: input.Args}, legacyMCP{}); err != nil {
				return nil, err
			}
			title, approval = "添加本机 MCP "+input.Name, map[string]any{"name": input.Name, "command": input.Command, "args": input.Args}
		default:
			return nil, errors.New("dsh_platform_request_invalid")
		}
		find := func(row acpSession) (int, *workspaceToolRun) {
			if row.Engine != "dsh" || row.State != "running" || row.DSHSessionID != session.DSHSessionID || row.ExpertID != "" || len(row.Turns) == 0 || row.Turns[len(row.Turns)-1].Operation != turn.Operation {
				return 0, nil
			}
			current := row.Turns[len(row.Turns)-1]
			for i := range current.ToolRuns {
				run := &current.ToolRuns[i]
				if run.DSHCallID != req.CallID || !run.Pending || run.Call == nil || run.Call.Function.Name != req.Name {
					continue
				}
				var expected, got any
				if json.Unmarshal([]byte(run.Call.Function.Arguments), &expected) != nil || json.Unmarshal(req.Arguments, &got) != nil || !reflect.DeepEqual(expected, got) {
					return 0, nil
				}
				return i, run
			}
			return 0, nil
		}
		allowed := func() error {
			if h.local == nil || h.live == nil || h.live.store == nil || h.acpScope(request.WithContext(ctx), "resources:manage") != nil {
				return errRelayAccess
			}
			return nil
		}
		if err := allowed(); err != nil {
			return nil, err
		}
		row, err := h.live.load(session.ID)
		if err != nil {
			return nil, err
		}
		step, run := find(row)
		if run == nil {
			return nil, errors.New("dsh_platform_request_invalid")
		}
		permissions, err := h.requestDSHToolApproval(ctx, ws, session.ID, run.Call.ID, title, req.Name, "other", approval, false)
		turn.Permissions.Asked += permissions.Asked
		turn.Permissions.Allowed += permissions.Allowed
		turn.Permissions.Denied += permissions.Denied
		if err != nil {
			return nil, err
		}
		if err = allowed(); err != nil {
			return nil, err
		}
		approvedStep := step
		effect, err := h.beginDSHSideEffect(ctx, session.ID, turn.Operation, req.CallID, func(current acpSession) bool {
			step, run = find(current)
			return run != nil && step == approvedStep
		})
		if err != nil {
			return nil, err
		}
		var output string
		switch req.Name {
		case dshCreateSkillTool:
			_, err = h.createPlatformSkill(ctx, input.Name, input.Description, input.Content)
			output = "Saved skill " + input.Name + " to the user's account. It is listed in Settings → Skills and can be loaded now with the skill tool."
		case dshCreateConnectorTool:
			var auth authorization
			if auth, err = h.live.authorization(ctx); err == nil {
				_, err = h.live.client.knowledgeChange(ctx, auth, knowledgeOperation{Verb: http.MethodPost, Path: "/api/v1/connectors", Body: approval})
			}
			output = "Saved connector " + input.Name + " to the user's account. It is listed in Settings → Connectors; if the service needs credentials the user adds them there. The user enables it in this conversation's connector menu, and its tools are available from the next message."
		case dshCreateLocalMCPTool:
			params, _ := json.Marshal(map[string]any{"server": legacyMCP{Name: input.Name, Transport: "stdio", Command: input.Command, Args: input.Args}})
			_, err = h.saveLocalConnector(ctx, params)
			output = "Added local MCP server " + input.Name + " on this device. It is listed in Settings → Connectors under 本机 MCP and starts, after the user approves it, from the next message."
		}
		if err != nil {
			return nil, errors.Join(errors.New("dsh_connector_outcome_unknown"), err)
		}
		if err = effect.complete(output); err != nil {
			return nil, err
		}
		return map[string]any{"output": output}, nil
	}
}

// dshPlatformCreateHostTools offers the create tools to Standard conversations
// on this device whose caller may manage the account's resources.
func (h *relaySessionHost) dshPlatformCreateHostTools(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn) (*dshHostToolBridge, string) {
	if session.ExpertID != "" || session.GoalRun != "" || ws == nil || h.local == nil || h.live == nil || h.live.client == nil || request == nil || h.acpScope(request.WithContext(ctx), "resources:manage") != nil {
		return nil, ""
	}
	return &dshHostToolBridge{Tools: dshPlatformCreateTools(), Handle: h.dshPlatformCreateBridge(ctx, ws, request, session, turn)}, dshPlatformCreateInstructions
}

