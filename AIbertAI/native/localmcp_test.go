package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestLocalMCPChild(t *testing.T) {
	if os.Getenv("AIBERTAI_LOCAL_MCP_FIXTURE") != "1" {
		return
	}
	reader := json.NewDecoder(os.Stdin)
	for {
		var request struct {
			ID     int
			Method string
		}
		if reader.Decode(&request) != nil {
			os.Exit(0)
		}
		var result any
		switch request.Method {
		case "initialize":
			result = map[string]string{"protocolVersion": "2025-03-26"}
		case "notifications/initialized":
			continue
		case "tools/list":
			result = map[string]any{"tools": []map[string]any{{"name": "check_local", "inputSchema": map[string]string{"type": "object"}}}}
		case "tools/call":
			result = map[string]any{"content": []map[string]string{{"type": "text", "text": "LOCAL_MCP_OK"}}}
		default:
			os.Exit(2)
		}
		_ = json.NewEncoder(os.Stdout).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
	}
}

func TestSavedLocalConnectorUsesImportedCommand(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	const name = "Fixture connector"
	host := &relaySessionHost{profile: t.TempDir(), importedResources: &importedDesktopResources{
		Local: map[string]legacyMCP{localMCPToolID(name): {
			Name: name, Command: executable, Args: []string{"-test.run=^TestLocalMCPChild$"},
			Env: json.RawMessage(`{"AIBERTAI_LOCAL_MCP_FIXTURE":"1"}`),
		}},
	}}
	result, err := host.testSavedLocalConnector(t.Context(), json.RawMessage(`{"name":"Fixture connector","command":"ignored"}`))
	if err != nil {
		t.Fatal(err)
	}
	response := result.(map[string]any)
	if response["ok"] != true || response["tools"].([]map[string]any)[0]["name"] != "check_local" {
		t.Fatalf("fixture did not list its actual tools: %v", response)
	}
}

func TestEditedLocalMCPRetainsMaskedValuesAndInvokesTool(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	previous := legacyMCP{
		Name: "Imported connector", Env: json.RawMessage(`[{"name":"AIBERTAI_LOCAL_MCP_FIXTURE","value":"1"}]`),
		ToolStates: map[string]bool{"check_local": true},
	}
	edited, err := localConnectorInput(legacyMCP{
		Name: previous.Name, Transport: "stdio", Command: executable,
		Args: []string{"-test.run=^TestLocalMCPChild$"}, Env: json.RawMessage(`{"AIBERTAI_LOCAL_MCP_FIXTURE":""}`),
	}, previous)
	if err != nil || !edited.ToolStates["check_local"] || !json.Valid(edited.Env) {
		t.Fatalf("edit did not retain the imported configuration: %v", err)
	}
	client, err := openLocalMCP(t.Context(), edited, executable, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer client.close()
	tool := connectorTool{Name: "check_local"}
	result, err := client.invoke(tool, json.RawMessage(`{}`))
	if err != nil || result != `{"content":[{"text":"LOCAL_MCP_OK","type":"text"}]}` {
		t.Fatalf("edited connector invocation: %s, %v", result, err)
	}
}
