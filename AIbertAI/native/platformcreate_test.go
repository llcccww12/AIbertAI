package main

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestPersonalSkillPackage(t *testing.T) {
	pkg, err := personalSkillPackage("weekly_report", " 写周报 ", "# Steps\nDo it.")
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(pkg), int64(len(pkg)))
	if err != nil || len(z.File) != 1 || z.File[0].Name != "weekly_report/SKILL.md" {
		t.Fatalf("package layout: %v", err)
	}
	f, _ := z.File[0].Open()
	body, _ := io.ReadAll(f)
	// The backend parser requires "---\n", YAML, then "\n---\n" before the body.
	if string(body) != "---\nname: \"weekly_report\"\ndescription: \"写周报\"\n---\n# Steps\nDo it." {
		t.Fatalf("SKILL.md: %q", body)
	}
	for _, bad := range [][3]string{{"周报", "d", "c"}, {"-x", "d", "c"}, {"ok", " ", "c"}, {"ok", "d", " "}, {"ok", "d", strings.Repeat("x", 128<<10+1)}} {
		if _, err := personalSkillPackage(bad[0], bad[1], bad[2]); err == nil {
			t.Errorf("accepted %q", bad[0])
		}
	}
}

func TestRemoteConnectorInput(t *testing.T) {
	got, err := remoteConnectorInput(" Linear ", "issues", "https://mcp.linear.app/mcp")
	if err != nil || got["name"] != "Linear" || got["url"] != "https://mcp.linear.app/mcp" || got["authorization_mode"] != "none" {
		t.Fatalf("%v %v", got, err)
	}
	for _, url := range []string{"11", "ftp://x/mcp", "https://user:pw@x/mcp", "https://x/mcp#a", "https:///mcp"} {
		if _, err := remoteConnectorInput("n", "", url); err == nil {
			t.Errorf("accepted %q", url)
		}
	}
}

func TestPlatformCreateToolSchemasAreStrict(t *testing.T) {
	for _, tool := range dshPlatformCreateTools() {
		if _, err := dshStrictToolSchema(tool.InputSchema); err != nil {
			t.Errorf("%s: %v", tool.Name, err)
		}
	}
}
