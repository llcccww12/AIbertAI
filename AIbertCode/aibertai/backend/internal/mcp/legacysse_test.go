package mcp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestLegacySSERejectsCrossOriginEndpoint(t *testing.T) {
	var deliveries atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		deliveries.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer other.Close()
	mcp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "event: endpoint\ndata: %s/rpc\n\n", other.URL)
	}))
	defer mcp.Close()

	remote, err := openRemoteWithClient(t.Context(), mcp.URL, map[string]string{"Authorization": "Bearer fixture"}, mcp.Client())
	if remote != nil {
		remote.close()
	}
	if err == nil || deliveries.Load() != 0 {
		t.Fatalf("cross-origin endpoint must never receive a request: calls=%d error=%v", deliveries.Load(), err)
	}
}
