package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDesktopStartupWaitsForStagedServices(t *testing.T) {
	const user = "11111111-1111-4111-8111-111111111111"
	const server = "22222222-2222-4222-8222-222222222222"
	const origin = "https://employee.example.test"
	models := `{"total_count":1,"models":[{"id":"33333333-3333-4333-8333-333333333333","kind":"text","protocol":"openai_chat_completions"}]}`
	for _, stage := range []struct {
		name, failedPath string
		emptyCatalog     bool
	}{
		{"account responds before model catalog", "/api/v1/models", false},
		{"empty interim model catalog", "/api/v1/models", true},
		{"account and model catalog respond before device bridge", "/api/v1/endpoint-bridge", false},
	} {
		t.Run(stage.name, func(t *testing.T) {
			c, err := newClient(origin)
			if err != nil {
				t.Fatal(err)
			}
			reached := false
			c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				body, status := "", http.StatusOK
				if r.URL.Path == stage.failedPath && !reached {
					reached = true
					if stage.emptyCatalog {
						body = `{"total_count":0,"models":[]}`
					} else {
						status = http.StatusServiceUnavailable
					}
				} else {
					switch r.URL.Path {
					case "/api/v1/me":
						body = fmt.Sprintf(`{"id":%q}`, user)
					case "/api/v1/models":
						body = models
					case "/.well-known/oauth-authorization-server":
						body = fmt.Sprintf(`{"issuer":%q,"monkeyai_runtime":{"server_id":%q,"bridge_url":"wss://employee.example.test/api/v1/endpoint-bridge","access_signing_keys":[{"kid":"test","algorithm":"Ed25519","public_key":%q}]}}`,
							origin, server, base64.RawURLEncoding.EncodeToString(make([]byte, 32)))
					default:
						t.Errorf("startup advanced to %s while %s was unavailable", r.URL.Path, stage.failedPath)
						status = http.StatusServiceUnavailable
					}
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})
			source := &desktopSourceAuth{client: c, grant: sourceGrant{Auth: authorization{Access: "test"}, Expires: time.Now().Add(time.Hour)}}
			key := base64.StdEncoding.EncodeToString(make([]byte, 32))
			connection, err := prepareDesktopStartup(t.Context(), source, key, user)
			if !reached || err == nil || !desktopNetworkUnavailable(err) || connection.key.ID != "" {
				t.Fatalf("startup must remain offline. Service reached: %v. Error: %v", reached, err)
			}
		})
	}
}
