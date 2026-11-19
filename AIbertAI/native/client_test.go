package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestPrivateBackendOrigin(t *testing.T) {
	for _, origin := range []string{
		"http://other.example.test", "https://user@example.test", "https://example.test/path",
		"https://example.test?token=test", "https://example.test/#fragment", "file:///etc/hosts",
	} {
		if _, err := newClient(origin); err == nil {
			t.Errorf("accepted invalid backend origin %q", origin)
		}
	}
	for _, origin := range []string{"https://private.example.test", "http://127.0.0.1:5173"} {
		if _, err := newClient(origin); err != nil {
			t.Errorf("rejected private backend %q: %v", origin, err)
		}
	}
}

func TestPrivateBackendRefusesRedirect(t *testing.T) {
	c, err := newClient("https://private.example.test")
	if err != nil {
		t.Fatal(err)
	}
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "private.example.test" {
			t.Fatalf("contacted redirect target %v", r.URL)
		}
		return &http.Response{
			StatusCode: http.StatusTemporaryRedirect,
			Header:     http.Header{"Location": {"https://external.example.test/oauth/token"}},
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})
	if _, err := c.request(context.Background(), http.MethodGet, "/oauth/token", "fixture", "", "", "", nil); err == nil {
		t.Fatal("redirect treated as a success")
	}
}
