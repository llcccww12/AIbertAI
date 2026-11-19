package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRemoteConnectionChecksServerIdentityAndRenewsWithoutChangingScopes(t *testing.T) {
	c, err := newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	const server = "c98798cc-66bb-423d-8f5d-7cdf86737db3"
	const machine = "28d8aa68-4663-4f4c-b355-d0056a4d7cd8"
	const connection = "d38ff8ce-b949-422d-a080-488b4c89c20a"
	const wrongServer = "6bfc669b-55d4-40eb-afb3-6a9df8247430"
	prefix := "/api/v1/endpoints/" + machine
	now := time.Date(2026, 11, 18, 13, 0, 0, 0, time.UTC)
	issued := 0
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body string
		if r.URL.Host != "private.example.test" {
			t.Fatalf("unexpected private backend: %v", r.URL)
		}
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			if r.Header.Get("Authorization") != "" {
				t.Fatal("metadata request must not carry token")
			}
			body = fmt.Sprintf(`{"issuer":%q,"monkeyai_runtime":{"server_id":%q,"features":{"runtime_relay":true}}}`, c.server, server)
		case prefix + "/connections":
			issued++
			if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer active" {
				t.Fatal("ticket request was not authorized")
			}
			var fields map[string]json.RawMessage
			if json.NewDecoder(r.Body).Decode(&fields) != nil || string(fields["connection_id"]) != fmt.Sprintf("%q", connection) || fields["scopes"] != nil {
				t.Fatalf("renewal changed the connection scope: %v", fields)
			}
			body = fmt.Sprintf(`{"connection_id":%q,"machine_id":%q,"http_base":%q,"ticket":"local-token","ticket_version":5,"expires_at":%q,"server_time":%q,"scopes":["files:read"]}`,
				connection, machine, c.server+prefix+"/proxy/", now.Add(14*time.Minute).Format(time.RFC3339), now.Format(time.RFC3339))
		default:
			t.Fatal(r.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	auth := authorization{Access: "active"}
	if _, err := c.remoteConnection(t.Context(), auth, wrongServer, machine, connection); err == nil || issued != 0 {
		t.Fatalf("wrong server accepted: err=%v, issued=%d", err, issued)
	}
	ticket, err := c.remoteConnection(t.Context(), auth, server, machine, connection)
	if err != nil || issued != 1 || ticket.ConnectionID != connection || ticket.Version != 5 ||
		ticket.RemainingMS != (14*time.Minute).Milliseconds() || len(ticket.Scopes) != 1 || ticket.Scopes[0] != "files:read" {
		t.Fatalf("invalid translated ticket: issued=%d, err=%v", issued, err)
	}
}
