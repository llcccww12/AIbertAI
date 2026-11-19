package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDesktopModelSharingUsesAccountAndDirectRecipients(t *testing.T) {
	const model = "c2bd985c-9f71-4b85-937a-c41b1b104d38"
	const user = "c4bd985c-9f71-4b85-937a-c41b1b104d38"
	const group = "c5bd985c-9f71-4b85-937a-c41b1b104d38"
	shared := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer account" {
			t.Error("share must use the active account")
		}
		switch r.URL.Path {
		case "/api/v1/users":
			if r.URL.Query().Get("q") != "中文 & A" {
				t.Error("search did not forward query")
			}
			_, _ = w.Write([]byte(`{"users":[{"id":"` + user + `","name":"Test user"}]}`))
		case "/api/v1/groups":
			_, _ = w.Write([]byte(`{"groups":[{"id":"` + group + `","name":"Test group"}]}`))
		case "/api/v1/models":
			users, groups := "[]", "[]"
			if shared {
				users, groups = `[{"id":"`+user+`"}]`, `[{"id":"`+group+`"}]`
			}
			_, _ = w.Write([]byte(`{"models":[{"id":"` + model + `","shared_users":` + users + `,"shared_groups":` + groups + `}],"total_count":1}`))
		case "/api/v1/resources/shares":
			var body struct {
				Resources []struct{ Type, ID string }
				Users     []string `json:"user_ids"`
				Groups    []string `json:"group_ids"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Resources) != 1 ||
				body.Resources[0].Type != "model" || body.Resources[0].ID != model ||
				len(body.Users) != 1 || body.Users[0] != user || len(body.Groups) != 1 || body.Groups[0] != group {
				t.Errorf("unexpected sharing payload: %+v (%v)", body, err)
			}
			shared = r.Method == http.MethodPost
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	host := &relaySessionHost{live: &acpLive{client: &client{server.URL, server.Client()}, auth: authorization{Access: "account"}}}
	call := func(method, input string) any {
		t.Helper()
		result, err := host.localQuery(t.Context(), method, json.RawMessage(input))
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	if len(call("_monkeyai/users/search", `{"q":"中文 & A"}`).([]json.RawMessage)) != 1 ||
		len(call("_monkeyai/groups/search", `{"q":""}`).([]json.RawMessage)) != 1 {
		t.Fatal("missing directory results")
	}
	get := func() desktopSharingRecipients {
		return call("_monkeyai/resources/recipients", `{"kind":"model","id":"`+model+`"}`).(desktopSharingRecipients)
	}
	for _, remove := range []string{"false", "true"} {
		call("_monkeyai/resources/share", `{"kind":"model","id":"`+model+`","user_ids":["`+user+`"],"group_ids":["`+group+`"],"remove":`+remove+`}`)
		got := get()
		want := 1
		if remove == "true" {
			want = 0
		}
		if len(got.Users) != want || len(got.Groups) != want {
			t.Fatalf("wrong direct recipients after remove=%s: %+v", remove, got)
		}
	}
}
