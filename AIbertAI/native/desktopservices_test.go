package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDesktopEnvironmentForwardsScopedOperations(t *testing.T) {
	const id = "c1bd985c-9f71-4b85-937a-c41b1b104d38"
	const user = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer owner" {
			t.Error("request was not account-scoped")
		}
		seen = append(seen, r.Method+" "+r.URL.RequestURI())
		if r.Method != http.MethodGet {
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload["request_key"] != "stable-request-key" {
				t.Errorf("missing idempotency key: %v", err)
			}
			if strings.Contains(r.URL.Path, id) && payload["generation"] != float64(3) {
				t.Errorf("missing target generation: %+v", payload)
			}
		}
		if strings.HasSuffix(r.URL.Path, "/start") {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":{"code":"generation_conflict"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()
	host := &relaySessionHost{local: &desktopRuntimeAccess{userID: user},
		live: &acpLive{client: &client{server.URL, server.Client()}, auth: authorization{Access: "owner"}}}
	call := func(method, fields string) any {
		t.Helper()
		payload := json.RawMessage(`{"serverUrl":"` + server.URL + `","userId":"` + user + `",` + fields + `}`)
		v, err := host.localQuery(t.Context(), "_monkeyai/environments/"+method, payload)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	call("list", `"page":1,"page_size":20`)
	call("get", `"id":"`+id+`"`)
	call("operations", `"id":"`+id+`","page":2,"page_size":20`)
	call("change", `"action":"create","name":"fixture","request_key":"stable-request-key"`)
	result := call("change", `"action":"start","id":"`+id+`","generation":3,"request_key":"stable-request-key"`)
	if result.(map[string]any)["error"].(map[string]any)["code"] != "generation_conflict" {
		t.Fatalf("conflict was lost: %v", result)
	}
	call("change", `"action":"stop","id":"`+id+`","generation":3,"request_key":"stable-request-key"`)
	call("change", `"action":"delete","id":"`+id+`","generation":3,"request_key":"stable-request-key"`)
	want := []string{"GET /api/v1/environments?page=1&page_size=20", "GET /api/v1/environments/" + id,
		"GET /api/v1/environments/" + id + "/operations?page=2&page_size=20", "POST /api/v1/environments",
		"POST /api/v1/environments/" + id + "/start", "POST /api/v1/environments/" + id + "/stop", "DELETE /api/v1/environments/" + id}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected dispatch: %v", seen)
	}
}

func TestDesktopFeedbackUploadsAndChecksCaller(t *testing.T) {
	const address = "127.0.0.1:12345"
	image := bytes.Repeat([]byte{42}, 96<<10)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/feedback" || r.Method != http.MethodPost || r.Header.Get("Idempotency-Key") != "desktop-feedback-fixture" ||
			r.Header.Get("Authorization") != "Bearer account" || r.FormValue("content") != "fixture" ||
			r.FormValue("rating") != "5" || r.FormValue("platform") != "desktop" || r.FormValue("client_version") != "fixture-version" {
			t.Error("incorrect feedback transport")
		}
		file, _, err := r.FormFile("images")
		if err != nil {
			t.Error(err)
		} else {
			defer file.Close()
			data, _ := io.ReadAll(file)
			if !bytes.Equal(data, image) {
				t.Error("attachment bytes changed")
			}
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"receipt"}`))
	}))
	defer server.Close()
	h := &relaySessionHost{live: &acpLive{client: &client{server.URL, server.Client()}, auth: authorization{Access: "account"}},
		local: &desktopRuntimeAccess{address: address, protocol: "monkeyai.token.localfixture", clientVersion: "fixture-version"}}
	body, _ := json.Marshal(map[string]any{"category": "bug", "content": "fixture", "rating": 5, "idempotencyKey": "desktop-feedback-fixture",
		"images": []string{"data:image/png;base64," + base64.StdEncoding.EncodeToString(image)}})
	for _, test := range []struct {
		origin, token string
		status        int
	}{
		{"", "bad", 404}, {"https://other.example", "localfixture", 404}, {"", "localfixture", 200},
	} {
		req := httptest.NewRequest(http.MethodPost, "http://"+address+desktopFeedbackRoute, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+test.token)
		if test.origin != "" {
			req.Header.Set("Origin", test.origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != test.status || test.status == 200 && !strings.Contains(w.Body.String(), "receipt") {
			t.Fatalf("wrong feedback result %d %s", w.Code, w.Body.String())
		}
	}
	if calls != 1 {
		t.Fatalf("unexpected backend submissions: %d", calls)
	}
}
