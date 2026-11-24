package proxy

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
)

type roundTripCounter struct {
	calls  int
	bodies []string
	fail   func(int) error
}

func (c *roundTripCounter) RoundTrip(req *http.Request) (*http.Response, error) {
	c.calls++
	data, _ := io.ReadAll(req.Body)
	c.bodies = append(c.bodies, string(data))
	if err := c.fail(c.calls); err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

func newRetryRequest(t *testing.T) *http.Request {
	t.Helper()
	body := []byte(`{"model":"m"}`)
	req, err := http.NewRequest(http.MethodPost, "http://upstream.test/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
	return req
}

func TestDialRetryTransportRetriesConnectionFailures(t *testing.T) {
	base := &roundTripCounter{fail: func(n int) error {
		if n < 3 {
			return &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("i/o timeout")}
		}
		return nil
	}}
	resp, err := dialRetryTransport{base: base}.RoundTrip(newRetryRequest(t))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("response=%v err=%v", resp, err)
	}
	if base.calls != 3 {
		t.Fatalf("calls=%d", base.calls)
	}
	for _, body := range base.bodies {
		if body != `{"model":"m"}` {
			t.Fatalf("retry lost request body: %q", base.bodies)
		}
	}
}

func TestDialRetryTransportStopsAfterLimit(t *testing.T) {
	base := &roundTripCounter{fail: func(int) error { return &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("refused")} }}
	if _, err := (dialRetryTransport{base: base}).RoundTrip(newRetryRequest(t)); err == nil || base.calls != upstreamDialAttempts {
		t.Fatalf("calls=%d err=%v", base.calls, err)
	}
}

func TestDialRetryTransportDoesNotRetryAfterDispatch(t *testing.T) {
	base := &roundTripCounter{fail: func(int) error { return &net.OpError{Op: "read", Net: "tcp", Err: errors.New("reset")} }}
	if _, err := (dialRetryTransport{base: base}).RoundTrip(newRetryRequest(t)); err == nil || base.calls != 1 {
		t.Fatalf("calls=%d err=%v", base.calls, err)
	}
}
