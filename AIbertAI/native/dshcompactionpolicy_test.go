package main

import (
	"encoding/json"
	"math"
	"testing"
)

// resolveSDKThreshold is compaction-basic's resolveCompactSpec: the trigger is
// floor(min(W*ratio, W-reserved-headroom)), retention floor((W-reserved)*0.16).
func resolveSDKThreshold(window, reserved int, ratio float64, headroom int) (threshold, retain int) {
	budget := window - reserved
	threshold = int(math.Floor(math.Min(float64(window)*ratio, float64(budget-headroom))))
	return threshold, int(math.Floor(float64(budget) * 0.16))
}

func TestDSHCompactionStartsAtEightyFivePercent(t *testing.T) {
	for _, tc := range []struct{ window, output int }{
		{200_000, 0}, {200_000, 8_192}, {200_000, 32_768}, {200_000, 64_000}, {200_000, 400_000},
		{128_000, 64_000}, {1_000_000, 384_000}, {32_768, 2_048}, {8_192, 4_096},
	} {
		model := selectedModelConfig{ID: "m", Protocol: "openai_chat_completions", ContextWindowTokens: tc.window, MaxOutputTokens: tc.output}
		gateway := &dshLocalGateway{model: model.ID, protocol: model.Protocol}
		data, err := gateway.Patch("http://127.0.0.1:12345/v1", model)
		if err != nil {
			t.Fatal(err)
		}
		var rows []struct {
			ID     string `json:"id"`
			Config struct {
				Providers map[string]struct {
					DefaultMaxTokens int `json:"defaultMaxTokens"`
					Models           []struct {
						ContextWindow int  `json:"contextWindow"`
						MaxTokens     *int `json:"maxTokens"`
					} `json:"models"`
				} `json:"providers"`
				ModelPolicies []struct {
					Provider       string  `json:"provider"`
					Model          string  `json:"model"`
					ThresholdRatio float64 `json:"thresholdRatio"`
					HeadroomTokens int     `json:"headroomTokens"`
					MaxTokens      int     `json:"maxTokens"`
				} `json:"modelPolicies"`
			} `json:"config"`
		}
		if err := json.Unmarshal(data, &rows); err != nil {
			t.Fatal(err)
		}
		var route, policy bool
		for _, row := range rows {
			switch row.ID {
			case "llm-pi-ai":
				provider := row.Config.Providers["aibertai"]
				entry := provider.Models[0]
				// A configured entry cap would become the SDK's fixed reservation again.
				if entry.MaxTokens != nil || entry.ContextWindow != tc.window {
					t.Fatalf("%+v: model entry %+v", tc, entry)
				}
				want := tc.output
				if want >= tc.window {
					want = tc.window / 4
				}
				if provider.DefaultMaxTokens != want {
					t.Fatalf("%+v: output capability %d, want %d", tc, provider.DefaultMaxTokens, want)
				}
				route = true
			case "compaction-basic":
				p := row.Config.ModelPolicies[0]
				if p.Provider != "aibertai" || p.Model != "m" || p.MaxTokens <= 0 || p.MaxTokens > dshCompactionSummaryCap {
					t.Fatalf("%+v: policy %+v", tc, p)
				}
				threshold, retain := resolveSDKThreshold(tc.window, 0, p.ThresholdRatio, p.HeadroomTokens)
				if threshold != dshCompactionThreshold(tc.window, tc.output) {
					t.Fatalf("%+v: threshold %d disagrees with the host mirror", tc, threshold)
				}
				// 85% wherever the window can afford it (80K and more).
				if want := tc.window * 85 / 100; tc.window >= 81_920 && threshold != want {
					t.Fatalf("%+v: threshold %d, want %d", tc, threshold, want)
				}
				// Just below the trigger, pi-ai still leaves real output room
				// (window - prompt - 4096), up to 8K or the configured cap.
				room := tc.window - threshold - 4096
				if want := min(8192, max(tc.output, 1)); tc.output == 0 {
					want = 8192
					if tc.window >= 16_384 && room < want {
						t.Fatalf("%+v: output room %d below %d", tc, room, want)
					}
				} else if tc.window >= 2*(4096+want) && room < want {
					t.Fatalf("%+v: output room %d below %d", tc, room, want)
				}
				if threshold < tc.window/2 {
					t.Fatalf("%+v: threshold %d below half the window", tc, threshold)
				}
				if retain >= threshold {
					t.Fatalf("%+v: retain %d is not below threshold %d", tc, retain, threshold)
				}
				policy = true
			}
		}
		if !route || !policy {
			t.Fatalf("%+v: route=%t policy=%t", tc, route, policy)
		}
	}
}

// The previous patch reproduced the reported behavior: 200K window, ~45K cap.
func TestDSHCompactionOldReservationWasEarly(t *testing.T) {
	threshold, _ := resolveSDKThreshold(200_000, 45_000, 0.8, 65_536)
	if threshold > 90_000 {
		t.Fatalf("fixture no longer describes the reported 90K trigger: %d", threshold)
	}
}
