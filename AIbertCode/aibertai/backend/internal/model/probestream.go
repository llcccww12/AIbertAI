package model

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

func probeStreamRejected(body io.Reader) bool {
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

func writeProbeStream(w http.ResponseWriter, protocol Protocol, body io.Reader, started time.Time) {
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("X-Accel-Buffering", "no")
	encoder := json.NewEncoder(w)
	controller := http.NewResponseController(w)
	send := func(event map[string]any) error {
		if err := encoder.Encode(event); err != nil {
			return err
		}
		return controller.Flush()
	}
	if send(map[string]any{"type": "start", "output_mode": "stream"}) != nil {
		return
	}

	var firstText *int64
	err := streamProbeText(protocol, body, func(text string) error {
		event := map[string]any{"type": "delta", "content": text}
		if firstText == nil && strings.TrimSpace(text) != "" {
			firstText = new(time.Since(started).Milliseconds())
			event["first_text_ms"] = *firstText
		}
		return send(event)
	})
	done := map[string]any{"type": "done", "valid": err == nil && firstText != nil, "total_ms": time.Since(started).Milliseconds()}
	switch {
	case err != nil:
		done["content"] = "模型事件流未正常完成"
	case firstText == nil:
		done["content"] = "模型响应缺少合法文本内容"
	}
	_ = send(done)
}

func streamProbeText(protocol Protocol, body io.Reader, emit func(string) error) error {
	limited := &io.LimitedReader{R: body, N: 8 << 20}
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var data []byte
	var finished, hasText bool
	invalid := errors.New("invalid model event stream")
	consume := func() error {
		if len(data) == 0 || finished {
			return nil
		}
		event := data
		data = nil
		if string(event) == "[DONE]" {
			finished = true
			return nil
		}
		var payload struct {
			Type     string
			Error    json.RawMessage
			Delta    json.RawMessage
			Response json.RawMessage
			Choices  []struct {
				Finish string `json:"finish_reason"`
				Delta  struct{ Content string }
			}
		}
		if json.Unmarshal(event, &payload) != nil || (len(payload.Error) != 0 && string(payload.Error) != "null") {
			return invalid
		}
		var text string
		switch protocol {
		case ProtocolOpenAIChat:
			if len(payload.Choices) != 0 {
				text = payload.Choices[0].Delta.Content
				finished = payload.Choices[0].Finish != ""
			}
		case ProtocolOpenAIResponses:
			switch payload.Type {
			case "response.output_text.delta":
				if json.Unmarshal(payload.Delta, &text) != nil {
					return invalid
				}
			case "response.completed", "response.incomplete":
				finished = true
				if !hasText {
					if content, valid := probeContent(protocol, payload.Response); valid {
						text = content
					}
				}
			case "response.failed", "error":
				return invalid
			}
		case ProtocolAnthropic:
			switch payload.Type {
			case "content_block_delta":
				var delta struct{ Type, Text string }
				if json.Unmarshal(payload.Delta, &delta) != nil {
					return invalid
				}
				if delta.Type == "text_delta" {
					text = delta.Text
				}
			case "message_stop":
				finished = true
			case "error":
				return invalid
			}
		}
		if text != "" {
			hasText = hasText || strings.TrimSpace(text) != ""
			return emit(text)
		}
		return nil
	}
	for scanner.Scan() {
		if scanner.Text() == "" {
			if err := consume(); err != nil {
				return err
			}
			if finished {
				return nil
			}
		} else if value, ok := strings.CutPrefix(scanner.Text(), "data:"); ok {
			if len(data)+len(value)+1 > 1<<20 {
				return invalid
			}
			if len(data) != 0 {
				data = append(data, '\n')
			}
			data = append(data, strings.TrimPrefix(value, " ")...)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if limited.N == 0 {
		return invalid
	}
	if err := consume(); err != nil {
		return err
	}
	if !finished {
		return invalid
	}
	return nil
}
