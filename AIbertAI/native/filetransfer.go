package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var transferTicket = regexp.MustCompile(`^[A-Za-z0-9_-]{32,256}$`)
var transferRange = regexp.MustCompile(`^bytes=([0-9]+-[0-9]*|-[0-9]+)$`)
var transferSession = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

type fileTransferInput struct {
	Server, FilesOrigin, MachineID, URL, Method string
	Ticket, Range, SessionID                    string
	Offset                                      int64
	Chunk                                       []byte
}

func transferOrigin(raw string) (*url.URL, bool) {
	address, err := url.Parse(raw)
	if err != nil || address.Hostname() == "" || address.User != nil || address.Opaque != "" ||
		address.RawQuery != "" || address.ForceQuery || address.Fragment != "" ||
		(address.Scheme != "https" && address.Scheme != "http") {
		return nil, false
	}
	return address, true
}

func sameOrigin(a, b *url.URL) bool { return a.Scheme == b.Scheme && a.Host == b.Host }

func fileTransferURL(in fileTransferInput) (*url.URL, error) {
	bad := errors.New("invalid_file_transfer")
	server, ok := transferOrigin(in.Server)
	if !ok || (server.Path != "" && server.Path != "/") || !validUUID(in.MachineID) || !transferTicket.MatchString(in.Ticket) {
		return nil, bad
	}
	if backendOrigin != "" && (strings.TrimSuffix(in.Server, "/") != backendOrigin ||
		(in.FilesOrigin != "" && strings.TrimSuffix(in.FilesOrigin, "/") != backendOrigin)) {
		return nil, bad
	}
	target, ok := transferOrigin(in.URL)
	if !ok {
		return nil, bad
	}
	prefix := "/api/v1/endpoints/" + in.MachineID + "/proxy/"
	switch in.Method {
	case http.MethodPut:
		id, found := strings.CutPrefix(target.Path, prefix+"uploads/")
		if !sameOrigin(server, target) || !found || !validUUID(id) || target.RawPath != "" || in.Offset < 0 || len(in.Chunk) == 0 || len(in.Chunk) > 1<<20 ||
			!transferSession.MatchString(in.SessionID) {
			return nil, bad
		}
	case http.MethodGet, http.MethodHead:
		if in.Range != "" && !transferRange.MatchString(in.Range) {
			return nil, bad
		}
		prefix += "files/"
		if !sameOrigin(server, target) || !strings.HasPrefix(target.EscapedPath(), prefix) {
			files, valid := transferOrigin(in.FilesOrigin)
			if !valid || (files.Path != "" && files.Path != "/") || files.Scheme != server.Scheme || !sameOrigin(files, target) {
				return nil, bad
			}
			prefix = "/endpoints/" + in.MachineID + "/files/"
		}
		suffix, found := strings.CutPrefix(target.EscapedPath(), prefix)
		grant, filePath, hasFile := strings.Cut(suffix, "/")
		if !found || !hasFile || !strings.HasPrefix(grant, "fg1.") || len(grant) > 12<<10 {
			return nil, bad
		}
		for _, component := range strings.Split(filePath, "/") {
			part, err := url.PathUnescape(component)
			if err != nil || part == "" || part == "." || part == ".." || strings.ContainsAny(part, "/\\\x00\r\n") {
				return nil, bad
			}
		}
	default:
		return nil, bad
	}
	return target, nil
}

func streamFileTransfer(ctx context.Context, input io.Reader, output io.Writer) error {
	var in fileTransferInput
	if err := json.NewDecoder(io.LimitReader(input, 2<<20)).Decode(&in); err != nil {
		return errors.New("invalid_file_transfer")
	}
	address, err := fileTransferURL(in)
	if err != nil {
		return err
	}
	var body io.Reader
	if in.Method == http.MethodPut {
		body = bytes.NewReader(in.Chunk)
	}
	req, err := http.NewRequestWithContext(ctx, in.Method, address.String(), body)
	if err != nil {
		return errors.New("invalid_file_transfer")
	}
	req.Header.Set("Authorization", "Bearer "+in.Ticket)
	req.Header.Set("Accept-Encoding", "identity")
	if in.Range != "" {
		req.Header.Set("Range", in.Range)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Upload-Session-ID", in.SessionID)
		req.Header.Set("Upload-Offset", strconv.FormatInt(in.Offset, 10))
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	defer transport.CloseIdleConnections()
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 120 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("file_transfer_failed")
	}
	defer resp.Body.Close()
	meta := struct {
		Status  int         `json:"status"`
		Headers http.Header `json:"headers"`
	}{resp.StatusCode, make(http.Header)}
	for _, header := range []string{
		"Content-Type", "Content-Length", "Content-Range", "Content-Encoding", "Accept-Ranges",
		"Upload-Offset", "Upload-Length", "Upload-State", "ETag", "Last-Modified",
	} {
		if values := resp.Header.Values(header); len(values) != 0 {
			meta.Headers[header] = values
		}
	}
	if err := json.NewEncoder(output).Encode(meta); err != nil {
		return err
	}
	if in.Method == http.MethodGet && (resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent) {
		_, err = io.Copy(output, resp.Body)
	}
	return err
}
