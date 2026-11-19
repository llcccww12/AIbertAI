package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

func desktopServiceResult(resp *http.Response, err error) (json.RawMessage, string) {
	if err != nil {
		return nil, "unavailable"
	}
	defer resp.Body.Close()
	var data json.RawMessage
	if json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&data) != nil {
		return nil, "unavailable"
	}
	if resp.StatusCode < 300 {
		return data, ""
	}
	var failure struct{ Error struct{ Code string } }
	if json.Unmarshal(data, &failure) == nil && knowledgeErrorCode.MatchString(failure.Error.Code) {
		return nil, failure.Error.Code
	}
	return nil, "unavailable"
}

func (h *relaySessionHost) desktopEnvironment(ctx context.Context, method string, raw json.RawMessage) (any, error) {
	failure := func(code string, uncertain bool) (any, error) {
		return map[string]any{"data": nil, "error": map[string]any{"code": code, "uncertain": uncertain}}, nil
	}
	var input struct {
		ServerURL, UserID, ID, Name, Action string
		Page, Generation                    int64
		PageSize                            int64  `json:"page_size"`
		RequestKey                          string `json:"request_key"`
	}
	if json.Unmarshal(raw, &input) != nil || input.UserID != h.local.userID || input.ServerURL != h.live.client.server {
		return failure("invalid_request", false)
	}
	methodHTTP, path := http.MethodGet, "/api/v1/environments"
	var body io.Reader
	switch method {
	case "_monkeyai/environments/get", "_monkeyai/environments/operations":
		if !validUUID(input.ID) {
			return failure("invalid_request", false)
		}
		path += "/" + input.ID
		if method == "_monkeyai/environments/operations" {
			path += "/operations"
		}
	case "_monkeyai/environments/change":
		if input.RequestKey == "" || len(input.RequestKey) > 128 {
			return failure("invalid_request", false)
		}
		methodHTTP = http.MethodPost
		fields := map[string]any{"request_key": input.RequestKey}
		if input.Action == "create" {
			fields["name"] = input.Name
		} else {
			if !validUUID(input.ID) || input.Generation < 1 || input.Generation > maxGoalBudget {
				return failure("invalid_request", false)
			}
			path += "/" + input.ID
			switch input.Action {
			case "start", "stop":
				path += "/" + input.Action
			case "delete":
				methodHTTP = http.MethodDelete
			default:
				return failure("invalid_request", false)
			}
			fields["generation"] = input.Generation
		}
		data, _ := json.Marshal(fields)
		body = bytes.NewReader(data)
	}
	if method == "_monkeyai/environments/list" || method == "_monkeyai/environments/operations" {
		if input.Page < 1 || input.Page > 100000 || input.PageSize < 1 || input.PageSize > 100 {
			return failure("invalid_request", false)
		}
		query := url.Values{"page": {strconv.FormatInt(input.Page, 10)}, "page_size": {strconv.FormatInt(input.PageSize, 10)}}
		path += "?" + query.Encode()
	}
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return failure("permission_denied", false)
	}
	resp, err := h.live.client.request(ctx, methodHTTP, path, auth.Access, "", "", "application/json", body,
		400, 401, 403, 404, 409, 412, 422, 429, 500, 502, 503, 504)
	data, code := desktopServiceResult(resp, err)
	if code != "" {
		return failure(code, methodHTTP != http.MethodGet && (err != nil || resp.StatusCode >= 500))
	}
	return map[string]any{"data": data}, nil
}

const desktopFeedbackRoute = "/_aibertai/feedback"
const maxFeedbackImageBytes = 5 << 20
const maxFeedbackJSON = 21 << 20

func (h *relaySessionHost) serveDesktopFeedback(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(h.local.protocol, "monkeyai.token.")
	if r.Method != http.MethodPost || r.Host != h.local.address || len(r.Header.Values("Origin")) != 0 ||
		subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+token)) != 1 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	var input struct {
		Category, Content, IdempotencyKey string
		Rating                            int
		Images                            []string
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxFeedbackJSON))
	if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF ||
		len(input.Images) > 3 || utf8.RuneCountInString(input.Content) > 5000 ||
		input.Rating < 0 || input.Rating > 5 || len(input.IdempotencyKey) > 128 ||
		strings.TrimSpace(input.Content) == "" && len(input.Images) == 0 {
		_ = encoder.Encode(map[string]string{"error": "invalid_request"})
		return
	}
	auth, err := h.live.authorization(r.Context())
	if err != nil {
		_ = encoder.Encode(map[string]string{"error": "unauthorized"})
		return
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	fields := map[string]string{"category": input.Category, "content": input.Content, "platform": "desktop", "client_version": h.local.clientVersion}
	if input.Rating != 0 {
		fields["rating"] = strconv.Itoa(input.Rating)
	}
	for key, value := range fields {
		_ = form.WriteField(key, value)
	}
	for index, image := range input.Images {
		header, content, ok := strings.Cut(image, ",")
		extension := map[string]string{"data:image/png;base64": "png", "data:image/jpeg;base64": "jpeg", "data:image/webp;base64": "webp"}[header]
		if !ok || extension == "" || len(content) > base64.StdEncoding.EncodedLen(maxFeedbackImageBytes) {
			_ = encoder.Encode(map[string]string{"error": "invalid_images"})
			return
		}
		data, err := base64.StdEncoding.DecodeString(content)
		if err != nil || len(data) == 0 || len(data) > maxFeedbackImageBytes {
			_ = encoder.Encode(map[string]string{"error": "invalid_images"})
			return
		}
		part, _ := form.CreateFormFile("images", strconv.Itoa(index+1)+"."+extension)
		_, _ = part.Write(data)
	}
	_ = form.Close()
	headers := http.Header{}
	if input.IdempotencyKey != "" {
		headers.Set("Idempotency-Key", input.IdempotencyKey)
	}
	resp, err := h.live.client.requestWithHeaders(r.Context(), http.MethodPost, "/api/v1/feedback", auth.Access, "", "",
		form.FormDataContentType(), &body, headers, 400, 401, 403, 409, 413, 422, 429, 500, 503)
	data, code := desktopServiceResult(resp, err)
	if code != "" {
		_ = encoder.Encode(map[string]string{"error": code})
		return
	}
	_ = encoder.Encode(map[string]any{"data": data})
}
