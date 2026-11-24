package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const version = "aibertai-acceptance-1"

type reportedTool struct {
	Category   string `json:"category"`
	Name       string `json:"name"`
	ResourceID string `json:"resource_id,omitempty"`
	Calls      int    `json:"calls"`
	Failed     int    `json:"failed"`
	DurationMS int64  `json:"duration_ms"`
}

type reportedPermissions struct {
	Asked   int `json:"asked"`
	Allowed int `json:"allowed"`
	Denied  int `json:"denied"`
}

type reportedInput struct {
	Kind          string `json:"kind"`
	Text          string `json:"text,omitempty"`
	TextTruncated bool   `json:"text_truncated,omitempty"`
}

func (h *relaySessionHost) collectTurnText(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	auth, err := h.live.authorization(ctx)
	if err != nil {
		return false
	}
	var config struct {
		Settings struct {
			Reporting struct {
				Enabled bool   `json:"enabled"`
				Level   string `json:"level"`
			} `json:"session_reporting"`
		} `json:"settings"`
	}
	err = h.live.client.json(ctx, http.MethodGet, "/api/v1/settings", auth.Access, "", nil, &config)
	return err == nil && config.Settings.Reporting.Enabled && config.Settings.Reporting.Level == "full"
}

func turnReportInput(prompt string) reportedInput {
	const maxRunes = 8192
	input := reportedInput{Kind: "prompt"}
	runes := []rune(strings.ReplaceAll(prompt, "\x00", ""))
	input.TextTruncated = len(runes) > maxRunes
	input.Text = string(runes[:min(len(runes), maxRunes)])
	return input
}

func (c *client) registerSession(ctx context.Context, auth authorization, machine, session, selected string, started time.Time) error {
	if selected == "" {
		selected = modelID
	}
	return c.json(ctx, "PUT", "/api/v1/sessions/"+session, auth.Access, machine, map[string]any{
		"state_seq": 1, "session_type": "conversation", "started_at": started, "model_id": selected,
		"client": map[string]string{"type": "desktop", "machine_id": machine, "version": version,
			"engine_version": version, "runtime_version": version},
	}, nil)
}

func (c *client) report(ctx context.Context, auth authorization, machine, session string, started, ended time.Time, stop string, usage modelUsage, tools ...reportedTool) error {
	return c.reportTurn(ctx, auth, machine, session, modelID, 1, 0, started, ended, stop, usage, turnReportInput(""), nil, tools...)
}

func (c *client) reportTurn(ctx context.Context, auth authorization, machine, session, selected string, index, after int, started, ended time.Time, stop string, usage modelUsage, input reportedInput, permissions *reportedPermissions, tools ...reportedTool) error {
	if selected == "" {
		selected = modelID
	}
	snapshot := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("[]")))
	turn := map[string]any{
		"turn_index": index, "input_seq": index, "started_at": started, "ended_at": ended,
		"stop_reason": stop, "model_id": selected, "resources_snapshot_id": snapshot,
		"client_version": version, "engine_version": version,
		"input": input,
		"usage": map[string]*int64{"input_tokens": usage.Input, "output_tokens": usage.Output},
	}
	if len(tools) != 0 {
		// Canvas and knowledge are desktop capabilities. Keep their local categories
		// for receipt verification, and report them as built-ins to the backend.
		facts := make([]reportedTool, len(tools))
		for i, tool := range tools {
			if tool.Category == "canvas" || tool.Category == "knowledge" {
				tool.Category = "builtin"
			}
			facts[i] = tool
		}
		turn["tools"] = facts
	}
	if permissions != nil {
		turn["permissions"] = permissions
	}
	payload, err := json.Marshal(map[string]any{
		"facts_version": 1, "turns": []any{turn},
		"snapshots": []any{map[string]any{"snapshot_id": snapshot, "items": []any{}}},
	})
	if err != nil {
		return err
	}
	// If a report committed before its reply was lost, the backend checks its
	// hash when the identical report is replayed at the returned cursor.
	for attempt := 0; attempt < 2; attempt++ {
		path := fmt.Sprintf("/api/v1/sessions/%s/turns?after_turn=%d", session, after)
		resp, err := c.request(ctx, "POST", path, auth.Access, machine, "", "application/json", bytes.NewReader(payload), http.StatusConflict)
		if err != nil {
			return err
		}
		var ack struct {
			Acked int `json:"acked_turn"`
			Error struct {
				Code       string
				References struct {
					Acked int `json:"acked_turn"`
				}
			}
		}
		err = json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&ack)
		_ = resp.Body.Close()
		if err != nil {
			return errors.New("invalid_session_report_response")
		}
		if resp.StatusCode != http.StatusConflict && ack.Acked == index {
			return nil
		}
		if resp.StatusCode != http.StatusConflict || ack.Error.Code != "cursor_mismatch" || ack.Error.References.Acked != index || after == index {
			break
		}
		after = index
	}
	return errors.New("session_report_not_acknowledged")
}
