package model

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestVisionAnswerMatches(t *testing.T) {
	red, blue := visionColors[0], visionColors[2]
	for answer, want := range map[string]bool{
		"left=red, right=blue":         true,
		"Left = Red; Right = BLUE.":    true,
		"左边是红色，右边是蓝色":                  true,
		"left=blue, right=red":         false,
		"left=red, right=red":          false,
		"I cannot see the image.":      false,
		"left=red or blue, right=blue": false,
	} {
		if got := visionAnswerMatches(answer, red, blue); got != want {
			t.Errorf("%q: got %t, want %t", answer, got, want)
		}
	}
}

// The fake upstream answers from the image it receives, or ignores it the way
// some servers ignore input_image on /v1/responses.
func visionUpstream(t *testing.T, readsImages bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Input []struct {
				Content []struct {
					Type     string `json:"type"`
					ImageURL string `json:"image_url"`
				} `json:"content"`
			} `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !strings.HasSuffix(r.URL.Path, "/responses") {
			t.Errorf("unexpected request %s: %v", r.URL.Path, err)
		}
		answer := "I only received text."
		for _, item := range body.Input {
			for _, part := range item.Content {
				if part.Type != "input_image" || !readsImages {
					continue
				}
				raw, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(part.ImageURL, "data:image/png;base64,"))
				img, err := png.Decode(bytes.NewReader(raw))
				if err != nil {
					t.Fatal(err)
				}
				name := func(x int) string {
					c := img.At(x, 64)
					for _, candidate := range visionColors {
						r1, g1, b1, _ := c.RGBA()
						r2, g2, b2, _ := candidate.RGBA.RGBA()
						if r1 == r2 && g1 == g2 && b1 == b2 {
							return candidate.Name
						}
					}
					return "unknown"
				}
				answer = "left=" + name(10) + ", right=" + name(200)
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"output_text": answer})
	}))
}

func TestModelProbeVision(t *testing.T) {
	for _, readsImages := range []bool{true, false} {
		upstream := visionUpstream(t, readsImages)
		service := NewService(&repositoryStub{models: []Model{{
			ID: "model-1", OwnershipType: "system", Kind: KindText, Protocol: ProtocolOpenAIResponses,
			ModelID: "fixture", BaseURL: upstream.URL + "/v1", APIKey: "test",
		}}})
		router := chi.NewRouter()
		router.Post("/models/{modelID}/probe", service.probeModel)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/models/model-1/probe", strings.NewReader(`{"vision":true}`)))
		upstream.Close()
		var result map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil || recorder.Code != http.StatusOK {
			t.Fatalf("probe failed: %d %s", recorder.Code, recorder.Body)
		}
		if result["vision_passed"] != readsImages || result["vision_expected"] == "" {
			t.Fatalf("readsImages=%t: %+v", readsImages, result)
		}
	}
}
