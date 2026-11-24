package proxy

import (
	"errors"
	"net"
	"net/http"
	"time"
)

const upstreamDialAttempts = 3

// dialRetryTransport retries only failures to open the upstream connection.
// No request bytes reach the model before a dial succeeds, so a retry cannot
// produce a second model call or charge.
type dialRetryTransport struct {
	base http.RoundTripper
}

func (t dialRetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for attempt := 1; ; attempt++ {
		response, err := t.base.RoundTrip(req)
		if err == nil || attempt == upstreamDialAttempts || !upstreamDialFailed(err) || req.GetBody == nil && req.Body != nil && req.Body != http.NoBody {
			return response, err
		}
		select {
		case <-req.Context().Done():
			return nil, err
		case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
		}
		if req.GetBody != nil {
			body, bodyErr := req.GetBody()
			if bodyErr != nil {
				return nil, err
			}
			req = req.Clone(req.Context())
			req.Body = body
		}
	}
}

func upstreamDialFailed(err error) bool {
	var op *net.OpError
	return errors.As(err, &op) && (op.Op == "dial" || op.Op == "proxyconnect")
}
