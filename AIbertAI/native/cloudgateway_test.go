package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
)

func TestCloudBackendDialRemapsOnlyCanonicalOrigin(t *testing.T) {
	original := http.DefaultTransport
	defer func() { http.DefaultTransport = original }()
	var destination string
	http.DefaultTransport = &http.Transport{DialContext: func(_ context.Context, _, address string) (net.Conn, error) {
		destination = address
		return nil, errors.New("fixture")
	}}
	c, err := newCloudBackendClient("http://192.168.20.2:5173", "172.18.0.1:58176")
	if err != nil {
		t.Fatal(err)
	}
	dial := c.http.Transport.(*http.Transport).DialContext
	_, _ = dial(t.Context(), "tcp", "192.168.20.2:5173")
	if destination != "172.18.0.1:58176" || c.server != "http://192.168.20.2:5173" {
		t.Fatalf("wrong destination or rewritten identity: %s / %s", destination, c.server)
	}
	destination = ""
	_, _ = dial(t.Context(), "tcp", "192.168.20.2:8080")
	if destination != "" {
		t.Fatal("unexpected destination was dialed")
	}
}
