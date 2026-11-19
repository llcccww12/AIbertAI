package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRefreshDesktopGrantRequiresLoginOnlyForInvalidGrant(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		body        string
		wantLogin   bool
		wantSuccess bool
	}{
		{"rejected grant", 400, `{"error":"invalid_grant"}`, true, false},
		{"other client error", 400, `{"error":"invalid_request"}`, false, false},
		{"temporary server error", 503, `{"error":"temporarily_unavailable"}`, false, false},
		{"refreshed grant", 200, `{"access_token":"new-access"}`, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := newClient("https://private.example.test")
			if err != nil {
				t.Fatal(err)
			}
			c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if err := req.ParseForm(); err != nil {
					t.Fatal(err)
				}
				if req.Method != http.MethodPost || req.URL.Path != "/oauth/token" ||
					req.PostForm.Get("grant_type") != "refresh_token" ||
					req.PostForm.Get("refresh_token") != "old-refresh" {
					t.Fatal("unexpected refresh request")
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})
			auth, err := c.refreshDesktopGrant(context.Background(), "old-refresh")
			if errors.Is(err, errSourceGrantRejected) != tc.wantLogin || (err == nil) != tc.wantSuccess {
				t.Fatalf("unexpected refresh result: %v", err)
			}
			if tc.wantSuccess && auth.Access != "new-access" {
				t.Fatal("missing refreshed access token")
			}
		})
	}
}
