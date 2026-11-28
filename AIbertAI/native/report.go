package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
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

type reportedFiles struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
}

// turnReportFacts are the optional facts the backend accepts beside usage and tools.
type turnReportFacts struct {
	Tokens        turnTokenFacts
	Permissions   *reportedPermissions
	ContextUsed   *int64
	ContextWindow *int64
	Files         *reportedFiles
	Compactions   int
	Resources     []turnResource
	SkillEvents   []map[string]any
}

// The backend rejects a turn with more tool facts than this.
const maxReportedTools = 500

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

// registerSession sends the first prompt's title only when the organization
// collects conversation text, the same rule as the reported prompt.
func (c *client) registerSession(ctx context.Context, auth authorization, machine, session, selected, title string, started time.Time) error {
	if selected == "" {
		selected = modelID
	}
	body := map[string]any{
		"state_seq": 1, "session_type": "conversation", "started_at": started, "model_id": selected,
		"client": map[string]string{"type": "desktop", "machine_id": machine, "version": version,
			"engine_version": version, "runtime_version": version},
	}
	if title != "" {
		body["title"], body["title_source"] = title, "prompt"
	}
	return c.json(ctx, "PUT", "/api/v1/sessions/"+session, auth.Access, machine, body, nil)
}

func (c *client) report(ctx context.Context, auth authorization, machine, session string, started, ended time.Time, stop string, usage modelUsage, tools ...reportedTool) error {
	return c.reportTurn(ctx, auth, machine, session, modelID, 1, 0, started, ended, stop, usage, turnReportInput(""), turnReportFacts{}, tools...)
}

// turnReportUsage follows model_calls: input_tokens include cache reads and
// writes, which DSH counts outside inputTokens. Team member usage is split out
// when it is a consistent part of the merged total.
func turnReportUsage(usage modelUsage, tokens turnTokenFacts) (map[string]*int64, map[string]*int64) {
	if tokens == (turnTokenFacts{}) || usage.Input == nil || usage.Output == nil {
		return map[string]*int64{"input_tokens": usage.Input, "output_tokens": usage.Output}, nil
	}
	input, output := *usage.Input, *usage.Output
	read, write := tokens.CacheRead, tokens.CacheWrite
	var subagent map[string]*int64
	if member := tokens.SubagentInput != 0 || tokens.SubagentOutput != 0 || tokens.SubagentCacheRead != 0 || tokens.SubagentCacheWrite != 0; member && input >= tokens.SubagentInput && output >= tokens.SubagentOutput {
		input, output = input-tokens.SubagentInput, output-tokens.SubagentOutput
		memberInput := tokens.SubagentInput + tokens.SubagentCacheRead + tokens.SubagentCacheWrite
		subagent = map[string]*int64{"input_tokens": &memberInput, "output_tokens": &tokens.SubagentOutput,
			"cache_read_input_tokens": &tokens.SubagentCacheRead, "cache_creation_input_tokens": &tokens.SubagentCacheWrite}
	} else {
		read, write = read+tokens.SubagentCacheRead, write+tokens.SubagentCacheWrite
	}
	input += read + write
	return map[string]*int64{"input_tokens": &input, "output_tokens": &output,
		"cache_read_input_tokens": &read, "cache_creation_input_tokens": &write}, subagent
}

// turnReportTools merges facts that share a category, name and resource, and
// truncates what still exceeds the backend limit.
func turnReportTools(tools []reportedTool) ([]reportedTool, bool) {
	facts := make([]reportedTool, 0, len(tools))
	seen := make(map[[3]string]int)
	for _, tool := range tools {
		// Canvas and knowledge are desktop capabilities. Keep their local categories
		// for receipt verification, and report them as built-ins to the backend.
		if tool.Category == "canvas" || tool.Category == "knowledge" {
			tool.Category = "builtin"
		}
		key := [3]string{tool.Category, tool.Name, tool.ResourceID}
		if i, ok := seen[key]; ok {
			facts[i].Calls += tool.Calls
			facts[i].Failed += tool.Failed
			facts[i].DurationMS += tool.DurationMS
			continue
		}
		seen[key] = len(facts)
		facts = append(facts, tool)
	}
	if len(facts) > maxReportedTools {
		return facts[:maxReportedTools], true
	}
	return facts, false
}

func (c *client) reportTurn(ctx context.Context, auth authorization, machine, session, selected string, index, after int, started, ended time.Time, stop string, usage modelUsage, input reportedInput, facts turnReportFacts, tools ...reportedTool) error {
	if selected == "" {
		selected = modelID
	}
	snapshot, items := resourceSnapshot(facts.Resources)
	turn := map[string]any{
		"turn_index": index, "input_seq": index, "started_at": started, "ended_at": ended,
		"stop_reason": stop, "model_id": selected, "resources_snapshot_id": snapshot,
		"client_version": version, "engine_version": version,
		"input": input,
	}
	reported, subagent := turnReportUsage(usage, facts.Tokens)
	turn["usage"] = reported
	if subagent != nil {
		turn["subagent_usage"] = subagent
	}
	if len(tools) != 0 {
		reportedTools, truncated := turnReportTools(tools)
		turn["tools"] = reportedTools
		if truncated {
			turn["truncated"] = true
		}
	}
	if facts.Permissions != nil {
		turn["permissions"] = facts.Permissions
	}
	if facts.ContextUsed != nil && facts.ContextWindow != nil {
		turn["context_used"], turn["context_window"] = facts.ContextUsed, facts.ContextWindow
	}
	if facts.Files != nil {
		turn["files_changed"] = facts.Files
	}
	if facts.Compactions > 0 {
		turn["compactions"] = facts.Compactions
	}
	if len(facts.SkillEvents) != 0 {
		turn["skill_events"] = facts.SkillEvents
	}
	payload, err := json.Marshal(map[string]any{
		"facts_version": 1, "turns": []any{turn},
		"snapshots": []any{map[string]any{"snapshot_id": snapshot, "items": items}},
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

// reportDevice updates the client inventory the admin console shows, including
// its last-active time. The identity handshake alone leaves those fields empty.
func (c *client) reportDevice(ctx context.Context, auth authorization, device relayDevice) error {
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}
	return c.json(ctx, http.MethodPut, "/api/v1/endpoints/"+device.machine, auth.Access, "", map[string]any{
		"device_name": device.name, "platform": platform, "os_version": osVersion(), "arch": runtime.GOARCH,
		"client_version": version, "protocol_version": 1, "client_type": "desktop", "client_name": "AIbertAI",
		"runtime_version": version, "engine_version": version,
	}, nil)
}

func osVersion() string {
	if runtime.GOOS == "darwin" {
		if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
			if v := strings.TrimSpace(string(out)); v != "" && len(v) <= 64 {
				return v
			}
		}
	}
	return runtime.GOOS
}

// touchDevice re-reports the device at most every five minutes while it is used.
func (l *acpLive) touchDevice(ctx context.Context) {
	l.mu.Lock()
	report := l.deviceReport
	due := report != nil && time.Since(l.deviceReportedAt) >= 5*time.Minute
	if due {
		l.deviceReportedAt = time.Now()
	}
	l.mu.Unlock()
	if !due {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := l.withTurnAuthorization(ctx, report); err != nil {
		fmt.Fprintf(os.Stderr, "device report failed: %v\n", err)
	}
}

// reportTitle is the first prompt's title when conversation text may be reported.
func (t acpTurn) reportTitle() string {
	if !t.ReportText {
		return ""
	}
	return promptTitle(t.Prompt)
}
