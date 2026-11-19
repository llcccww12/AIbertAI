package model

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestModelProbeStreamsBeforeCompletion(t *testing.T) {
	release := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !probeStreamRequested(t, r) {
			t.Error("legacy model must request streaming by default")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"private\"}}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"first\"}}]}\n\n")
		http.NewResponseController(w).Flush()
		select {
		case <-release:
			io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\" second\"},\"finish_reason\":\"stop\"}]}\n\n")
		case <-r.Context().Done():
		}
	}))
	defer upstream.Close()
	server := httptest.NewServer(probeStreamRouter(upstream.URL, ""))
	defer server.Close()
	defer finish()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/models/model-1/probe", strings.NewReader(`{"message":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Accept", "application/x-ndjson, application/json")
	response, err := server.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "application/x-ndjson") {
		t.Fatalf("unexpected response type: %s", response.Header.Get("Content-Type"))
	}
	decode := json.NewDecoder(response.Body)
	read := func() map[string]any {
		t.Helper()
		var event map[string]any
		if err := decode.Decode(&event); err != nil {
			t.Fatal(err)
		}
		return event
	}
	start, first := read(), read()
	if start["output_mode"] != "stream" || first["content"] != "first" || first["first_text_ms"] == nil {
		t.Fatalf("missing mode or live first text: %+v %+v", start, first)
	}
	finish()
	second, done := read(), read()
	if second["content"] != " second" || done["type"] != "done" || done["valid"] != true || done["total_ms"] == nil {
		t.Fatalf("missing streamed completion: %+v %+v", second, done)
	}
}

func TestModelProbeHonorsSavedModeAndExplicitRejection(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mode       StreamMode
		errorBody  string
		wantStream []bool
	}{
		{"disabled", StreamOff, "", []bool{false}},
		{"JSON despite requested stream", StreamAuto, "", []bool{true}},
		{"explicit unsupported stream", StreamAuto, `{"error":{"code":"streaming_not_supported"}}`, []bool{true, false}},
		{"unrelated rejection", StreamAuto, `{"error":{"param":"model","message":"unsupported model"}}`, []bool{true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			streams := make(chan bool, 3)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				stream := probeStreamRequested(t, r)
				streams <- stream
				w.Header().Set("Content-Type", "application/json")
				if stream && tc.errorBody != "" {
					w.WriteHeader(http.StatusBadRequest)
					io.WriteString(w, tc.errorBody)
					return
				}
				io.WriteString(w, `{"choices":[{"message":{"content":"answer"}}]}`)
			}))
			defer upstream.Close()
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/models/model-1/probe", strings.NewReader(`{"message":"hello"}`))
			request.Header.Set("Accept", "application/x-ndjson")
			probeStreamRouter(upstream.URL, tc.mode).ServeHTTP(response, request)
			close(streams)
			var got []bool
			for stream := range streams {
				got = append(got, stream)
			}
			if !slices.Equal(got, tc.wantStream) {
				t.Fatalf("stream policy = %v, want %v", got, tc.wantStream)
			}
			if tc.name == "unrelated rejection" {
				if response.Code != http.StatusBadGateway {
					t.Fatalf("rejection status: %d", response.Code)
				}
				return
			}
			var result map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK || result["output_mode"] != "buffered" || result["content"] != "answer" ||
				result["first_text_ms"] == nil || result["first_text_ms"] != result["total_ms"] {
				t.Fatalf("invalid buffered response: %d %+v", response.Code, result)
			}
		})
	}
}

func probeStreamRouter(upstream string, mode StreamMode) http.Handler {
	service := NewService(&repositoryStub{models: []Model{{
		ID: "model-1", OwnershipType: "system", Kind: KindText, Protocol: ProtocolOpenAIChat,
		ModelID: "fixture", BaseURL: upstream, APIKey: "test", AdvancedConfig: AdvancedConfig{StreamMode: mode},
	}}})
	router := chi.NewRouter()
	router.Post("/models/{modelID}/probe", service.probeModel)
	return router
}

func probeStreamRequested(t *testing.T, r *http.Request) bool {
	t.Helper()
	var body struct{ Stream bool }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Error(err)
	}
	return body.Stream
}
