package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

type legacySSE struct {
	postURL string
	body    io.ReadCloser
	scanner *bufio.Scanner
}

func (c *remoteClient) openLegacySSE(ctx context.Context) (*legacySSE, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.target, nil)
	if err != nil {
		return nil, err
	}
	for name, value := range c.headers {
		req.Header.Set(name, value)
	}
	req.Header.Set("Accept", "text/event-stream")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	stream := &legacySSE{body: res.Body, scanner: bufio.NewScanner(io.LimitReader(res.Body, 32<<20))}
	stream.scanner.Buffer(make([]byte, 4096), 4<<20)
	defer func() {
		if stream.postURL == "" {
			_ = stream.body.Close()
		}
	}()
	if res.StatusCode != http.StatusOK {
		return nil, remoteStatus(res.StatusCode)
	}
	media, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if err != nil || media != "text/event-stream" {
		return nil, failure("mcp_protocol_invalid", nil)
	}
	event, data, err := stream.next()
	if err != nil {
		return nil, err
	}
	post, err := req.URL.Parse(strings.TrimSpace(data))
	if event != "endpoint" || err != nil || !sameMCPEndpointOrigin(req.URL, post) {
		return nil, failure("mcp_protocol_invalid", nil)
	}
	stream.postURL = post.String()
	return stream, nil
}

func sameMCPEndpointOrigin(base, post *url.URL) bool {
	return post != nil && validURL(post.String()) && base.Scheme == post.Scheme && strings.EqualFold(base.Host, post.Host)
}

func (s *legacySSE) next() (string, string, error) {
	var event string
	var data strings.Builder
	for s.scanner.Scan() {
		line := s.scanner.Text()
		if line == "" {
			if data.Len() != 0 {
				return event, data.String(), nil
			}
			event = ""
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "event":
			event = value
		case "data":
			if data.Len()+len(value)+1 > 4<<20 {
				return "", "", failure("mcp_protocol_invalid", nil)
			}
			data.WriteString(value)
			data.WriteByte('\n')
		}
	}
	return "", "", upstreamFailure(s.scanner.Err(), "mcp_protocol_invalid")
}

func (s *legacySSE) response(id int) ([]byte, error) {
	for {
		event, data, err := s.next()
		if err != nil {
			return nil, err
		}
		if event != "message" && event != "" {
			continue
		}
		var candidate struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if json.Unmarshal([]byte(data), &candidate) == nil && candidate.ID == id && candidate.Method == "" {
			return []byte(data), nil
		}
	}
}
