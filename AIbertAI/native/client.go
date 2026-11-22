package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const clientID = "monkeyai-desktop"

var (
	backendOrigin string
	modelID       string
)

type backendHTTPError struct {
	Method, Path string
	Status       int
}

func (e *backendHTTPError) Error() string {
	return fmt.Sprintf("%s %s: HTTP %d", e.Method, e.Path, e.Status)
}

type client struct {
	server string
	http   *http.Client
}

func newClient(raw string) (*client, error) {
	return newBackendClient(raw, false)
}

func newBackendClient(raw string, configuredHTTP bool) (*client, error) {
	origin, err := url.Parse(raw)
	if err != nil || origin.Hostname() == "" || origin.User != nil ||
		origin.Path != "" || origin.RawQuery != "" || origin.ForceQuery || origin.Fragment != "" ||
		(origin.Scheme != "https" && !(origin.Scheme == "http" && (configuredHTTP || origin.Hostname() == "127.0.0.1"))) {
		return nil, errors.New("后端必须是无路径的 HTTPS 地址，本机验收可用 127.0.0.1 HTTP")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &client{origin.String(), &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 3 * time.Minute,
	}}, nil
}

func (c *client) request(ctx context.Context, method, path, token, machine, session, mime string, in io.Reader, accepted ...int) (*http.Response, error) {
	return c.requestWithHeaders(ctx, method, path, token, machine, session, mime, in, nil, accepted...)
}

func (c *client) requestWithHeaders(ctx context.Context, method, path, token, machine, session, mime string, in io.Reader, headers http.Header, accepted ...int) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.server+path, in)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if machine != "" {
		req.Header.Set("X-MAI-Machine-ID", machine)
	}
	if session != "" {
		req.Header.Set("X-MAI-Session-ID", session)
	}
	if mime != "" {
		req.Header.Set("Content-Type", mime)
	}
	for name, values := range headers {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, req.URL.Path, err)
	}
	if (resp.StatusCode < 200 || resp.StatusCode >= 300) && !slices.Contains(accepted, resp.StatusCode) {
		resp.Body.Close()
		return nil, &backendHTTPError{method, req.URL.Path, resp.StatusCode}
	}
	return resp, nil
}

func (c *client) json(ctx context.Context, method, path, token, machine string, in, out any) error {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	resp, err := c.request(ctx, method, path, token, machine, "", "application/json", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(out)
}

func (c *client) form(ctx context.Context, path string, fields url.Values, out any) error {
	resp, err := c.request(ctx, http.MethodPost, path, "", "", "", "application/x-www-form-urlencoded", strings.NewReader(fields.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 32<<10)).Decode(out)
}
