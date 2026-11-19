package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDesktopRuntimeAccess(t *testing.T) {
	access := desktopRuntimeAccess{
		address: "127.0.0.1:12345", origin: "http://127.0.0.1:1435", protocol: "monkeyai.token.test",
	}
	for _, tc := range []struct {
		name, host, protocol string
		origins              []string
		allowed              bool
	}{
		{"renderer", access.address, "monkeyai.v1, " + access.protocol, []string{access.origin}, true},
		{"desktop main", access.address, "monkeyai.v1, " + access.protocol, nil, true},
		{"foreign origin", access.address, "monkeyai.v1, " + access.protocol, []string{"https://untrusted.example"}, false},
		{"duplicate origin", access.address, "monkeyai.v1, " + access.protocol, []string{access.origin, access.origin}, false},
		{"foreign host", "untrusted.example", "monkeyai.v1, " + access.protocol, []string{access.origin}, false},
		{"missing token", access.address, "monkeyai.v1", []string{access.origin}, false},
		{"wrong token", access.address, "monkeyai.v1, monkeyai.token.other", []string{access.origin}, false},
		{"duplicate token", access.address, "monkeyai.v1, " + access.protocol + ", " + access.protocol, []string{access.origin}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://"+tc.host+"/acp", nil)
			for _, origin := range tc.origins {
				request.Header.Add("Origin", origin)
			}
			request.Header.Set("Sec-WebSocket-Protocol", tc.protocol)
			if allowed := access.authorize(request) == nil; allowed != tc.allowed {
				t.Fatalf("authorization = %v, want %v", allowed, tc.allowed)
			}
		})
	}
}

func TestDesktopRuntimeDisablesRelayHTTP(t *testing.T) {
	host := &relaySessionHost{local: &desktopRuntimeAccess{}}
	for _, path := range []string{"/read/session", "/read/config", "/cap/workspace/mcp"} {
		w := httptest.NewRecorder()
		host.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s returned %d, want 404", path, w.Code)
		}
	}
}
