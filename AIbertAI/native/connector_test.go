package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestConnectorToolsUsesSelectedCredential(t *testing.T) {
	c, err := newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	const connector = "c98798cc-66bb-423d-8f5d-7cdf86737db3"
	const credential = "28d8aa68-4663-4f4c-b355-d0056a4d7cd8"
	calls := 0
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		want := fmt.Sprintf("/api/v1/connectors/%s/credentials/%s/tools", connector, credential)
		if r.Method != http.MethodGet || r.URL.Path != want || r.URL.Host != "private.example.test" ||
			r.Header.Get("Authorization") != "Bearer account-access" {
			t.Fatalf("invalid request: %s %s", r.Method, r.URL)
		}
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"permission_denied"}}`)),
		}, nil
	})
	auth := authorization{Access: "account-access"}
	if _, err := c.connectorTools(t.Context(), auth, connector, credential); err == nil || err.Error() != "permission_denied" {
		t.Fatalf("expected permission_denied, got %v", err)
	}
	if _, err := c.connectorTools(t.Context(), auth, connector, "../tools"); err == nil || calls != 1 {
		t.Fatalf("unsafe credential ID: err=%v, calls=%d", err, calls)
	}
}
