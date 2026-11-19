package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *client) requestModelRound(ctx context.Context, path, key, session string, request map[string]any) (*http.Response, error) {
	send := func() (*http.Response, error) {
		data, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		return c.request(ctx, http.MethodPost, path, key, "", session, "application/json", bytes.NewReader(data),
			http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusNotImplemented)
	}
	resp, err := send()
	if err != nil {
		return nil, err
	}
	switch resp.StatusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusNotImplemented:
		rejected := request["stream"] == true && streamRejected(resp.Body)
		resp.Body.Close()
		if !rejected {
			return nil, fmt.Errorf("POST %s: HTTP %d", path, resp.StatusCode)
		}
		request["stream"] = false
		delete(request, "stream_options")
		data, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		return c.request(ctx, http.MethodPost, path, key, "", session, "application/json", bytes.NewReader(data))
	default:
		return resp, nil
	}
}

func streamRejected(body io.Reader) bool {
	var payload struct {
		Error struct{ Param, Code, Message string }
	}
	if json.NewDecoder(io.LimitReader(body, 64<<10)).Decode(&payload) != nil {
		return false
	}
	failure := payload.Error
	message := strings.ToLower(strings.TrimSpace(failure.Message))
	return failure.Code == "stream_not_supported" || failure.Code == "streaming_not_supported" ||
		(failure.Param == "stream" && (strings.Contains(message, "not support") || strings.Contains(message, "unsupported"))) ||
		message == "streaming is not supported" || message == "this model does not support streaming"
}
