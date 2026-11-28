package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/coder/websocket"
)

const desktopBrowserRead = "browser_read_current_page"
const desktopBrowserOpen = "browser_open_for_verification"
const desktopBrowserInspect = "browser_inspect_controls"
const desktopBrowserClick = "browser_click_control"
const desktopBrowserFill = "browser_fill_field"

var desktopBrowserRef = regexp.MustCompile(`^e[1-9][0-9]{0,3}$`)

func (h *relaySessionHost) dshBuiltinBrowserTools(ctx context.Context, ws *websocket.Conn, request *http.Request, session acpSession, turn *acpTurn) *dshHostToolBridge {
	if h.local == nil || h.browserBridge == nil || session.ExpertID != "" || !validUUID(session.ID) || h.acpScope(request, "session:read") != nil {
		return nil
	}
	tools := []dshHostTool{
		{Name: desktopBrowserRead, Title: "读取右栏网页", Kind: "read", Category: "browser",
			Description: "Read the rendered text from the visible AIbertAI browser tab in this conversation when the user asks about it. Uses the page's existing login state.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)},
		{Name: desktopBrowserOpen, Title: "在浏览器中验证网页", Kind: "read", Category: "browser",
			Description: "Open the HTTP(S) URL the user asked to verify in a new AIbertAI browser tab. Returns the HTTP status, final URL, title and rendered page text. Use for deployment and rendering checks. Only open a URL for viewing, not a URL that performs an action.",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"url":{"type":"string","maxLength":4096}},"required":["url"],"additionalProperties":false}`)},
	}
	interactive := ws != nil && validUUID(turn.Operation) && session.permissionMode() != modeReadOnly
	if interactive {
		tools = append(tools,
			dshHostTool{Name: desktopBrowserInspect, Title: "查看网页控件", Kind: "read", Category: "browser",
				Description: "List interactive controls on the visible AIbertAI browser tab. Use the returned snapshotId and ref to click or fill.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)},
			dshHostTool{Name: desktopBrowserClick, Title: "点击网页控件", Kind: "execute", Category: "browser",
				Description: "Click one observed control in the current built-in browser. Always asks the user to confirm the control before dispatch. Use this for saving or submitting a form only if the user requested it. If declined or uncertain, do not repeat.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"snapshotId":{"type":"string"},"ref":{"type":"string"}},"required":["snapshotId","ref"],"additionalProperties":false}`)},
			dshHostTool{Name: desktopBrowserFill, Title: "填写网页字段", Kind: "edit", Category: "browser",
				Description: "Replace the text of one observed editable field in the current built-in browser after confirmation. This does not press Enter. Use a separate click for Save or Submit. Never fill authentication or payment credentials.",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"snapshotId":{"type":"string"},"ref":{"type":"string"},"text":{"type":"string","maxLength":4000}},"required":["snapshotId","ref","text"],"additionalProperties":false}`)})
	}
	var cleanup sync.Once
	return &dshHostToolBridge{
		Tools: tools,
		Handle: func(raw json.RawMessage) (any, error) {
			var call dshMutation
			if json.Unmarshal(raw, &call) != nil || len(call.Arguments) > 8192 {
				return nil, errors.New("browser_tool_invalid")
			}
			args, err := strictDSHJSON(call.Arguments)
			if err != nil || !validDesktopBrowserArgs(call.Name, args) {
				return nil, errors.New("browser_tool_invalid")
			}
			if browserControlTool(call.Name) {
				if !interactive || ctx.Err() != nil {
					return nil, errors.New("browser_tools_disabled")
				}
				cleanup.Do(func() {
					context.AfterFunc(ctx, func() {
						_ = h.browserBridge.send(map[string]string{"event": "browser_tool_end", "operationId": turn.Operation})
					})
				})
				return h.dshBrowserControl(ctx, ws, session, turn, call, args)
			}
			result, err := h.browserBridge.call(ctx, session, nil, call.Name, args)
			if err != nil {
				result = `{"error":"browser_unavailable"}`
			}
			return map[string]string{"output": result}, nil
		},
	}
}

func browserControlTool(name string) bool {
	return name == desktopBrowserInspect || name == desktopBrowserClick || name == desktopBrowserFill
}

func (h *relaySessionHost) dshBrowserControl(ctx context.Context, ws *websocket.Conn, session acpSession, turn *acpTurn, call dshMutation, raw json.RawMessage) (any, error) {
	plan := turn.DSHPlanActive
	for i := len(session.Turns) - 1; plan == nil && i >= 0; i-- {
		plan = session.Turns[i].DSHPlanActive
	}
	if call.Name != desktopBrowserInspect && plan != nil && *plan {
		return nil, errors.New("tool_permission_denied")
	}
	var args map[string]string
	_ = json.Unmarshal(raw, &args)
	args["operationId"] = turn.Operation
	invoke := func() string {
		wire, _ := json.Marshal(args)
		result, err := h.browserBridge.call(ctx, session, nil, call.Name, wire)
		if err != nil {
			return `{"error":"browser_unavailable","hint":"Inspect the current page before trying another action."}`
		}
		return result
	}
	result := invoke()
	if call.Name != desktopBrowserInspect {
		var target struct{ ApprovalID, Origin, Label string }
		if json.Unmarshal([]byte(result), &target) == nil && validUUID(target.ApprovalID) {
			title := "确认点击网页控件"
			details := map[string]string{"网站": target.Origin, "控件": target.Label}
			if call.Name == desktopBrowserFill {
				title = "确认填写网页字段"
				details["拟填写内容"] = args["text"]
			}
			counts, err := h.requestDSHToolApproval(ctx, ws, session.ID, call.CallID, title+" · "+target.Label, call.Name, "execute", details, false)
			turn.Permissions.Asked += counts.Asked
			turn.Permissions.Allowed += counts.Allowed
			turn.Permissions.Denied += counts.Denied
			if err != nil {
				return nil, err
			}
			if ctx.Err() != nil {
				return nil, errors.New("tool_permission_cancelled")
			}
			current, err := h.live.load(session.ID)
			if err != nil || current.permissionMode() == modeReadOnly {
				return nil, errors.New("tool_permission_denied")
			}
			args["approvalId"] = target.ApprovalID
			result = invoke()
		}
	}
	return map[string]string{"output": result}, nil
}

func validDesktopBrowserArgs(name string, args []byte) bool {
	switch name {
	case desktopBrowserRead, desktopBrowserInspect:
		return string(args) == "{}"
	case desktopBrowserOpen:
		var fields map[string]string
		if json.Unmarshal(args, &fields) != nil || len(fields) != 1 || len(fields["url"]) > 4096 {
			return false
		}
		parsed, err := url.Parse(fields["url"])
		return err == nil && parsed.Hostname() != "" && parsed.User == nil && (parsed.Scheme == "http" || parsed.Scheme == "https")
	case desktopBrowserClick, desktopBrowserFill:
		var fields map[string]string
		if json.Unmarshal(args, &fields) != nil || !validUUID(fields["snapshotId"]) || !desktopBrowserRef.MatchString(fields["ref"]) {
			return false
		}
		if name == desktopBrowserClick {
			return len(fields) == 2
		}
		value, ok := fields["text"]
		return ok && len(fields) == 3 && len(value) <= 8000 && !strings.ContainsRune(value, 0)
	}
	return false
}
