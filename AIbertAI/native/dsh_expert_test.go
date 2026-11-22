package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDSHExpertInstructions(t *testing.T) {
	const expertID = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const skillID = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	var pkg bytes.Buffer
	archive := zip.NewWriter(&pkg)
	file, err := archive.Create("fixture/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("verified skill")); err != nil {
		t.Fatal(err)
	}
	asset, err := archive.Create("fixture/templates/example.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := asset.Write([]byte("verified asset")); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(pkg.Bytes()))
	for _, tc := range []struct {
		name, id, manifest, digest string
		fail                       string
	}{
		{"valid", expertID, `"available":true`, digest, ""},
		{"unavailable", expertID, `"available":false`, digest, "expert_unavailable"},
		{"mismatched id", expertID, `"available":true,"expert_id":"c3bd985c-9f71-4b85-937a-c41b1b104d38"`, digest, "expert_unavailable"},
		{"tampered package", expertID, `"available":true`, strings.Repeat("0", 64), "invalid_expert_skill"},
		{"expert with connector", expertID, `"available":true,"connectors":[{"id":"c3bd985c-9f71-4b85-937a-c41b1b104d38","required":true}]`, digest, ""},
		{"invalid id", "bad-id", "", "", "expert_unavailable"},
		{"design unsupported", designExpertID, "", "", "dsh_canvas_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c, err := newBackendClient("https://gateway.test", false)
			if err != nil {
				t.Fatal(err)
			}
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Header.Get("Authorization") != "Bearer test-account" {
					t.Errorf("request lost account authorization")
				}
				var body string
				switch r.URL.Path {
				case "/api/v1/experts/" + expertID + "/manifest":
					idField := `"expert_id":"` + expertID + `",`
					if strings.Contains(tc.manifest, `"expert_id"`) {
						idField = ""
					}
					body = `{` + idField + tc.manifest + `,"prompt":"expert prompt","rules":[{"content":"expert rule"}],"skills":[{"id":"` + skillID + `","package_sha256":"` + tc.digest + `"}]}`
				case "/api/v1/experts/" + expertID + "/skills/" + skillID + "/package":
					if r.URL.Query().Get("sha256") != tc.digest {
						t.Error("skill digest not bound to request")
					}
					body = pkg.String()
				default:
					t.Errorf("unexpected path %q", r.URL.Path)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})
			h := &relaySessionHost{live: &acpLive{client: c, auth: authorization{Access: "test-account"}, workspaces: &desktopWorkspaces{}}}
			bound, release, err := h.bindDSHExpert(t.Context(), acpSession{ExpertID: tc.id})
			defer release()
			got, _ := bound.Value(dshExpertInstructionsKey{}).(string)
			if tc.fail != "" {
				if err == nil || err.Error() != tc.fail || got != "" {
					t.Fatalf("got instructions %q, error %v", got, err)
				}
				if (tc.name == "invalid id" || tc.name == "design unsupported") && calls != 0 {
					t.Fatal("invalid expert reached backend")
				}
				return
			}
			if err != nil || got != "expert prompt\n\nexpert rule" || calls != 2 {
				t.Fatalf("instructions=%q calls=%d error=%v", got, calls, err)
			}
			skills, _ := bound.Value(dshExpertSkillsKey{}).([]dshExpertSkill)
			if len(skills) != 1 || skills[0].Name != "aibert-"+skillID || skills[0].Content != "verified skill" {
				t.Fatalf("skills=%v", skills)
			}
			dir := skills[0].Directory
			data, err := os.ReadFile(filepath.Join(dir, "templates", "example.txt"))
			if err != nil || string(data) != "verified asset" {
				t.Fatal("asset unavailable", err)
			}
			release()
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatal("turn left extracted skill files")
			}
		})
	}
	t.Run("authorization denied", func(t *testing.T) {
		c, err := newBackendClient("https://gateway.test", false)
		if err != nil {
			t.Fatal(err)
		}
		c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusForbidden, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}, nil
		})
		h := &relaySessionHost{live: &acpLive{client: c, auth: authorization{Access: "denied"}, workspaces: &desktopWorkspaces{}}}
		if got, _, err := h.bindDSHExpert(t.Context(), acpSession{ExpertID: expertID}); err == nil || got.Value(dshExpertSkillsKey{}) != nil {
			t.Fatalf("unauthorized expert accepted: %v %v", got, err)
		}
	})
	h := &relaySessionHost{}
	if got, _, err := h.bindDSHExpert(t.Context(), acpSession{}); got.Value(dshExpertInstructionsKey{}) != nil || err != nil {
		t.Fatalf("no expert: %v %v", got, err)
	}
	if _, _, err := h.bindDSHExpert(t.Context(), acpSession{ExpertID: expertID}); err == nil {
		t.Fatal("missing host accepted expert")
	}
}
