package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopImportMCPFormat(t *testing.T) {
	rows, err := decodeLegacyMCP([]byte(`{"mcpServers":{"status":{"url":"https://tools.example.test/mcp","headers":{}}}}`))
	if err != nil || len(rows) != 1 || rows[0].Name != "status" {
		t.Fatalf("旧版 MCP 导出解析失败: %v", err)
	}
	for _, value := range []legacyMCP{
		{Command: "test"},
		{Name: "status", URL: "file:///etc/hosts"},
		{Name: "status", URL: "https://tools.example.test/mcp", Headers: json.RawMessage(`[{"name":"authorization","value":"a"},{"name":"Authorization","value":"b"}]`)},
	} {
		if _, err := legacyConnectorInput(value); err == nil {
			t.Fatalf("导入了不兼容的连接器")
		}
	}
}

// The backend integration test supplies an isolated PostgreSQL-backed server and
// a real HTTP MCP fixture. This test uses the same native entry points as a turn.
func TestDesktopLegacyImportIntegration(t *testing.T) {
	server := os.Getenv("AIBERTAI_IMPORT_TEST_BACKEND")
	if server == "" {
		t.Skip("需要隔离的导入测试后端")
	}
	c, err := newClient(server)
	if err != nil {
		t.Fatal(err)
	}
	profile := t.TempDir()
	const dir = "runtime/ohmyagent/experts/mine/local/legacy-editor/revision/"
	block, err := aes.NewCipher(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	key, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	mcp, _ := json.Marshal([]map[string]any{
		{"transport": "sse", "name": "Legacy status", "url": os.Getenv("AIBERTAI_IMPORT_TEST_MCP"), "headers": map[string]string{"Authorization": "Bearer synthetic-legacy-header"}},
		{"transport": "stdio", "name": "Local check", "command": os.Args[0], "args": []string{"-test.run=^TestLocalMCPChild$"}, "env": map[string]string{"AIBERTAI_LOCAL_MCP_FIXTURE": "1"}},
	})
	nonce := make([]byte, key.NonceSize())
	encrypted, _ := json.Marshal(map[string][]byte{"nonce": nonce, "ciphertext": key.Seal(nil, nonce, mcp, nil)})
	files := map[string]string{
		"runtime/ohmyagent/connectors/user/local.enc.json": string(encrypted),
		"runtime/ohmyagent/experts/.catalog.json":          `{"kind":"experts","resources":[{"id":"legacy-editor","scope":"mine","origin":"local","path":"experts/mine/local/legacy-editor/revision"},{"id":"cached-server","scope":"mine","origin":"server"}]}`,
		"runtime/ohmyagent/connectors/.catalog.json":       `{"kind":"connectors","resources":[{"id":"cached-server","scope":"mine","origin":"server"}]}`,
		dir + "bundle.json":                                `{"schema_version":1,"kind":"expert","id":"legacy-editor","entry":"expert.json","requires":{"agent_mcp_servers":["Legacy status","Local check"],"host_tools":["Read"]}}`,
		dir + "expert.json":                                `{"schema_version":1,"id":"legacy-editor","name":"Legacy editor","prompt_file":"prompt.md","builtin_tools":{"whitelist":["Read"]},"rules":[{"path":"rules/preserve.md"}],"skills":[{"path":"skills/check"}]}`,
		dir + "prompt.md":                                  "Return LEGACY_EXPERT_OK.",
		dir + "rules/preserve.md":                          "Legacy rule marker.",
		dir + "skills/check/SKILL.md":                      "Legacy skill marker.",
		"runtime/sessions/legacy-history/meta.json":        `{"message_count":3}`,
	}
	for name, data := range files {
		file := filepath.Join(profile, name)
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	state, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer state.Close()
	auth := authorization{Access: "desktop-import-test"}
	h := relaySessionHost{profile: profile, live: &acpLive{client: c, auth: auth, store: &desktopSessionStore{Root: state}}}
	for i := 0; i < 3; i++ {
		if i == 2 {
			// Force a retry after a lost local receipt. The backend has already committed.
			if err := state.Remove(desktopImportReceipt); err != nil {
				t.Fatal(err)
			}
		}
		result := h.importLegacyResources(t.Context(), auth, key)
		if result.Experts != 1 || result.Connectors != 2 || len(result.Pending) != 0 {
			t.Fatalf("导入轮次 %d: %+v", i, result)
		}
	}
	for name, original := range files {
		data, err := os.ReadFile(filepath.Join(profile, name))
		if err != nil || string(data) != original {
			t.Fatalf("原始数据有变化: %s: %v", name, err)
		}
	}
	rows, err := c.desktopCatalog(t.Context(), auth, "experts")
	if err != nil || len(rows) != 1 {
		t.Fatalf("专家重复或未入库: %d %v", len(rows), err)
	}
	var selected expertSelection
	if err := json.Unmarshal(rows[0], &selected); err != nil {
		t.Fatal(err)
	}
	expert, err := h.resolveDesktopExpert(t.Context(), selected.ID)
	if err != nil || len(expert.Connectors) != 1 || !expert.Connectors[0].Required {
		t.Fatalf("专家或连接器依赖不可用: %+v, %v", expert, err)
	}
	instructions, err := c.expertInstructions(t.Context(), auth, expert)
	if err != nil || !strings.Contains(instructions, "LEGACY_EXPERT_OK") ||
		!strings.Contains(instructions, "Legacy rule marker") || !strings.Contains(instructions, "Legacy skill marker") {
		t.Fatalf("旧专家指令未完整接入: %q %v", instructions, err)
	}
	imported := h.importedResources.Experts[expert.ID]
	if !imported.Builtin.allows(legacyReadTool) || imported.Builtin.allows(legacyWriteTool) || len(imported.LocalTools) != 1 {
		t.Fatalf("旧专家宿主工具或本机关联缺失: %+v", imported)
	}
	local, err := openLocalMCP(t.Context(), h.importedResources.Local[imported.LocalTools[0]], os.Args[0], profile)
	if err != nil {
		t.Fatal(err)
	}
	defer local.close()
	localTools, err := local.tools(imported.LocalTools[0], h.importedResources.Local[imported.LocalTools[0]], nil)
	if err != nil || len(localTools) != 1 {
		t.Fatalf("本机工具列表不可用: %v", err)
	}
	for _, tool := range localTools {
		out, err := local.invoke(tool, json.RawMessage(`{}`))
		if err != nil || !strings.Contains(out, "LOCAL_MCP_OK") {
			t.Fatalf("本机工具调用失败: %v %q", err, out)
		}
	}
	catalog, err := c.employeeConnectors(t.Context(), auth)
	if err != nil {
		t.Fatal(err)
	}
	catalogMatches := 0
	for _, raw := range catalog {
		var row connectorCatalogRow
		if json.Unmarshal(raw, &row) == nil && row.Name == "Legacy status" && row.ID == expert.Connectors[0].ID {
			catalogMatches++
		}
	}
	if catalogMatches != 1 {
		t.Fatalf("导入的连接器出现在目录 %d 次", catalogMatches)
	}
	prefs := h.live.connectorPrefs.normalized()
	if !validUUID(prefs.Bindings[expert.Connectors[0].ID]) {
		t.Fatal("导入的 Header 凭证未绑定")
	}
	prefs.Enabled["mcp/server/"+expert.Connectors[0].ID] = true
	tools, err := c.activeConnectorTools(t.Context(), auth, prefs)
	if err != nil || len(tools) != 1 {
		t.Fatalf("导入的 MCP 工具不可用: %+v %v", tools, err)
	}
	baseTransport := c.http.Transport
	var alias string
	for name := range tools {
		alias = name
	}
	var rounds int
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" {
			return baseTransport.RoundTrip(r)
		}
		rounds++
		data, err := io.ReadAll(r.Body)
		if err != nil || !strings.Contains(string(data), "Legacy rule marker") || !strings.Contains(string(data), "Legacy skill marker") {
			return nil, fmt.Errorf("模型回合未携带旧专家指令")
		}
		body := `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"LEGACY_EXPERT_OK LEGACY_MCP_OK"}}]}`
		if rounds == 1 {
			body = fmt.Sprintf(`{"choices":[{"finish_reason":"tool_calls","message":{"role":"assistant","tool_calls":[{"id":"legacy-call","type":"function","function":{"name":%q,"arguments":"{}"}}]}}]}`, alias)
		} else if !strings.Contains(string(data), "LEGACY_MCP_OK") {
			return nil, fmt.Errorf("模型没有收到真实工具结果")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	answer, _, runs, err := c.completeWorkspaceToolTurn(t.Context(), "openai_chat_completions", "test-key", "", "fixture-model",
		nil, "核验状态。", nil, true, false, 1, tools, &workspaceToolPolicy{Instructions: instructions}, defaultModelInputBudget, nil,
		func(_ toolConversationMessage, args toolArguments, step int) (string, error) {
			if args.Connector == nil {
				return "", fmt.Errorf("模型选择了意外的工具")
			}
			return c.callConnector(t.Context(), auth, "", "legacy-import-test", step, *args.Connector, args.JSON)
		})
	if err != nil || rounds != 2 || len(runs) != 1 || answer != "LEGACY_EXPERT_OK LEGACY_MCP_OK" {
		t.Fatalf("新运行时专家回合异常: %q, %d 轮, %d 次工具调用: %v", answer, rounds, len(runs), err)
	}
}
