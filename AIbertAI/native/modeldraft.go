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
	"path"
	"strings"
	"time"
)

type modelDraft struct {
	Protocol string
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	ModelID  string `json:"model_id"`
}

func requestModelDraft(ctx context.Context, raw json.RawMessage, discover bool) (any, error) {
	var draft modelDraft
	if json.Unmarshal(raw, &draft) != nil {
		return nil, errors.New("model_config_invalid")
	}
	upstream, err := url.Parse(strings.TrimSpace(draft.BaseURL))
	if err != nil || upstream.Hostname() == "" || upstream.User != nil || upstream.Fragment != "" ||
		(upstream.Scheme != "http" && upstream.Scheme != "https") ||
		strings.TrimSpace(draft.APIKey) == "" || len(draft.APIKey) > 16<<10 || strings.ContainsAny(draft.APIKey, "\r\n") {
		return nil, errors.New("model_config_invalid")
	}
	var endpoint string
	switch draft.Protocol {
	case "openai_chat_completions":
		endpoint = "chat/completions"
	case "openai_responses":
		endpoint = "responses"
	case "anthropic":
		endpoint = "messages"
	default:
		return nil, errors.New("model_protocol_unsupported")
	}
	method := http.MethodGet
	var reader io.Reader
	if discover {
		endpoint = "models"
	} else {
		if strings.TrimSpace(draft.ModelID) == "" || len(draft.ModelID) > 256 {
			return nil, errors.New("model_config_invalid")
		}
		method = http.MethodPost
		input := map[string]any{"model": draft.ModelID}
		if draft.Protocol == "openai_responses" {
			input["input"], input["max_output_tokens"] = "Reply OK.", 64
		} else {
			input["messages"], input["max_tokens"] = []map[string]string{{"role": "user", "content": "Reply OK."}}, 64
		}
		encoded, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(encoded)
	}
	upstream.Path = path.Join(upstream.Path, endpoint)
	upstream.RawPath = ""
	req, err := http.NewRequestWithContext(ctx, method, upstream.String(), reader)
	if err != nil {
		return nil, errors.New("model_config_invalid")
	}
	if draft.Protocol == "anthropic" {
		req.Header.Set("x-api-key", draft.APIKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else {
		req.Header.Set("Authorization", "Bearer "+draft.APIKey)
	}
	if reader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("model_provider_unreachable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model_provider_http_%d", resp.StatusCode)
	}
	decode := json.NewDecoder(io.LimitReader(resp.Body, 2<<20))
	if discover {
		var results struct {
			Data []struct {
				ID          string `json:"id"`
				DisplayName string `json:"display_name,omitempty"`
			}
		}
		if decode.Decode(&results) != nil || results.Data == nil {
			return nil, errors.New("model_provider_invalid_response")
		}
		return map[string]any{"models": results.Data}, nil
	}
	var output struct {
		Choices []json.RawMessage
		Output  []json.RawMessage
		Content []json.RawMessage
	}
	if decode.Decode(&output) != nil || len(output.Choices)+len(output.Output)+len(output.Content) == 0 {
		return nil, errors.New("model_provider_invalid_response")
	}
	return map[string]any{"ok": true, "model": draft.ModelID}, nil
}
