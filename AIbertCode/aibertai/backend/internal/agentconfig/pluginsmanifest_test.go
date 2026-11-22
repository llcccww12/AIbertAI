package agentconfig

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aibertai.local/backend/internal/expert"
	"aibertai.local/backend/internal/identity"
	"aibertai.local/backend/internal/resource"
	"github.com/go-chi/chi/v5"
)

func TestExpertPluginSelectionManifest(t *testing.T) {
	resources, admin, employee := catalogDatabase(t)
	catalogExec(t, resources, `UPDATE users SET role='admin' WHERE id=$1`, admin)
	tokenHash := sha256.Sum256([]byte(admin))
	catalogExec(t, resources, `INSERT INTO browser_sessions(token_hash,user_id,authentication_method,expires_at) VALUES($1,$2,'oauth',now()+interval '1 hour')`, hex.EncodeToString(tokenHash[:]), admin)
	adminAPI := chi.NewRouter()
	adminAPI.Use(identity.NewService(resources.store.Pool, nil, "http://example.test").RequireAdmin)
	expert.NewService(resources.store).RegisterAdmin(adminAPI)
	employeeAPI := catalogHandler(t, resources, employee)
	var id, revision string
	for _, step := range []struct{ name, send, expect, canvas, canvasExpect string }{
		{"create inherited", "null", "", "null", ""},
		{"select tools", `["tool-fs","tool-todo"]`, `["tool-fs","tool-todo"]`, `["canvas_get_document","canvas_create_text"]`, `["canvas_get_document","canvas_create_text"]`},
		{"legacy client retains selection", "", `["tool-fs","tool-todo"]`, "", `["canvas_get_document","canvas_create_text"]`},
		{"disable all", "[]", "[]", "[]", "[]"},
		{"restore inheritance", "null", "", "null", ""},
	} {
		in := resource.Object{"name": "Plugin expert", "description": "Test", "prompt": "Help with a task", "grants": []resource.Object{{"user_id": employee}}}
		if step.send != "" {
			in["dsh_plugins"] = json.RawMessage(step.send)
		}
		if step.canvas != "" {
			in["canvas_tools"] = json.RawMessage(step.canvas)
		}
		data, err := json.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		method, path := http.MethodPost, "/experts"
		if id != "" {
			method, path = http.MethodPut, "/experts/"+id
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(data))
		req.AddCookie(&http.Cookie{Name: "monkeyai_session", Value: admin})
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("If-Match", revision)
		written := httptest.NewRecorder()
		adminAPI.ServeHTTP(written, req)
		if written.Code != http.StatusOK {
			t.Fatalf("%s: admin %d %s", step.name, written.Code, written.Body)
		}
		var saved resource.Object
		if err := json.Unmarshal(written.Body.Bytes(), &saved); err != nil || saved.String("id") == "" {
			t.Fatalf("%s: invalid admin response %s", step.name, written.Body)
		}
		id, revision = saved.String("id"), written.Header().Get("ETag")
		view := catalogRead(t, employeeAPI, employee, "/experts/"+id+"/manifest", "")
		var manifest map[string]json.RawMessage
		if err := json.Unmarshal(view.Body.Bytes(), &manifest); err != nil {
			t.Fatal(err)
		}
		if got := string(manifest["canvas_tools"]); got != step.canvasExpect {
			t.Fatalf("%s: employee manifest canvas_tools = %q, want %q", step.name, got, step.canvasExpect)
		}
		if got := string(manifest["dsh_plugins"]); got != step.expect {
			t.Fatalf("%s: employee manifest dsh_plugins = %q, want %q", step.name, got, step.expect)
		}
	}
}

func TestEmployeeOwnedExpertPluginSelection(t *testing.T) {
	resources, owner, coworker := catalogDatabase(t)
	catalog := catalogHandler(t, resources, owner, coworker)
	writer := chi.NewRouter()
	writer.Use(identity.NewService(resources.store.Pool, nil, "http://example.test").RequireAgent)
	expert.NewService(resources.store).RegisterAgent(writer)
	skillID := resource.ID()
	catalogExec(t, resources, `INSERT INTO skills(id,owner_user_id,ownership_type,name,description,package_file_name,package_s3_key,package_size_bytes,package_sha256)
		VALUES($1,$2,'user','Drawing','Draw a circle','drawing.zip','drawing.zip',1,'digest')`, skillID, owner)
	var id, revision string
	save := func(actor string, in resource.Object, want int) {
		t.Helper()
		in["name"] = "Drawing expert"
		data, err := json.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		method, path := http.MethodPost, "/experts"
		if id != "" {
			method, path = http.MethodPut, "/experts/"+id
		}
		req := httptest.NewRequest(method, path, bytes.NewReader(data))
		req.Header.Set("Authorization", "Bearer "+actor)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("If-Match", revision)
		w := httptest.NewRecorder()
		writer.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, want, w.Body)
		}
		if want == http.StatusCreated || want == http.StatusOK {
			var saved resource.Object
			if err := json.Unmarshal(w.Body.Bytes(), &saved); err != nil || saved.String("id") == "" || saved.String("created_by_user_id") != owner || saved.String("ownership_type") != "user" {
				t.Fatalf("invalid employee-owned expert: %s", w.Body)
			}
			id, revision = saved.String("id"), w.Header().Get("ETag")
		}
	}
	check := func(plugins string) {
		t.Helper()
		w := catalogRead(t, catalog, owner, "/experts/"+id+"/manifest", "")
		var manifest struct {
			Plugins json.RawMessage       `json:"dsh_plugins"`
			Canvas  json.RawMessage       `json:"canvas_tools"`
			Skills  []struct{ ID string } `json:"skills"`
			Prompt  string                `json:"prompt"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &manifest); err != nil {
			t.Fatal(err)
		}
		if string(manifest.Plugins) != plugins || string(manifest.Canvas) != `["media_generate_image","canvas_create_image"]` ||
			manifest.Prompt != "Draw from this brief." || len(manifest.Skills) != 1 || manifest.Skills[0].ID != skillID {
			t.Fatalf("unexpected employee manifest, wanted plugins %q: %s", plugins, w.Body)
		}
	}
	save(owner, resource.Object{"prompt": "Draw from this brief.", "skill_ids": []string{skillID}, "canvas_tools": []string{"media_generate_image", "canvas_create_image"}}, http.StatusCreated)
	check("")
	save(owner, resource.Object{"dsh_plugins": []string{"tool-skill"}}, http.StatusOK)
	check(`["tool-skill"]`)
	save(coworker, resource.Object{"dsh_plugins": []string{}}, http.StatusNotFound)
	save(owner, resource.Object{"dsh_plugins": []string{"unknown"}}, http.StatusBadRequest)
	save(owner, resource.Object{"dsh_plugins": []string{"tool-fs", "tool-fs"}}, http.StatusBadRequest)
	save(owner, resource.Object{"description": "Legacy update"}, http.StatusOK)
	check(`["tool-skill"]`)
	save(owner, resource.Object{"dsh_plugins": []string{}}, http.StatusOK)
	check("[]")
	save(owner, resource.Object{"dsh_plugins": nil}, http.StatusOK)
	check("")
}
