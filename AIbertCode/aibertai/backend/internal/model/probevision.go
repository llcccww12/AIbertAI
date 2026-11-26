package model

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

// A model can be marked as reading images while its upstream ignores them on
// the configured protocol (some servers drop input_image on /v1/responses but
// accept image_url on /v1/chat/completions). The text probe cannot notice that,
// so this probe sends a generated two-colour image and checks the answer.

type visionColor struct {
	Name  string
	Alias []string
	RGBA  color.RGBA
}

var visionColors = []visionColor{
	{"red", []string{"红"}, color.RGBA{220, 40, 40, 255}},
	{"green", []string{"绿"}, color.RGBA{40, 170, 60, 255}},
	{"blue", []string{"蓝"}, color.RGBA{40, 80, 220, 255}},
	{"yellow", []string{"黄"}, color.RGBA{240, 210, 30, 255}},
	{"black", []string{"黑"}, color.RGBA{10, 10, 10, 255}},
	{"white", []string{"白"}, color.RGBA{250, 250, 250, 255}},
}

const visionPrompt = "This image has a left half and a right half, each a single solid colour. " +
	"Reply with exactly: left=<colour>, right=<colour>, using one English word from " +
	"red, green, blue, yellow, black, white for each."

// visionImage draws a 256x128 PNG with the two colours side by side.
func visionImage(left, right visionColor) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 256, 128))
	for y := 0; y < 128; y++ {
		for x := 0; x < 256; x++ {
			if x < 128 {
				img.SetRGBA(x, y, left.RGBA)
			} else {
				img.SetRGBA(x, y, right.RGBA)
			}
		}
	}
	var out bytes.Buffer
	_ = png.Encode(&out, img)
	return out.Bytes()
}

// visionAnswerMatches checks that the text after "left" names the left colour
// and the text after "right" names the right colour.
func visionAnswerMatches(answer string, left, right visionColor) bool {
	text := strings.ToLower(answer)
	l, r := strings.Index(text, "left"), strings.Index(text, "right")
	if l < 0 || r < 0 {
		l, r = strings.Index(text, "左"), strings.Index(text, "右")
	}
	if l < 0 || r < 0 {
		return false
	}
	var leftPart, rightPart string
	if l < r {
		leftPart, rightPart = text[l:r], text[r:]
	} else {
		rightPart, leftPart = text[r:l], text[l:]
	}
	names := func(part string, want visionColor) bool {
		for _, other := range visionColors {
			found := strings.Contains(part, other.Name)
			for _, alias := range other.Alias {
				found = found || strings.Contains(part, alias)
			}
			if found != (other.Name == want.Name) {
				return false
			}
		}
		return true
	}
	return names(leftPart, left) && names(rightPart, right)
}

// visionRequestBody builds the request body with the image for the protocol.
func visionRequestBody(item Model, data []byte) (any, error) {
	encoded := base64.StdEncoding.EncodeToString(data)
	url := "data:image/png;base64," + encoded
	switch item.Protocol {
	case ProtocolOpenAIChat:
		return map[string]any{"model": item.ModelID, "stream": false, "messages": []any{map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "text", "text": visionPrompt},
			map[string]any{"type": "image_url", "image_url": map[string]string{"url": url}},
		}}}}, nil
	case ProtocolOpenAIResponses:
		return map[string]any{"model": item.ModelID, "stream": false, "input": []any{map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "input_text", "text": visionPrompt},
			map[string]any{"type": "input_image", "image_url": url},
		}}}}, nil
	case ProtocolAnthropic:
		return map[string]any{"model": item.ModelID, "stream": false, "max_tokens": 256, "messages": []any{map[string]any{"role": "user", "content": []any{
			map[string]any{"type": "image", "source": map[string]string{"type": "base64", "media_type": "image/png", "data": encoded}},
			map[string]any{"type": "text", "text": visionPrompt},
		}}}}, nil
	}
	return nil, errors.New("仅支持测试大语言模型协议")
}

func (s *Service) probeVision(w http.ResponseWriter, r *http.Request, item Model) {
	picks := rand.Perm(len(visionColors))
	left, right := visionColors[picks[0]], visionColors[picks[1]]
	upstream, err := probeRequest(item, visionPrompt, false)
	if err != nil {
		modelError(r.Context(), w, http.StatusBadRequest, err.Error())
		return
	}
	body, err := visionRequestBody(item, visionImage(left, right))
	if err == nil {
		upstream.Body, err = json.Marshal(body)
	}
	if err != nil {
		modelError(r.Context(), w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Minute)
	defer cancel()
	client, failure := s.probeClient(ctx, item)
	if failure != "" {
		modelError(r.Context(), w, http.StatusBadGateway, failure)
		return
	}
	defer client.CloseIdleConnections()
	started := time.Now()
	response, err := upstream.send(ctx, client)
	if err != nil {
		status, message := http.StatusBadGateway, "连接模型失败，请检查地址、协议和网络代理"
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			status, message = http.StatusGatewayTimeout, "模型响应超时，请重试"
		} else if errors.Is(ctx.Err(), context.Canceled) {
			return
		}
		modelError(r.Context(), w, status, message)
		return
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		modelError(r.Context(), w, http.StatusBadGateway, "读取模型响应失败")
		return
	}
	expected := fmt.Sprintf("left=%s, right=%s", left.Name, right.Name)
	if response.StatusCode != http.StatusOK {
		// Rejecting the image outright is a definite answer, not a transport fault.
		modelJSON(w, http.StatusOK, map[string]any{"content": fmt.Sprintf("模型返回 HTTP %d，未接受图片输入", response.StatusCode), "valid": false, "vision_passed": false, "vision_expected": expected, "total_ms": time.Since(started).Milliseconds()})
		return
	}
	content, valid := probeContent(item.Protocol, data)
	modelJSON(w, http.StatusOK, map[string]any{"content": content, "valid": valid, "vision_passed": valid && visionAnswerMatches(content, left, right), "vision_expected": expected, "total_ms": time.Since(started).Milliseconds()})
}
