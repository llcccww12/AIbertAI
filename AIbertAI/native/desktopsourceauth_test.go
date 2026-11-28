package main

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
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
			auth, err := c.refreshSourceGrant(context.Background(), clientID, "old-refresh")
			if errors.Is(err, errSourceGrantRejected) != tc.wantLogin || (err == nil) != tc.wantSuccess {
				t.Fatalf("unexpected refresh result: %v", err)
			}
			if tc.wantSuccess && auth.Access != "new-access" {
				t.Fatal("missing refreshed access token")
			}
		})
	}
}

func TestDesktopGrantRefreshAfterUnauthorized(t *testing.T) {
	const grantID = "11111111-1111-4111-8111-111111111111"
	for _, rejected := range []bool{false, true} {
		t.Run(map[bool]string{false: "renewed", true: "rejected"}[rejected], func(t *testing.T) {
			c, err := newClient("https://private.example.test")
			if err != nil {
				t.Fatal(err)
			}
			store, err := openSourceGrantStore(t.TempDir(), base64.StdEncoding.EncodeToString(make([]byte, 32)), c.server, grantID)
			if err != nil {
				t.Fatal(err)
			}
			defer store.root.Close()
			refreshes, notifications := 0, 0
			c.http.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
				refreshes++
				if req.URL.Path != "/oauth/token" {
					t.Fatalf("unexpected request %s", req.URL.Path)
				}
				if rejected {
					return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`))}, nil
				}
				body := `{"access_token":"new","refresh_token":"rotated","grant_id":"` + grantID + `","expires_in":3600,"scope":"device:bind session:report"}`
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
			})
			source := &desktopSourceAuth{client: c, store: store,
				grant:      sourceGrant{Auth: authorization{Grant: grantID, Access: "old", Refresh: "refresh"}, Expires: time.Now().Add(time.Hour)},
				onRejected: func() { notifications++ },
			}
			calls := 0
			err = withAuthRetry(t.Context(), source, func(_ context.Context, auth authorization) error {
				calls++
				if auth.Access == "old" {
					return &backendHTTPError{Status: http.StatusUnauthorized}
				}
				if auth.Access != "new" {
					t.Fatalf("unexpected access token %q", auth.Access)
				}
				return nil
			})
			if rejected {
				if !errors.Is(err, errSourceGrantRejected) || calls != 1 || notifications != 1 {
					t.Fatalf("rejected grant: err=%v, calls=%d, notifications=%d", err, calls, notifications)
				}
				if _, err := source.access(t.Context()); !errors.Is(err, errSourceGrantRejected) {
					t.Fatalf("rejected grant was reused: %v", err)
				}
			} else {
				if err != nil || calls != 2 || notifications != 0 {
					t.Fatalf("refresh: err=%v, calls=%d, notifications=%d", err, calls, notifications)
				}
				if source.grant.Expires != source.grant.Expires.Round(0) {
					t.Fatal("grant expiry uses a monotonic clock that can pause during sleep")
				}
				persisted, err := store.load()
				if err != nil || persisted.Auth.Refresh != "rotated" {
					t.Fatalf("rotated grant was not saved: %v", err)
				}
				if auth, err := source.refreshForce(t.Context(), "old"); err != nil || auth.Access != "new" {
					t.Fatalf("concurrent 401 did not reuse renewed grant: %v", err)
				}
			}
			if refreshes != 1 {
				t.Fatalf("expected one refresh, got %d", refreshes)
			}
		})
	}
}
