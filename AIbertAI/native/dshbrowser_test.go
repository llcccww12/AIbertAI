package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDSHBuiltinBrowserToolsUseLocalSessionBridge(t *testing.T) {
	const sessionID = "11111111-1111-4111-8111-111111111111"
	const response = `{"url":"http://127.0.0.1:5173/","title":"Deployment fixture","text":"ready","httpStatus":200}`
	var sent int
	bridge := &desktopDesignBridge{pending: make(map[string]chan designToolResult)}
	bridge.send = func(value any) error {
		event := value.(map[string]any)
		sent++
		if event["event"] != "browser_tool" || event["sessionId"] != sessionID || event["name"] != desktopBrowserOpen || event["cwd"] != nil {
			t.Fatalf("incorrect browser event: %#v", event)
		}
		bridge.read(strings.NewReader(fmt.Sprintf("{\"type\":\"browser_tool_result\",\"id\":%q,\"result\":%s}\n", event["id"], response)))
		return nil
	}
	req := httptest.NewRequest("GET", "http://127.0.0.1:3000/acp", nil)
	req.Header.Set("Sec-WebSocket-Protocol", "monkeyai.v1, monkeyai.token.fixture")
	host := &relaySessionHost{local: &desktopRuntimeAccess{address: req.Host, protocol: "monkeyai.token.fixture"}, browserBridge: bridge, live: &acpLive{}}
	session := acpSession{ID: sessionID, CWD: "/workspace"}
	bridges, hint, err := host.dshTurnHostTools(t.Context(), nil, req, session, &acpTurn{})
	if err != nil || len(bridges) != 1 || len(bridges[0].Tools) != 2 || !strings.Contains(hint, desktopBrowserRead) {
		t.Fatalf("browser tool discovery failed: %v %s %v", bridges, hint, err)
	}
	run := func(args string) (any, error) {
		raw := json.RawMessage(fmt.Sprintf(`{"name":%q,"arguments":%s}`, desktopBrowserOpen, args))
		return bridges[0].Handle(raw)
	}
	for _, args := range []string{`{}`, `{"url":"file:///tmp/test"}`, `{"url":"https://user:pass@example.test"}`, `{"url":"https://example.test","tab":"another"}`, `{"url":"https://example.test","url":"https://duplicate.test"}`} {
		if _, err := run(args); err == nil {
			t.Fatalf("accepted invalid arguments %s", args)
		}
	}
	if result, err := run(`{"url":"http://127.0.0.1:5173/"}`); err != nil || result.(map[string]string)["output"] != response || sent != 1 {
		t.Fatalf("unexpected result %s, %v, %d requests", result, err, sent)
	}
	req.Header.Del("Sec-WebSocket-Protocol")
	if host.dshBuiltinBrowserTools(t.Context(), nil, req, session, &acpTurn{}) != nil {
		t.Fatal("unauthorized client could discover the local tools")
	}
	host.local = nil
	if host.dshBuiltinBrowserTools(t.Context(), nil, req, session, &acpTurn{}) != nil {
		t.Fatal("remote session could discover the local tools")
	}
}

func TestDSHBrowserActionRejectsModelSuppliedApproval(t *testing.T) {
	const snapshot = "11111111-1111-4111-8111-111111111111"
	click := fmt.Sprintf(`{"snapshotId":%q,"ref":"e1"}`, snapshot)
	fill := fmt.Sprintf(`{"snapshotId":%q,"ref":"e2","text":"Fixture contact"}`, snapshot)
	if !validDesktopBrowserArgs(desktopBrowserClick, []byte(click)) || !validDesktopBrowserArgs(desktopBrowserFill, []byte(fill)) {
		t.Fatal("normal click and fill arguments should be accepted")
	}
	for _, injected := range []string{`"approvalId":"fake"`, `"operationId":"fake"`, `"selector":"form"`} {
		if validDesktopBrowserArgs(desktopBrowserClick, []byte(strings.TrimSuffix(click, "}")+","+injected+"}")) ||
			validDesktopBrowserArgs(desktopBrowserFill, []byte(strings.TrimSuffix(fill, "}")+","+injected+"}")) {
			t.Fatalf("browser action accepted caller metadata: %s", injected)
		}
	}
}
