package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCloudGatewayRoutes(t *testing.T) {
	const origin = "http://192.168.20.2:5173"
	gateway, err := cloudGateway("172.18.0.1:5580", origin, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "Bearer fixture" {
			t.Error("incorrect device or browser credential forwarding")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, uri string
		allow       bool
	}{
		{"GET", "/.well-known/oauth-authorization-server", true},
		{"POST", "/oauth/device/authorization", true},
		{"POST", "/oauth/token", true},
		{"GET", "/api/v1/endpoint-bridge?channel_kind=control", true},
		{"GET", "/api/v1/models", true},
		{"POST", "/v1/chat/completions", true},
		{"GET", "/api/admin/v1/models", false},
		{"GET", "/api/v1/%2e%2e/admin/v1/models", false},
		{"GET", "/api/v1//../admin/", false},
		{"GET", "/console", false},
		{"POST", "/oauth/authorizations", false},
		{"POST", "/api/v1/environments", false},
		{"POST", "/api/v1/environments/123/start", false},
		{"CONNECT", "/api/v1/endpoint-bridge", false},
	} {
		t.Run(tc.method+tc.uri, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.uri, nil)
			req.Host = "192.168.20.2:5173"
			req.Header.Set("Cookie", "session=browser")
			req.Header.Set("Authorization", "Bearer fixture")
			res := httptest.NewRecorder()
			gateway.Handler.ServeHTTP(res, req)
			if (res.Code == http.StatusNoContent) != tc.allow {
				t.Fatalf("HTTP %d", res.Code)
			}
		})
	}
	req := httptest.NewRequest("GET", "/api/v1/models", nil)
	res := httptest.NewRecorder()
	gateway.Handler.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatal("gateway accepted a different Host")
	}
	if _, err := cloudGateway("0.0.0.0:5580", origin, nil); err == nil {
		t.Fatal("gateway accepted a wildcard bind address")
	}
}
