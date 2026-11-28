package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	imagepkg "image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Read the preceding real canvas tool result from the synthetic model input.
func dshFixtureCanvasValue(value any, key string) string {
	switch item := value.(type) {
	case string:
		var decoded any
		if json.Unmarshal([]byte(item), &decoded) == nil {
			return dshFixtureCanvasValue(decoded, key)
		}
	case []any:
		for _, child := range item {
			if found := dshFixtureCanvasValue(child, key); found != "" {
				return found
			}
		}
	case map[string]any:
		if key == "node_id" {
			if node, ok := item["node"].(map[string]any); ok {
				if id, ok := node["id"].(string); ok && id != "" {
					return id
				}
			}
		} else if found, ok := item[key].(string); ok && found != "" {
			return found
		}
		for _, child := range item {
			if found := dshFixtureCanvasValue(child, key); found != "" {
				return found
			}
		}
	}
	return ""
}

// Local browser fixture only. It serves a deliberately synthetic model and
// an isolated workspace, while using the real platform and patched DSH SDK.
func TestDSHWebFixture(t *testing.T) {
	if os.Getenv("DSH_WEB_SERVE") != "1" {
		t.Skip("interactive fixture disabled")
	}
	node, entry := os.Getenv("DSH_TEST_NODE"), os.Getenv("DSH_PATCHED_TEST_ENTRY")
	if node == "" || entry == "" {
		t.Fatal("DSH_TEST_NODE and DSH_PATCHED_TEST_ENTRY required")
	}
	const modelID = "22222222-2222-4222-8222-222222222222"
	const alternateModel = "23232323-2323-4232-8232-232323232323"
	conversationReview := os.Getenv("DSH_WEB_CONVERSATION_REVIEW") == "1"
	switchModel := os.Getenv("DSH_WEB_MODEL_SWITCH") == "1"
	var stepCancelled, feedbackReceived atomic.Bool
	const keyID = "33333333-3333-4333-8333-333333333333"
	root := t.TempDir()
	goalMode := os.Getenv("DSH_WEB_GOAL") == "1"
	goalApproval := goalMode && os.Getenv("DSH_WEB_GOAL_APPROVAL") == "1"
	legacyMode := os.Getenv("DSH_WEB_LEGACY") == "1"
	attachmentMode := os.Getenv("DSH_WEB_ATTACHMENTS") == "1"
	visionMode := os.Getenv("DSH_WEB_VISION") == "1"
	imageHistoryMode := os.Getenv("DSH_WEB_IMAGE_HISTORY") == "1"
	expertMode := os.Getenv("DSH_WEB_EXPERT") == "1"
	autoCompact := os.Getenv("DSH_WEB_NATIVE_COMPACT") == "auto"
	teamSwitch := os.Getenv("DSH_WEB_TEAM_SWITCH") == "1"
	failMember := os.Getenv("DSH_WEB_MEMBERS") == "error"
	beforeMemberHeaders := os.Getenv("DSH_WEB_MEMBERS") == "stop-before-headers"
	withholdReceipt := os.Getenv("DSH_WEB_MEMBERS") == "stop-no-receipt"
	holdMember := failMember || beforeMemberHeaders || withholdReceipt || os.Getenv("DSH_WEB_MEMBERS") == "stop"
	stopOneMember := os.Getenv("DSH_WEB_MEMBERS") == "chat-stop"
	stopColdMember := os.Getenv("DSH_WEB_MEMBERS") == "chat-cold-stop"
	limitOneMember := os.Getenv("DSH_WEB_MEMBERS") == "chat-limit"
	memberChat := stopOneMember || stopColdMember || limitOneMember || os.Getenv("DSH_WEB_MEMBERS") == "chat"
	memberMode := memberChat || holdMember || os.Getenv("DSH_WEB_MEMBERS") == "1"
	uxMode := os.Getenv("DSH_WEB_UX") == "1"
	goalHold := goalMode && os.Getenv("DSH_WEB_GOAL_HOLD") == "1"
	todoMode := goalMode && os.Getenv("DSH_WEB_DYNAMIC") == "todos"
	releaseGoal := make(chan struct{})
	var goalHeld, goalReleased atomic.Bool
	t.Cleanup(func() {
		if goalReleased.CompareAndSwap(false, true) {
			close(releaseGoal)
		}
	})
	if os.Getenv("DSH_WEB_TRACE") == "1" {
		quotedEntry, _ := json.Marshal(entry)
		wrapper := filepath.Join(root, "trace-sdk.cjs")
		source := `const {spawn}=require('node:child_process');
const child=spawn(process.execPath,[` + string(quotedEntry) + `,...process.argv.slice(2)],{stdio:['pipe','pipe','inherit']});
process.stdin.pipe(child.stdin); child.stdout.pipe(process.stdout);
let buffer=''; child.stdout.on('data', data=>{buffer+=data; let n; while((n=buffer.indexOf('\n'))>=0){const line=buffer.slice(0,n);buffer=buffer.slice(n+1);try{const frame=JSON.parse(line);if(frame.method==='aibertai/model-settled')console.error('FIXTURE_SETTLEMENT',JSON.stringify(frame.params));}catch{}}});
child.on('exit',code=>process.exit(code??1));process.on('SIGTERM',()=>child.kill('SIGTERM'));
`
		if err := os.WriteFile(wrapper, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		entry = wrapper
	}
	handle, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	fixtureLock, err := lockDesktopSessionStore(handle)
	if err != nil {
		t.Fatal(err)
	}
	defer fixtureLock.Close()
	workspaces, err := openDesktopWorkspaces(handle, filepath.Join(root, "workspaces"), "")
	if err != nil {
		t.Fatal(err)
	}
	c, err := newBackendClient("https://gateway.test", false)
	if err != nil {
		t.Fatal(err)
	}
	const connectorID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	const credentialID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	var revoked atomic.Bool
	var connectorCalls atomic.Int64
	var calls atomic.Int64
	var memberRootCalls, designerCalls atomic.Int32
	var memberModelHeld, memberModelCancelled, memberOutputLimited atomic.Bool
	modelFailures := os.Getenv("DSH_WEB_GATEWAY_FAILURE") == "1"
	var reportFailures atomic.Int32
	var failedPath, failedHash atomic.Value
	var deferredModelReceipt, deferredStreamReceipt atomic.Bool
	var holdStreamReceipt, streamReceiptWaiting atomic.Bool
	streamReceiptRelease := make(chan struct{})
	var releaseStreamReceipt sync.Once
	var memberReceiptPath, memberReceiptHash atomic.Value
	var memberReceiptRequested, memberReceiptVerified atomic.Bool
	const memberTransaction = "77777777-7777-4777-8777-777777777777"
	var compactCalls atomic.Int64
	var disconnectServer *httptest.Server
	if os.Getenv("DSH_WEB_CONNECTOR_DISCONNECT") == "1" {
		disconnectServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			connectorCalls.Add(1) // The simulated side effect occurs before the reply is interrupted.
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				return
			}
			defer conn.Close()
			_, _ = io.WriteString(conn, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 100\r\n\r\n{\"jsonrpc\":\"2.0\"")
		}))
		defer disconnectServer.Close()
	}
	var live *acpLive
	expertEditor := os.Getenv("DSH_WEB_EXPERT_EDITOR") == "1"
	var expertMu sync.Mutex
	var personalExpert map[string]any
	expertRevision := 0
	const personalExpertID = "99999999-9999-4999-8999-999999999999"
	c.http.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		switch {
		case r.URL.Path == "/api/v1/connectors" && os.Getenv("DSH_WEB_DYNAMIC") == "connector":
			status := "authorized"
			if revoked.Load() {
				status = "revoked"
			}
			fmt.Fprintf(w, `{"connectors":[{"id":%q,"name":"模拟连接器","authorization_mode":"independent","credentials":[{"id":%q,"authorization_status":%q}]}],"total_count":1}`, connectorID, credentialID, status)
		case r.URL.Path == "/api/v1/connectors/"+connectorID+"/credentials/"+credentialID+"/tools" && os.Getenv("DSH_WEB_DYNAMIC") == "connector":
			fmt.Fprint(w, `{"items":[{"name":"perform","description":"模拟写入","enabled":true,"input_schema":{"type":"object","properties":{"message":{"type":"string"}},"required":["message"]}}]}`)
		case r.URL.Path == "/mcp/connectors/"+connectorID+"/credentials/"+credentialID && os.Getenv("DSH_WEB_DYNAMIC") == "connector":
			if disconnectServer != nil {
				target, _ := url.Parse(disconnectServer.URL)
				cloned := r.Clone(r.Context())
				cloned.URL = target
				cloned.Host = target.Host
				return http.DefaultTransport.RoundTrip(cloned)
			}
			connectorCalls.Add(1)
			if os.Getenv("DSH_WEB_CONNECTOR_UNKNOWN") == "1" {
				w.WriteHeader(503)
				break
			}
			fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"模拟连接器调用成功"}]}}`)
		case r.URL.Path == "/api/v1/models" && switchModel:
			fmt.Fprintf(w, `{"total_count":2,"models":[{"id":%q,"model":"fixture","display_name":"验收模型","kind":"text","protocol":"openai_chat_completions","context_window_tokens":100000,"max_output_tokens":4096,"ready":true},{"id":%q,"model":"fixture-alternate","display_name":"备用验收模型","kind":"text","protocol":"openai_chat_completions","context_window_tokens":100000,"max_output_tokens":4096,"ready":true}]}`, modelID, alternateModel)
		case r.URL.Path == "/api/v1/models":
			fmt.Fprintf(w, `{"total_count":1,"models":[{"id":%q,"model":"fixture","display_name":"验收模型","kind":"text","protocol":"openai_chat_completions","context_window_tokens":100000,"max_output_tokens":4096,"supports_vision":%t,"ready":true}]}`, modelID, visionMode)
		case expertEditor && r.URL.Path == "/api/v1/skills":
			fmt.Fprint(w, `{"total_count":1,"skills":[{"id":"88888888-8888-4888-8888-888888888888","name":"fixture-drawing","description":"Drawing skill","ownership_type":"system"}]}`)
		case expertEditor && (r.URL.Path == "/api/v1/experts" || r.URL.Path == "/api/v1/experts/"+personalExpertID):
			expertMu.Lock()
			defer expertMu.Unlock()
			if r.Method == http.MethodGet && r.URL.Path == "/api/v1/experts" {
				experts := []map[string]any{}
				if personalExpert != nil {
					experts = append(experts, personalExpert)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"total_count": len(experts), "experts": experts})
				break
			}
			if r.Method == http.MethodPost || r.Method == http.MethodPut {
				if r.Method == http.MethodPut && r.Header.Get("If-Match") != fmt.Sprintf(`"%d"`, expertRevision) {
					w.WriteHeader(http.StatusPreconditionFailed)
					break
				}
				if personalExpert == nil {
					personalExpert = map[string]any{"id": personalExpertID, "ownership_type": "user", "user": map[string]string{"id": "demo", "name": "Demo User"}, "available": true}
				}
				var patch map[string]any
				if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					break
				}
				for k, v := range patch {
					personalExpert[k] = v
				}
				expertRevision++
				personalExpert["version"] = expertRevision
			}
			w.Header().Set("ETag", fmt.Sprintf(`"%d"`, expertRevision))
			_ = json.NewEncoder(w).Encode(personalExpert)
		case r.URL.Path == "/api/v1/experts" && teamSwitch:
			fmt.Fprint(w, `{"total_count":1,"experts":[{"id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","name":"受限测试专家","description":"没有共享团队工具"}]}`)
		case r.URL.Path == "/api/v1/experts/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/manifest" && teamSwitch:
			fmt.Fprint(w, `{"expert_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","name":"受限测试专家","available":true,"dsh_plugins":["tool-fs"]}`)
		case r.URL.Path == "/api/v1/experts" && expertMode:
			fmt.Fprint(w, `{"total_count":2,"experts":[{"id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","name":"模拟专家甲","description":"本地验收专家"},{"id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","name":"模拟专家乙","description":"本地验收专家"}]}`)
		case r.URL.Path == "/api/v1/experts/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/manifest" && expertMode:
			fmt.Fprint(w, `{"expert_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","name":"模拟专家甲","available":true,"prompt":"FIXTURE_EXPERT_A_SYSTEM {{literal}}","rules":[{"content":"FIXTURE_EXPERT_A_RULE"}]}`)
		case r.URL.Path == "/api/v1/experts/bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb/manifest" && expertMode:
			fmt.Fprint(w, `{"expert_id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","name":"模拟专家乙","available":true,"prompt":"FIXTURE_EXPERT_B_SYSTEM {{literal}}","rules":[{"content":"FIXTURE_EXPERT_B_RULE"}]}`)
		case r.URL.Path == "/api/v1/kb" && os.Getenv("DSH_WEB_KNOWLEDGE") == "1":
			fmt.Fprint(w, `{"knowledge_bases":[{"id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","name":"模拟产品文档","description":"本地验收资料","enabled":true,"learned_count":2,"document_count":2},{"id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","name":"模拟团队规范","description":"本地验收资料","enabled":true,"learned_count":1,"document_count":1}],"total_count":2,"page":1,"page_size":50}`)
		case r.URL.Path == "/api/v1/kb/search" && os.Getenv("DSH_WEB_DYNAMIC") == "knowledge":
			var request struct {
				Query string   `json:"query"`
				IDs   []string `json:"kb_ids"`
			}
			if json.NewDecoder(r.Body).Decode(&request) != nil || request.Query != "产品规范" || len(request.IDs) != 1 || request.IDs[0] != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" {
				w.WriteHeader(http.StatusBadRequest)
				break
			}
			fmt.Fprint(w, `{"query":"产品规范","retrieval_mode":"lexical","channels_used":["lexical"],"fusion":{},"degraded":false,"warnings":[],"items":[{"kb_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","document_id":"cccccccc-cccc-4ccc-8ccc-cccccccccccc","chunk_id":"dddddddd-dddd-4ddd-8ddd-dddddddddddd","citation":{"heading_path":"产品 / 指南","page_no":2,"source_url":null},"content":"模拟知识库参考内容。"}]}`)
		case r.URL.Path == "/api/v1/attachments/extract" && attachmentMode:
			data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			if err != nil || len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
				w.WriteHeader(http.StatusBadRequest)
				break
			}
			fmt.Println("DSH_WEB_ATTACHMENT_IMAGE_EXTRACTED=1 (synthetic)")
			fmt.Fprint(w, `{"text":"模拟图片描述"}`)
		case r.URL.Path == "/api/v1/api-keys" && r.Method == "POST":
			fmt.Fprintf(w, `{"id":%q,"api_key":"fixture"}`, keyID)
		case r.URL.Path == "/api/v1/api-keys/"+keyID && r.Method == "DELETE":
		case modelFailures && r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/receipt"):
			var input map[string]string
			_ = json.NewDecoder(r.Body).Decode(&input)
			if r.URL.Path != failedPath.Load() || input["request_hash"] != failedHash.Load() || input["model_id"] != modelID || input["transaction_id"] != memberTransaction {
				w.WriteHeader(http.StatusNotFound)
				break
			}
			if calls.Load() == 9 && holdStreamReceipt.Load() {
				streamReceiptWaiting.Store(true)
				select {
				case <-streamReceiptRelease:
				case <-r.Context().Done():
					return nil, r.Context().Err()
				}
			}
			if calls.Load() == 9 && deferredStreamReceipt.Load() {
				fmt.Fprintf(w, `{"recorded":true,"status":"released","result":"failed","error_code":"upstream_error_event","transaction_id":%q}`, memberTransaction)
				break
			}
			if calls.Load() == 9 || calls.Load() >= 7 && !deferredModelReceipt.Load() {
				fmt.Fprintf(w, `{"recorded":true,"status":"unknown","result":"failed","error_code":"upstream_http_error","transaction_id":%q}`, memberTransaction)
				break
			}
			fmt.Fprintf(w, `{"recorded":true,"status":"released","result":"failed","error_code":"upstream_http_error","transaction_id":%q}`, memberTransaction)
		case (holdMember || stopOneMember || stopColdMember) && r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/receipt"):
			memberReceiptRequested.Store(true)
			var input map[string]string
			_ = json.NewDecoder(r.Body).Decode(&input)
			if r.URL.Path != memberReceiptPath.Load() || input["request_hash"] != memberReceiptHash.Load() ||
				input["model_id"] != modelID || !memberModelCancelled.Load() || (input["transaction_id"] != "" && input["transaction_id"] != memberTransaction) {
				w.WriteHeader(http.StatusNotFound)
				break
			}
			if withholdReceipt {
				fmt.Fprint(w, `{"recorded":false}`)
				break
			}
			memberReceiptVerified.Store(true)
			fmt.Fprintf(w, `{"recorded":true,"status":"released","result":"cancelled","transaction_id":%q}`, memberTransaction)
		case strings.HasPrefix(r.URL.Path, "/api/v1/sessions/") && strings.HasSuffix(r.URL.Path, "/receipt") && os.Getenv("DSH_WEB_CANCEL_RECOVERY") == "1":
			fmt.Fprint(w, `{"recorded":true,"transaction_id":"44444444-4444-4444-8444-444444444444"}`)
		case strings.HasPrefix(r.URL.Path, "/api/v1/sessions/") && strings.HasSuffix(r.URL.Path, "/turns"):
			if modelFailures {
				reportFailures.Add(1)
				w.WriteHeader(http.StatusServiceUnavailable)
				break
			}
			var body struct {
				Turns []struct {
					Index int `json:"turn_index"`
				} `json:"turns"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if len(body.Turns) != 1 {
				w.WriteHeader(400)
			} else {
				fmt.Fprintf(w, `{"acked_turn":%d}`, body.Turns[0].Index)
			}
		case strings.HasPrefix(r.URL.Path, "/api/v1/sessions/"):
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/api/v1/settings":
			fmt.Fprint(w, `{"settings":{"session_reporting":{"enabled":true,"level":"full"}}}`)
		case r.URL.Path == "/v1/chat/completions":
			select {
			case <-r.Context().Done():
				return nil, r.Context().Err()
			case <-time.After(700 * time.Millisecond):
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			var request map[string]any
			_ = json.Unmarshal(body, &request)
			n := calls.Add(1)
			if conversationReview {
				emit := func(out io.Writer, delta map[string]any, finish string) {
					frame, _ := json.Marshal(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}})
					_, _ = fmt.Fprintf(out, "data: %s\n\n", frame)
					frame, _ = json.Marshal(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": finish}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 2, "total_tokens": 5}})
					_, _ = fmt.Fprintf(out, "data: %s\n\ndata: [DONE]\n\n", frame)
				}
				tool := func(name string, args map[string]any) map[string]any {
					arguments, _ := json.Marshal(args)
					return map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("fixture-call-%d", n), "type": "function", "function": map[string]any{"name": name, "arguments": string(arguments)}}}}
				}
				w.Header().Set("Content-Type", "text/event-stream")
				switch n {
				case 1:
					reader, writer := io.Pipe()
					go func() {
						defer writer.Close()
						_, _ = io.WriteString(writer, `data: {"id":"fixture","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"正在准备首轮校验。"},"finish_reason":null}]}`+"\n\n")
						select {
						case <-releaseGoal:
						case <-r.Context().Done():
							stepCancelled.Store(true)
							return
						}
						emit(writer, tool("bash", map[string]any{"description": "后台校验示例", "command": "printf 'fixture-check-ready\\n'; exec sleep 45", "run_in_background": true}), "tool_calls")
					}()
					return &http.Response{StatusCode: http.StatusOK, Header: w.Result().Header, Body: reader}, nil
				case 2:
					answer := "本轮没有收到补充消息。"
					if bytes.Contains(body, []byte("补充验收条件")) {
						answer = "本轮已读取补充条件，后台校验继续运行。"
					}
					emit(w, map[string]any{"content": answer}, "stop")
				case 3, 4:
					plan := "# 初版校验计划\n\n- 核对第一节"
					if n == 4 {
						feedbackReceived.Store(bytes.Contains(body, []byte("覆盖第二节")))
						plan = "# 修订校验计划\n\n- 核对第一节和第二节"
					}
					emit(w, tool("exit_plan_mode", map[string]any{"plan": plan}), "tool_calls")
				case 5:
					emit(w, map[string]any{"content": "已按批准的计划执行。"}, "stop")
				case 7:
					w.Header().Set("X-Billing-Transaction-ID", memberTransaction)
					reader, writer := io.Pipe()
					go func() {
						defer writer.Close()
						_, _ = io.WriteString(writer, `data: {"id":"fixture","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"正在拟定待停止的计划。"},"finish_reason":null}]}`+"\n\n")
						<-r.Context().Done()
					}()
					// No receipt route for this transaction.
					return &http.Response{StatusCode: http.StatusOK, Header: w.Result().Header, Body: reader}, nil
				case 8:
					emit(w, map[string]any{"content": "停止后仍可在原会话继续。"}, "stop")
				default:
					emit(w, map[string]any{"content": "历史暂不可用时仍能回复。"}, "stop")
				}
				return w.Result(), nil
			}
			if modelFailures {
				invocation := r.Header.Get("Idempotency-Key")
				switch n {
				case 1:
					w.Header().Set(dshNotDispatchedHeader, invocation)
					w.WriteHeader(http.StatusTooManyRequests)
				case 3, 7:
					session := r.Header.Get("X-MAI-Session-ID")
					hash, err := dshCancelRequestHash(dshLocalRecord{SessionID: session, ModelID: modelID, Body: body, Protocol: "openai_chat_completions"})
					if err != nil {
						return nil, err
					}
					failedPath.Store("/api/v1/sessions/" + session + "/model-invocations/" + invocation + "/receipt")
					failedHash.Store(hash)
					w.Header().Set("X-Billing-Transaction-ID", memberTransaction)
					if n == 3 {
						w.WriteHeader(http.StatusTooManyRequests)
					} else {
						w.WriteHeader(http.StatusServiceUnavailable)
					}
				case 9:
					session := r.Header.Get("X-MAI-Session-ID")
					hash, err := dshCancelRequestHash(dshLocalRecord{SessionID: session, ModelID: modelID, Body: body, Protocol: "openai_chat_completions"})
					if err != nil {
						return nil, err
					}
					failedPath.Store("/api/v1/sessions/" + session + "/model-invocations/" + invocation + "/receipt")
					failedHash.Store(hash)
					w.Header().Set("X-Billing-Transaction-ID", memberTransaction)
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprintf(w, `data: {"id":"incomplete","model":%q,"choices":[{"index":0,"delta":{"role":"assistant","content":"断开前已收到的文字"},"finish_reason":null}]}`+"\n\n", modelID)
					return w.Result(), nil
				default:
					answers := map[int64]string{2: "限流后已继续处理。", 4: "已在原会话继续。", 5: "已保留的半句", 6: "已接着写完。", 8: "已从确认的失败处继续。", 10: "已补完断流的回复。"}
					finish := "stop"
					if n == 5 {
						finish = "length"
					}
					w.Header().Set("Content-Type", "text/event-stream")
					fmt.Fprintf(w, `data: {"id":"fixture","object":"chat.completion.chunk","model":%q,"choices":[{"index":0,"delta":{"role":"assistant","content":%q},"finish_reason":null}]}`+"\n\n", modelID, answers[int64(n)])
					fmt.Fprintf(w, `data: {"id":"fixture","object":"chat.completion.chunk","model":%q,"choices":[{"index":0,"delta":{},"finish_reason":%q}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`+"\n\ndata: [DONE]\n\n", modelID, finish)
					return w.Result(), nil
				}
				fmt.Fprint(w, `{"error":{"message":"private test diagnostic"}}`)
				return w.Result(), nil
			}
			if expertMode {
				var system, lastUser string
				var userHistory strings.Builder
				messages, _ := request["messages"].([]any)
				for _, item := range messages {
					message, _ := item.(map[string]any)
					value, _ := json.Marshal(message["content"])
					if message["role"] == "system" {
						system += string(value)
					}
					if message["role"] == "user" && (strings.Contains(string(value), "验收甲") || strings.Contains(string(value), "验收乙") || strings.Contains(string(value), "验收清除")) {
						lastUser = string(value)
						userHistory.Write(value)
					}
				}
				if strings.Contains(userHistory.String(), "第二会话") && (strings.Contains(userHistory.String(), "第一轮") || strings.Contains(userHistory.String(), "切换后") || strings.Contains(userHistory.String(), "验收清除")) {
					fmt.Println("DSH_WEB_EXPERT_HISTORY_CROSSTALK")
					w.WriteHeader(http.StatusBadRequest)
					return w.Result(), nil
				}
				want := ""
				if strings.Contains(lastUser, "验收甲") {
					want = "A"
				}
				if strings.Contains(lastUser, "验收乙") {
					want = "B"
				}
				if !strings.Contains(lastUser, "验收甲") && !strings.Contains(lastUser, "验收乙") && !strings.Contains(lastUser, "验收清除") {
					fmt.Println("DSH_WEB_EXPERT_UNKNOWN_PROMPT")
					w.WriteHeader(http.StatusBadRequest)
					return w.Result(), nil
				}
				for _, label := range []string{"A", "B"} {
					present := strings.Contains(system, "FIXTURE_EXPERT_"+label+"_SYSTEM {{literal}}") && strings.Contains(system, "FIXTURE_EXPERT_"+label+"_RULE")
					if present != (want == label) || want != label && strings.Contains(system, "FIXTURE_EXPERT_"+label) {
						fmt.Printf("DSH_WEB_EXPERT_MISMATCH want=%s label=%s\n", want, label)
						w.WriteHeader(http.StatusBadRequest)
						return w.Result(), nil
					}
				}
				fmt.Printf("DSH_WEB_EXPERT_SYSTEM_OK=%s request=%d\n", want, n)
			}
			if attachmentMode {
				wire, _ := json.Marshal(request)
				wants := []string{"notes.txt", "附件文本验收内容", "report.docx", "办公文档验收内容", "photo.png", "模拟图片描述"}
				if visionMode {
					wants = []string{"notes.txt", "附件文本验收内容", "report.docx", "办公文档验收内容", "data:image/png;base64,"}
				}
				for _, want := range wants {
					if !strings.Contains(string(wire), want) {
						fmt.Printf("DSH_WEB_ATTACHMENT_MODEL_MISSING=%s\n", want)
						w.WriteHeader(http.StatusBadRequest)
						return w.Result(), nil
					}
				}
				fmt.Printf("DSH_WEB_ATTACHMENT_MODEL_OK=%d\n", n)
			}
			if goalMode && n == 1 && os.Getenv("DSH_WEB_GOAL_INFLIGHT") == "1" {
				fmt.Println("DSH_WEB_GOAL_INFLIGHT=1")
				select {
				case <-r.Context().Done():
					return nil, r.Context().Err()
				case <-time.After(10 * time.Second):
				}
			}
			stage := 0
			toolFailed := false
			uxScenario := ""
			if messages, ok := request["messages"].([]any); ok {
				for _, item := range messages {
					message, _ := item.(map[string]any)
					if message["role"] == "user" {
						stage = 0
						toolFailed = false
						if uxMode {
							value, _ := json.Marshal(message["content"])
							for _, marker := range []string{"验收长文", "验收缺失", "验收停止", "验收继续"} {
								if strings.Contains(string(value), marker) {
									uxScenario = marker
								}
							}
						}
					}
					if message["role"] == "tool" {
						value, _ := json.Marshal(message["content"])
						result := strings.ToLower(string(value))
						toolFailed = toolFailed || strings.Contains(result, "error") || strings.Contains(result, "denied") || strings.Contains(result, "拒绝")
					}
					if message["role"] == "assistant" {
						if toolCalls, ok := message["tool_calls"].([]any); ok {
							stage += len(toolCalls)
						}
					}
				}
			}
			w.Header().Set("Content-Type", "text/event-stream")
			delta := map[string]any{"role": "assistant", "content": "已完成。工作区中的说明已根据读取的内容更新。"}
			finish := "stop"
			if attachmentMode {
				delta["content"] = "模拟模型已收到附件内容。"
				stage = -1
			}
			if expertMode {
				delta["content"] = "模拟专家验收通过。"
				stage = -1
			}
			if toolFailed {
				delta["content"] = "部分步骤未完成，请查看执行结果。"
			}
			if os.Getenv("DSH_WEB_PROMPT_RECOVERY") == "1" {
				delta = map[string]any{"role": "assistant", "content": "这是已经提交的模拟回复。"}
				stage = -1
			}
			if legacyMode {
				data, _ := json.Marshal(request)
				if strings.Contains(string(data), "旧引擎继续") {
					fmt.Println("DSH_WEB_LEGACY_MODEL=1")
					delta = map[string]any{"role": "assistant", "content": "旧引擎继续处理完成。"}
					stage = -1
				}
			}
			if os.Getenv("DSH_WEB_DYNAMIC") == "knowledge" {
				if n == 1 {
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "knowledge-1", "type": "function", "function": map[string]string{"name": "knowledge_search", "arguments": `{"query":"产品规范"}`}}}}
					finish = "tool_calls"
				} else {
					delta = map[string]any{"role": "assistant", "content": "模拟知识库检索结束。"}
				}
				stage = -1
			}
			if os.Getenv("DSH_WEB_DYNAMIC") == "connector" {
				if n <= 2 {
					alias := (connectorTool{ConnectorID: connectorID, Name: "perform"}).alias()
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("connector-%d", n), "type": "function", "function": map[string]string{"name": alias, "arguments": `{"message":"模拟内容"}`}}}}
					finish = "tool_calls"
				} else {
					text := "模拟连接器处理完成。"
					if toolFailed {
						text = "部分连接器调用未执行，请查看审批结果。"
					}
					delta = map[string]any{"role": "assistant", "content": text}
				}
				stage = -1
			}
			if os.Getenv("DSH_WEB_DYNAMIC") == "canvas_write" {
				if n <= 2 || n == 4 {
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("canvas-write-%d", n), "type": "function", "function": map[string]string{"name": "canvas_create_text", "arguments": fmt.Sprintf(`{"content":"工作台画布验收 %d","x":%d,"y":32}`, n, n*32)}}}}
					finish = "tool_calls"
				} else {
					delta = map[string]any{"role": "assistant", "content": "画布写入已完成。"}
					if toolFailed {
						delta["content"] = "画布操作未完成，请查看审批及执行状态。"
					}
				}
				stage = -1
			}
			if os.Getenv("DSH_WEB_DYNAMIC") == "pptx" {
				name := ""
				args := map[string]any{}
				switch n {
				case 1:
					name = designCreateHTMLTool
					args = map[string]any{"title": "模拟幻灯片", "slide_index": 1}
				case 2:
					name = "write"
					args = map[string]any{"file_path": dshFixtureCanvasValue(request, "artifact_path"), "content": "<html><body>模拟幻灯片</body></html>"}
				case 3:
					name = designCommitHTMLTool
					args = map[string]any{"node_id": dshFixtureCanvasValue(request, "node_id")}
				case 4:
					name = designPreparePPTXTool
				}
				if name != "" && !toolFailed {
					encoded, _ := json.Marshal(args)
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("pptx-%d", n), "type": "function", "function": map[string]string{"name": name, "arguments": string(encoded)}}}}
					finish = "tool_calls"
				} else {
					delta = map[string]any{"role": "assistant", "content": "模拟幻灯片已准备。"}
					if toolFailed {
						delta["content"] = "模拟幻灯片未完成，请查看执行步骤。"
					}
				}
				stage = -1
			}
			if os.Getenv("DSH_WEB_DYNAMIC") == "canvas" {
				if n == 1 {
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "canvas-1", "type": "function", "function": map[string]string{"name": "canvas_get_selection", "arguments": `{}`}}}}
					finish = "tool_calls"
				} else {
					delta = map[string]any{"role": "assistant", "content": "模拟画布读取结束。"}
					if toolFailed {
						delta["content"] = "模拟画布读取未完成。"
					}
				}
				stage = -1
			}
			if os.Getenv("DSH_WEB_DYNAMIC") == "canvas_catalog" {
				if n == 1 {
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "catalog-1", "type": "function", "function": map[string]string{"name": "design_catalog_search", "arguments": `{"query":"海报"}`}}}}
					finish = "tool_calls"
				} else {
					delta = map[string]any{"role": "assistant", "content": "已找到模拟设计模板。"}
					modelInput, _ := json.Marshal(request)
					if toolFailed || !strings.Contains(string(modelInput), "模拟设计模板") {
						delta["content"] = "模拟设计目录查询未完成。"
					}
				}
				stage = -1
			}
			if goalMode {
				step := n
				if goalApproval && step > 1 {
					step--
				}
				switch step {
				case 1:
					name, arguments := "read", `{"file_path":"brief.txt"}`
					if goalApproval {
						name, arguments = "write", `{"file_path":"goal-result.txt","content":"模拟目标输出。\n"}`
					}
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("goal-file-%d", n), "type": "function", "function": map[string]string{"name": name, "arguments": arguments}}}}
					finish = "tool_calls"
				case 2:
					delta = map[string]any{"role": "assistant", "content": "已检查工作区说明。"}
				case 3:
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "goal-get", "type": "function", "function": map[string]string{"name": "get_goal", "arguments": "{}"}}}}
					finish = "tool_calls"
				case 4:
					data, _ := json.Marshal(request)
					goalID := regexp.MustCompile(`goal-[0-9a-f-]{36}`).Find(data)
					revision := int64(1)
					live.mu.Lock()
					for _, row := range live.sessions {
						if row.Goal != nil {
							revision = row.Goal.DSHRevision
						}
					}
					live.mu.Unlock()
					args, _ := json.Marshal(map[string]any{"goal_id": string(goalID), "revision": revision, "action": "complete"})
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "goal-complete", "type": "function", "function": map[string]string{"name": "update_goal", "arguments": string(args)}}}}
					finish = "tool_calls"
				default:
					delta = map[string]any{"role": "assistant", "content": "模拟目标已完成。"}
				}
				stage = -1
			}
			if todoMode && n <= 2 {
				first, second := "in_progress", "pending"
				if n == 2 {
					select {
					case <-releaseGoal:
					case <-r.Context().Done():
						return nil, r.Context().Err()
					}
					first, second = "completed", "completed"
				}
				args, _ := json.Marshal(map[string]any{"todos": []map[string]string{{"content": "整理测试画布", "status": first}, {"content": "交付测试画布", "status": second}}})
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("goal-todo-%d", n), "type": "function", "function": map[string]string{"name": "todo_write", "arguments": string(args)}}}}
				finish = "tool_calls"
				stage = -1
			}
			if goalMode && os.Getenv("DSH_WEB_DYNAMIC") == "connector" {
				switch n {
				case 1:
					alias := (connectorTool{ConnectorID: connectorID, Name: "perform"}).alias()
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "goal-connector", "type": "function", "function": map[string]string{"name": alias, "arguments": `{"message":"目标模拟内容"}`}}}}
					finish = "tool_calls"
				case 2:
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "goal-get", "type": "function", "function": map[string]string{"name": "get_goal", "arguments": "{}"}}}}
					finish = "tool_calls"
				case 3:
					data, _ := json.Marshal(request)
					goalID := regexp.MustCompile(`goal-[0-9a-f-]{36}`).Find(data)
					arg, _ := json.Marshal(map[string]any{"goal_id": string(goalID), "revision": 1, "action": "complete"})
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "goal-complete", "type": "function", "function": map[string]string{"name": "update_goal", "arguments": string(arg)}}}}
					finish = "tool_calls"
				default:
					delta = map[string]any{"role": "assistant", "content": "模拟目标已完成。"}
				}
				stage = -1
			}
			if teamSwitch {
				var tool string
				delta, tool = dshFixtureTeamSwitch(request, n)
				if tool != "" {
					finish = "tool_calls"
				}
				if tool == "team_task_create" {
					select {
					case <-releaseGoal:
					case <-r.Context().Done():
						return nil, r.Context().Err()
					}
				}
				stage = -1
			}
			if memberMode {
				delta, finish = dshFixtureMemberReply(request, n, &memberRootCalls, &designerCalls, holdMember, memberChat)
				designerFinishing := delta["content"] == "## 设计检查结论\n\n**层级已核对**。\n\n- 标题清晰\n- 正文可读"
				if limitOneMember && designerFinishing && memberOutputLimited.CompareAndSwap(false, true) {
					delta["content"], finish = "尚待补充的设计说明", "length"
				}
				if memberChat && !stopOneMember && !stopColdMember && !limitOneMember && designerFinishing {
					memberModelHeld.Store(true)
					select {
					case <-releaseGoal:
					case <-r.Context().Done():
						return nil, r.Context().Err()
					}
				}
				if failMember && delta["content"] == "等待中的父会话已经结束。" {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprint(w, `{"error":{"type":"invalid_request_error","message":"模拟模型拒绝请求"}}`)
					return w.Result(), nil
				}
				if (holdMember && delta["content"] == "完成等待中的成员检查。") || (stopOneMember && designerFinishing) || (stopColdMember && delta["content"] == "等待本轮画布确认。") {
					session, invocation := r.Header.Get("X-MAI-Session-ID"), r.Header.Get("Idempotency-Key")
					hash, err := dshCancelRequestHash(dshLocalRecord{SessionID: session, ModelID: modelID, Body: body, Protocol: "openai_chat_completions"})
					if err != nil {
						return nil, err
					}
					memberReceiptPath.Store("/api/v1/sessions/" + session + "/model-invocations/" + invocation + "/receipt")
					memberReceiptHash.Store(hash)
					if !failMember && !beforeMemberHeaders {
						reader, writer := io.Pipe()
						go func() {
							defer writer.Close()
							if _, err := io.WriteString(writer, "data: {\"id\":\"fixture\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"正在检查。\"},\"finish_reason\":null}]}\n\n"); err != nil {
								return
							}
							memberModelHeld.Store(true)
							select {
							case <-releaseGoal:
							case <-r.Context().Done():
								memberModelCancelled.Store(true)
							}
						}()
						header := http.Header{"Content-Type": {"text/event-stream"}}
						header.Set("X-Billing-Transaction-ID", memberTransaction)
						return &http.Response{StatusCode: http.StatusOK, Header: header, Body: reader}, nil
					}
					memberModelHeld.Store(true)
					select {
					case <-releaseGoal:
					case <-r.Context().Done():
						memberModelCancelled.Store(true)
						return nil, r.Context().Err()
					}
				}
				stage = -1
			}
			switch stage {
			case 0:
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("call-%d", n), "type": "function", "function": map[string]string{"name": "read", "arguments": `{"file_path":"brief.txt"}`}}}}
				finish = "tool_calls"
			case 1:
				delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("call-%d", n), "type": "function", "function": map[string]string{"name": "write", "arguments": `{"file_path":"result.txt","content":"已整理工作区说明。\n"}`}}}}
				finish = "tool_calls"
			}
			uxStream := false
			if uxMode {
				switch uxScenario {
				case "验收长文":
					if stage == 2 && !toolFailed {
						delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("call-%d", n), "type": "function", "function": map[string]string{"name": "read", "arguments": `{"file_path":"result.txt"}`}}}}
						finish = "tool_calls"
					} else if stage >= 3 || stage >= 2 && toolFailed {
						if toolFailed {
							delta = map[string]any{"role": "assistant", "content": "模拟执行未完成，请查看失败步骤。"}
						} else {
							uxStream = true
						}
					}
				case "验收缺失":
					if stage == 0 {
						delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("call-%d", n), "type": "function", "function": map[string]string{"name": "read", "arguments": `{"file_path":"missing.txt"}`}}}}
						finish = "tool_calls"
					} else {
						delta = map[string]any{"role": "assistant", "content": "模拟读取失败，请查看执行步骤。"}
						finish = "stop"
					}
				case "验收停止":
					uxStream = true
				case "验收继续":
					delta = map[string]any{"role": "assistant", "content": "模拟续聊已完成。"}
					finish = "stop"
				}
			}
			if os.Getenv("DSH_WEB_NATIVE_APPROVAL") == "1" {
				if stage == 0 {
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("native-approval-%d", n), "type": "function", "function": map[string]string{"name": "bash", "arguments": `{"description":"验证原生审批","command":"printf 'NATIVE_APPROVAL_OK\\n'","sandbox_permissions":"workspace-write","justification":"需要在隔离工作区验证原生审批"}`}}}}
					finish = "tool_calls"
				} else {
					delta = map[string]any{"role": "assistant", "content": "原生审批测试已结束，请查看执行步骤。"}
					finish = "stop"
				}
			}
			if os.Getenv("DSH_WEB_NATIVE_QUESTION") == "1" {
				if n == 1 {
					delta = map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "question-1", "type": "function", "function": map[string]string{"name": "ask_user_question", "arguments": `{"questions":[{"id":"approach","question":"采用哪种处理方式？","header":"验收问题","options":[{"label":"方案一","description":"继续当前任务"},{"label":"方案二","description":"切换处理方式"}]},{"id":"detail","question":"还有什么需要说明？"}]}`}}}}
					finish = "tool_calls"
				} else {
					wire, _ := json.Marshal(request)
					if strings.Contains(string(wire), "方案一") && strings.Contains(string(wire), "补充说明") {
						fmt.Println("DSH_WEB_NATIVE_QUESTION_ANSWERED=1")
						delta = map[string]any{"role": "assistant", "content": "已收到所选方案和补充说明。"}
					} else if strings.Contains(string(wire), "方案一") && strings.Contains(string(wire), "\\\"id\\\":\\\"detail\\\",\\\"selected\\\":[]") {
						fmt.Println("DSH_WEB_NATIVE_QUESTION_SKIPPED=1")
						delta = map[string]any{"role": "assistant", "content": "已收到所选方案，第二题已跳过。"}
					} else {
						delta = map[string]any{"role": "assistant", "content": "提问未收到完整回答。"}
					}
				}
				stage = -1
			}
			if os.Getenv("DSH_WEB_REASONING") == "1" {
				delta = map[string]any{"role": "assistant", "content": "模拟任务已完成。"}
				finish, stage = "stop", -1
			}
			if os.Getenv("DSH_WEB_REASONING_HOLD") == "1" || os.Getenv("DSH_WEB_CANCEL_RECOVERY") == "1" && n == 1 {
				reader, writer := io.Pipe()
				go func() {
					defer writer.Close()
					frame, _ := json.Marshal(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{"reasoning_content": "先检查任务条件，再整理结果。", "content": "已开始处理。"}, "finish_reason": nil}}})
					if _, err := fmt.Fprintf(writer, "data: %s\n\n", frame); err != nil {
						return
					}
					fmt.Println("DSH_WEB_REASONING_HELD=1")
					<-r.Context().Done()
				}()
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: reader}, nil
			}
			if autoCompact || os.Getenv("DSH_WEB_NATIVE_COMPACT") == "1" {
				body, _ := json.Marshal(request)
				text := strings.Repeat("这是用于验证压缩结果的隔离会话资料。", 300)
				if dshSyntheticCompactionRequest(body) {
					if autoCompact || os.Getenv("DSH_WEB_NATIVE_COMPACT_DELAY") == "1" {
						fmt.Println("DSH_WEB_NATIVE_COMPACT_HELD=1")
						select {
						case <-r.Context().Done():
							return nil, r.Context().Err()
						case <-time.After(8 * time.Second):
						}
					}
					text = "模拟摘要：已核对隔离会话资料。"
					compactCalls.Add(1)
					fmt.Println("DSH_WEB_NATIVE_COMPACT_MODEL=1")
				}
				delta = map[string]any{"role": "assistant", "content": text}
				finish = "stop"
				stage = 0
			}
			if uxStream {
				reader, writer := io.Pipe()
				go func() {
					defer writer.Close()
					fmt.Printf("DSH_WEB_UX_STREAM_START=%s\n", uxScenario)
					chunks := 16
					if uxScenario == "验收停止" {
						chunks = 80
					}
					for i := 1; i <= chunks; i++ {
						part := fmt.Sprintf("模拟流段 %02d：这是一段用于检查连续输出、断线恢复和重复内容的测试文本。\n\n", i)
						frame, _ := json.Marshal(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "content": part}, "finish_reason": nil}}})
						if _, err := fmt.Fprintf(writer, "data: %s\n\n", frame); err != nil {
							return
						}
						select {
						case <-r.Context().Done():
							return
						case <-time.After(700 * time.Millisecond):
						}
					}
					frame, _ := json.Marshal(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": "stop"}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 16, "total_tokens": 19}})
					if _, err := fmt.Fprintf(writer, "data: %s\n\ndata: [DONE]\n\n", frame); err == nil {
						fmt.Println("DSH_WEB_UX_STREAM_END=1")
					}
				}()
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: reader}, nil
			}
			if stage >= 2 && !uxMode {
				text := delta["content"].(string)
				runes := []rune(text)
				reader, writer := io.Pipe()
				go func() {
					defer writer.Close()
					emit := func(v any) error {
						b, _ := json.Marshal(v)
						_, err := fmt.Fprintf(writer, "data: %s\n\n", b)
						return err
					}
					for _, part := range []string{string(runes[:len(runes)/2]), string(runes[len(runes)/2:])} {
						if emit(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "content": part}, "finish_reason": nil}}}) != nil {
							return
						}
						select {
						case <-r.Context().Done():
							return
						case <-time.After(2 * time.Second):
						}
					}
					if emit(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": "stop"}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4}}) == nil {
						_, _ = fmt.Fprint(writer, "data: [DONE]\n\n")
					}
				}()
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: reader}, nil
			}
			emit := func(v any) { b, _ := json.Marshal(v); fmt.Fprintf(w, "data: %s\n\n", b) }
			if os.Getenv("DSH_WEB_REASONING") == "1" {
				emit(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "reasoning_content": "先检查任务条件，再整理结果。"}, "finish_reason": nil}}})
			}
			emit(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": nil}}})
			emit(map[string]any{"id": "fixture", "object": "chat.completion.chunk", "created": 1, "model": modelID, "choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": finish}}, "usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 1, "total_tokens": 4}})
			fmt.Fprint(w, "data: [DONE]\n\n")
		default:
			fmt.Fprint(w, `{}`)
		}
		return w.Result(), nil
	})
	selected := selectedModelConfig{ID: modelID, Protocol: "openai_chat_completions", ContextWindowTokens: 100000, MaxOutputTokens: 4096, SupportsVision: visionMode}
	if autoCompact {
		selected.ContextWindowTokens, selected.MaxOutputTokens = 32768, 2048
	}
	gw := &dshGateway{client: c, authorization: func(context.Context) (authorization, error) { return authorization{Access: "fixture"}, nil }}
	cfg, err := newDSHGatewayRuntime(dshInstalledRuntime{Executable: node, Entry: entry, Home: filepath.Join(root, "home"), Env: []string{"PATH=" + os.Getenv("PATH")}}, filepath.Join(root, "journal"), gw, func(ctx context.Context, row acpSession) (selectedModelConfig, error) {
		if switchModel && row.Model == alternateModel {
			return c.modelConfig(ctx, authorization{Access: "fixture"}, alternateModel)
		}
		return selected, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg.LeasePool = &dshLeasePool{}
	cfg.ContinueAfterFailedTurn = modelFailures || conversationReview
	t.Cleanup(func() { _ = cfg.LeasePool.close(context.Background()) })
	if goalHold {
		bind := cfg.BindLease
		var admissions atomic.Int32
		cfg.BindLease = func(ctx context.Context, session acpSession, operation string, local *dshLocalGateway) error {
			if admit, ok := ctx.Value(dshGoalAdmissionKey{}).(func(string) error); ok {
				ctx = context.WithValue(ctx, dshGoalAdmissionKey{}, func(id string) error {
					if admissions.Add(1) == 2 {
						goalHeld.Store(true)
						<-releaseGoal
					}
					return admit(id)
				})
			}
			return bind(ctx, session, operation, local)
		}
	}
	live = &acpLive{client: c, auth: authorization{Access: "fixture"}, dir: root, cwd: root, dsh: cfg, store: &desktopSessionStore{Root: handle, lock: fixtureLock}, workspaces: workspaces, operations: map[string]string{}, sessions: map[string]acpSession{}}
	var attachments *os.Root
	if legacyMode || attachmentMode || imageHistoryMode {
		attachments, err = openDesktopAttachments(root)
		if err != nil {
			t.Fatal(err)
		}
		defer attachments.Close()
	}
	if legacyMode {
		legacyID, createErr := live.createInWorkspace("66666666-6666-4666-8666-666666666666", "", "", modelID, expertSelection{}, nil, "", desktopMediaModels{})
		if createErr != nil {
			t.Fatal(createErr)
		}
		const attachmentID = "77777777-7777-4777-8777-777777777777"
		if err = attachments.MkdirAll(attachmentID, 0700); err != nil {
			t.Fatal(err)
		}
		attachmentPath := filepath.Join(attachments.Name(), attachmentID, "旧资料.txt")
		if err = os.WriteFile(attachmentPath, []byte("旧附件的示例内容。\n"), 0600); err != nil {
			t.Fatal(err)
		}
		legacy := live.sessions[legacyID]
		legacy.Engine, legacy.DSHSessionID = "", ""
		now := time.Now().UTC().Add(-time.Minute)
		legacy.Created, legacy.Updated = now, now
		legacy.Turns = []acpTurn{{Operation: "88888888-8888-4888-8888-888888888888", Model: modelID, Started: now, Ended: now, Acked: true, Prompt: "旧会话：请查看附件", Answer: "旧引擎历史回复。", Attachments: []acpAttachment{{Name: "旧资料.txt", URI: (&url.URL{Scheme: "file", Path: filepath.ToSlash(attachmentPath)}).String()}}}}
		if os.Getenv("DSH_WEB_MIGRATION") == "1" {
			legacy.Turns[0].ModelInput = "旧会话：请查看附件\n\n[Attachment: 旧资料.txt]\n旧附件的示例内容。\n"
		}
		if err = live.save(legacy); err != nil {
			t.Fatal(err)
		}
		if os.Getenv("DSH_WEB_MIGRATION") == "1" {
			textID, e := live.createInWorkspace("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", "", "", modelID, expertSelection{}, nil, "", desktopMediaModels{})
			if e != nil {
				t.Fatal(e)
			}
			text := live.sessions[textID]
			text.Engine, text.DSHSessionID = "", ""
			text.Created, text.Updated = now, now
			text.Turns = []acpTurn{{Operation: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Model: modelID, Started: now, Ended: now, Acked: true, Prompt: "纯文本旧会话", Answer: "纯文本历史回复。"}}
			if e = live.save(text); e != nil {
				t.Fatal(e)
			}
		}
	}
	if goalMode {
		budget := int64(10000)
		if value := os.Getenv("DSH_WEB_GOAL_BUDGET"); value != "" {
			budget, err = strconv.ParseInt(value, 10, 64)
			if err != nil || budget <= 0 {
				t.Fatal("DSH_WEB_GOAL_BUDGET must be a positive integer")
			}
		}
		live.settings.GoalBudget = &budget
	}
	id, err := live.createInWorkspace("55555555-5555-4555-8555-555555555555", "", "", modelID, expertSelection{}, nil, "", desktopMediaModels{})
	if err != nil {
		t.Fatal(err)
	}
	row, err := live.load(id)
	if err != nil {
		t.Fatal(err)
	}
	if memberMode {
		enabled := true
		row.AgentTeamEnabled, row.AgentTeamAvailable = &enabled, &enabled
		if err := live.save(row); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(row.CWD, "skills", "visual-review")
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		var skill strings.Builder
		skill.WriteString("---\nname: visual-review\ndescription: 设计交付检查\n---\n")
		for i := 1; i <= 90; i++ {
			fmt.Fprintf(&skill, "技能检查项 %03d 核对页面交付记录。\n", i)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skill.String()), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if imageHistoryMode {
		png, e := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/aS8AAAAASUVORK5CYII=")
		if e != nil {
			t.Fatal(e)
		}
		webp, e := os.ReadFile(filepath.Join("..", "code", "web", "design-styles", "document-docs-reference.webp"))
		if e != nil {
			t.Fatal(e)
		}
		img := imagepkg.NewRGBA(imagepkg.Rect(0, 0, 1, 1))
		var jpg, animated bytes.Buffer
		if e = jpeg.Encode(&jpg, img, nil); e != nil {
			t.Fatal(e)
		}
		if e = gif.Encode(&animated, img, nil); e != nil {
			t.Fatal(e)
		}
		files := []struct {
			name string
			data []byte
		}{{"fixture.png", png}, {"fixture.jpg", jpg.Bytes()}, {"fixture.webp", webp}, {"fixture.gif", animated.Bytes()}}
		var saved []acpAttachment
		for _, file := range files {
			fileID, e := newUUID()
			if e != nil {
				t.Fatal(e)
			}
			if e = attachments.MkdirAll(fileID, 0700); e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(attachments.Name(), fileID, file.name)
			if e = os.WriteFile(path, file.data, 0600); e != nil {
				t.Fatal(e)
			}
			saved = append(saved, acpAttachment{Name: file.name, URI: (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()})
		}
		now := time.Now().UTC()
		row.Turns = []acpTurn{{Operation: "99999999-9999-4999-8999-999999999999", Model: modelID, Started: now, Ended: now, Acked: true, Prompt: "检查四种图片的历史记录", Answer: "模拟历史回复。", Attachments: saved}}
		if err = live.save(row); err != nil {
			t.Fatal(err)
		}
	}
	if os.Getenv("DSH_WEB_DYNAMIC") == "connector" {
		prefs := connectorPreferences{Enabled: map[string]bool{"mcp/server/" + connectorID: true}, Bindings: map[string]string{connectorID: credentialID}}
		row.ConnectorPreferences = &prefs
		if err = live.save(row); err != nil {
			t.Fatal(err)
		}
	}
	filePanelMode := os.Getenv("DSH_WEB_FILES") == "1"
	if filePanelMode {
		now := time.Now().UTC()
		row.Turns = []acpTurn{{
			Operation: "99999999-9999-4999-8999-999999999999", Model: modelID, Started: now, Ended: now, Acked: true,
			Prompt: "交付指南和示例", Answer: "[查看开发指南](./指南/README.md)",
			DSHDeliverables: []dshPresentedFile{{Path: "指南/example.js", Description: "示例代码"}},
		}}
		if err := live.save(row); err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(row.CWD, "指南")
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		const sheetNS = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
		const officeRel = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
		const packageRel = "http://schemas.openxmlformats.org/package/2006/relationships"
		const sheet = `<worksheet xmlns="` + sheetNS + `"><sheetData><row r="1"><c r="A1"><v>42</v></c></row></sheetData></worksheet>`
		workbook := officeTestArchive(t, map[string]string{
			"[Content_Types].xml":        `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
			"_rels/.rels":                `<Relationships xmlns="` + packageRel + `"><Relationship Id="rId1" Type="` + officeRel + `/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
			"xl/workbook.xml":            `<workbook xmlns="` + sheetNS + `" xmlns:r="` + officeRel + `"><sheets><sheet name="预算" sheetId="1" r:id="rId1"/><sheet name="实际支出" sheetId="2" r:id="rId2"/></sheets></workbook>`,
			"xl/_rels/workbook.xml.rels": `<Relationships xmlns="` + packageRel + `"><Relationship Id="rId1" Type="` + officeRel + `/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="` + officeRel + `/worksheet" Target="worksheets/sheet2.xml"/></Relationships>`,
			"xl/worksheets/sheet1.xml":   sheet,
			"xl/worksheets/sheet2.xml":   sheet,
		})
		if err := os.WriteFile(filepath.Join(dir, "开支.xlsx"), workbook, 0600); err != nil {
			t.Fatal(err)
		}
		for name, contents := range map[string]string{
			"README.md":      "---\ntitle: 文档\n---\n# 工作区指南\n\n[同目录说明](./note.txt)\n\n![远程图片](https://remote-image.invalid/example.png)\n\n```js\nconst language = \"Chinese\";\n```\n\n[跳至验证章节](#验证章节) · [另一份文档](./%E5%8F%82%E8%80%83.md#%E9%AA%8C%E8%AF%81%E7%AB%A0%E8%8A%82) · [定位配置](./settings.json#L3) · [查看网页](https://docs.example.invalid/demo#setup)\n\n## 验证章节\n\n当前文档的目标。\n\n[打开长文](./long-guide.md#第二页目标) · [定位大文件行](./long-script.js#L750)",
			"参考.md":          "# 参考资料\n\n## 验证章节\n\n另一份文档的目标。",
			"long-guide.md":  "# 长文首页\n\n[分页中的本地链接](./note.txt)\n\n" + strings.Repeat("文档内容。\n", 550) + "## 第二页目标\n\n" + strings.Repeat("长文内容。\n", 80000),
			"long-script.js": strings.Repeat("const example = true; // "+strings.Repeat("x", 80)+"\n", 749) + "const target750 = true;\n" + strings.Repeat("const later = false; // "+strings.Repeat("y", 80)+"\n", 10000),
			"settings.json":  "{\n  \"enabled\": true,\n  \"label\": \"演示\"\n}",
			"note.txt":       "同目录定位成功。",
			"example.js":     "const accepted = true;",
			"图示.svg":         `<svg xmlns="http://www.w3.org/2000/svg" width="1360" height="600"><rect width="1360" height="600" fill="gold"/></svg>`,
			"销售.csv":         "名称,金额\n甲,12\n乙,24",
			"viewer.html":    `<!doctype html><html lang="zh"><title>查看器验收</title><h1>查看器验收</h1><button onclick="this.textContent='交互已运行'">运行示例</button></html>`,
		} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err = os.WriteFile(filepath.Join(row.CWD, "brief.txt"), []byte("请整理这份工作区说明。\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if modelFailures {
		row.Queue.Paused = true
		if err = live.save(row); err != nil {
			t.Fatal(err)
		}
	}
	if os.Getenv("DSH_WEB_PROMPT_RECOVERY") == "1" {
		now := time.Now().UTC()
		turn := acpTurn{Operation: "99999999-9999-4999-8999-999999999999", Model: modelID, Started: now, Prompt: "模拟中断的普通聊天"}
		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		err = runDSHTurn(ctx, cfg, row, &turn, false, nil)
		cancel()
		if err != nil || turn.Answer == "" || len(turn.ToolRuns) != 0 {
			t.Fatalf("prepare prompt recovery history: %v", err)
		}
		turn.Answer = ""
		turn.Ended = time.Now().UTC()
		turn.Acked, turn.Failed, turn.FailureCode = true, true, "dsh_runtime_exited"
		row.State, row.Turns = "error", []acpTurn{turn}
		if err = live.save(row); err != nil {
			t.Fatal(err)
		}
	}
	if os.Getenv("DSH_WEB_RECOVERY") == "1" {
		now := time.Now().UTC()
		call := &modelToolUse{ID: "fixture-write", Type: "function"}
		call.Function.Name = "write"
		call.Function.Arguments = `{"file_path":"demo.txt","content":"fixture"}`
		row.State = "error"
		row.Turns = []acpTurn{{Operation: "99999999-9999-4999-8999-999999999999", Model: modelID, Started: now, Ended: now, Acked: true, Failed: true, FailureCode: "dsh_write_outcome_unknown", Prompt: "模拟中断写入", ToolRuns: []workspaceToolRun{{DSHCallID: "fixture-write", Pending: true, Call: call}}}}
		if err = live.save(row); err != nil {
			t.Fatal(err)
		}
	}
	if os.Getenv("DSH_WEB_STRUCTURED") == "1" {
		fixtures := []struct{ name, title, input, output string }{
			{"knowledge_search", "搜索知识库", `{"query":"产品规范"}`, `{"items":[{"content":"模拟知识库参考内容。","score":0.87,"citation":{"heading_path":"产品 / 指南"}}]}`},
			{"canvas_get_document", "读取画布", `{"title":"首页"}`, `{"title":"首页","nodes":[{"name":"主标题","text":"欢迎使用"}]}`},
			{"connector_0123456789abcdef0123456789abcdef", "模拟连接器 / 查询记录", `{"name":"示例记录"}`, `{"content":[{"type":"text","text":"模拟连接器结果。"}],"count":3}`},
			{"web_search", "web_search", `{"queries":["模拟查询"]}`, `模拟网页搜索结果。`},
			{"web_fetch", "web_fetch", `{"url":"https://example.com"}`, `模拟网页内容。`},
			{"subagent", "subagent", `{"description":"模拟子任务","prompt":"检查数据"}`, `模拟子任务结果。`},
			{"skill", "skill", `{"name":"模拟技能"}`, `模拟技能已加载。`},
			{"job_list", "job_list", `{}`, `模拟后台任务列表。`},
			{"custom_native_tool", "自定义原生工具", `{"query":"保留原生参数"}`, `模拟原生工具错误。`},
		}
		now := time.Now().UTC()
		turn := acpTurn{Operation: "99999999-9999-4999-8999-999999999999", Model: modelID, Started: now, Ended: now, Acked: true, Prompt: "查看模拟工具步骤", Answer: "模拟步骤已结束。"}
		for i, fixture := range fixtures {
			call := &modelToolUse{ID: fmt.Sprintf("fixture-tool-%d", i), Type: "function"}
			call.Function.Name, call.Function.Arguments = fixture.name, fixture.input
			report := reportedTool{Category: "builtin", Name: fixture.name, Calls: 1}
			if fixture.name == "custom_native_tool" {
				report.Failed = 1
			}
			turn.ToolRuns = append(turn.ToolRuns, workspaceToolRun{DSHCallID: call.ID, DisplayName: fixture.title, Call: call, Report: report, Output: fixture.output})
		}
		row.Turns = []acpTurn{turn}
		if err = live.save(row); err != nil {
			t.Fatal(err)
		}
	}
	address := "127.0.0.1:18767"
	if os.Getenv("DSH_WEB_DYNAMIC") == "canvas_write" {
		address = "127.0.0.1:18931"
	}
	if os.Getenv("DSH_WEB_DYNAMIC") == "connector" {
		address = "127.0.0.1:18769"
	}
	if os.Getenv("DSH_WEB_MIGRATION") == "1" {
		address = "127.0.0.1:18768"
	}
	if raw := os.Getenv("DSH_WEB_PORT"); raw != "" {
		port, parseErr := strconv.Atoi(raw)
		if parseErr != nil || port < 1 || port > 65535 {
			t.Fatal("invalid DSH_WEB_PORT")
		}
		address = fmt.Sprintf("127.0.0.1:%d", port)
	}
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	host := &relaySessionHost{profile: root, live: live, local: &desktopRuntimeAccess{address: listener.Addr().String(), origin: "http://" + listener.Addr().String(), protocol: "monkeyai.token.fixture", reportMachine: "machine", attachments: attachments}, goals: newGoalRuntime(), terminalLifetime: t.Context()}
	if conversationReview {
		host.interactions = newDesktopDSHInteractions()
	}
	if modelFailures {
		live.reportWake = make(chan struct{}, 1)
		reportCtx, cancelReports := context.WithCancel(t.Context())
		reportDone := make(chan struct{})
		go func() {
			defer close(reportDone)
			host.superviseDesktopReports(reportCtx, 30*time.Second)
		}()
		defer func() {
			cancelReports()
			<-reportDone
		}()
	}
	if os.Getenv("DSH_WEB_MIGRATION") == "1" {
		host.local.userID = "11111111-1111-4111-8111-111111111111"
		host.local.serverID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	}
	canvasDocument := `{"schemaVersion":1,"id":"eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee","title":"模拟画布","revision":1,"createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-01-01T00:00:00Z","nodes":[{"id":"fixture-text","type":"text","title":"主标题","content":"画布验收内容。","contentFormat":"markdown","textRole":"note","x":100,"y":100,"width":300,"height":150,"zIndex":0,"fontFamily":"Arial","fontSize":16,"fontWeight":400,"color":"#111111","textAlign":"left","lineHeight":1.6,"letterSpacing":0}]}`
	if os.Getenv("DSH_WEB_DYNAMIC") == "canvas" || os.Getenv("DSH_WEB_DYNAMIC") == "canvas_catalog" {
		const canvasID = "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee"
		base := filepath.Join(row.CWD, workspaceDataDir, "design")
		for _, dir := range []string{filepath.Join(base, "sessions"), filepath.Join(base, canvasID)} {
			if err = os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
		}
		files := map[string]string{
			filepath.Join(base, "sessions", id+".json"):          `{"canvasId":"` + canvasID + `","localDesignPath":"` + workspaceDataDir + `/design/` + canvasID + `"}`,
			filepath.Join(base, canvasID, "design-context.json"): `{"designSessionId":"` + id + `","canvasId":"` + canvasID + `"}`,
			filepath.Join(base, canvasID, "canvas.json"):         canvasDocument,
		}
		for path, content := range files {
			if err = os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
		}
		bridge := &desktopDesignBridge{pending: map[string]chan designToolResult{}}
		bridge.send = func(event any) error {
			request := event.(map[string]any)
			name, requestID := request["name"].(string), request["id"].(string)
			result := `{"title":"模拟画布","nodes":[{"name":"主标题","text":"画布验收内容。"}]}`
			switch name {
			case "design_tool_catalog":
				result = `[{"name":"canvas_get_document","description":"Read the canvas","inputSchema":{"type":"object","properties":{},"additionalProperties":false}},{"name":"canvas_get_selection","description":"Read selected canvas nodes","inputSchema":{"type":"object","properties":{},"additionalProperties":false}},{"name":"design_catalog_search","description":"Search design catalog","inputSchema":{"type":"object","properties":{"query":{"type":"string"}},"required":["query"],"additionalProperties":false}}]`
			case "design_guard_capabilities":
				result = `{"version":4,"tools":["canvas_get_document","canvas_get_selection"]}`
			case "canvas_get_selection":
				var args struct {
					NodeIDs []string `json:"node_ids"`
				}
				if json.Unmarshal(request["arguments"].(json.RawMessage), &args) != nil || len(args.NodeIDs) != 1 || args.NodeIDs[0] != "fixture-text" {
					return fmt.Errorf("unexpected synthetic canvas selection: %s", request["arguments"])
				}
				fmt.Println("DSH_WEB_CANVAS_READ=1")
			case "design_catalog_search":
				var args struct {
					Query string `json:"query"`
				}
				if json.Unmarshal(request["arguments"].(json.RawMessage), &args) != nil || args.Query != "海报" {
					return fmt.Errorf("unexpected synthetic catalog query: %s", request["arguments"])
				}
				result = `{"items":[{"title":"模拟设计模板"}]}`
				fmt.Println("DSH_WEB_CANVAS_CATALOG=1")
			default:
				return fmt.Errorf("unexpected synthetic canvas request: %s", name)
			}
			bridge.mu.Lock()
			ch := bridge.pending[requestID]
			bridge.mu.Unlock()
			ch <- designToolResult{Result: json.RawMessage(result)}
			return nil
		}
		host.designBridge = bridge
	}
	if os.Getenv("DSH_WEB_DYNAMIC") == "canvas_write" || os.Getenv("DSH_WEB_DYNAMIC") == "pptx" {
		electron, worker, adapter := os.Getenv("DSH_TEST_ELECTRON"), os.Getenv("DSH_TEST_CANVAS_WORKER"), os.Getenv("DSH_TEST_CANVAS_ADAPTER")
		if electron == "" || worker == "" || adapter == "" {
			t.Fatal("Electron canvas fixture not configured")
		}
		cmd := exec.CommandContext(t.Context(), electron, adapter, worker)
		in, e := cmd.StdinPipe()
		if e != nil {
			t.Fatal(e)
		}
		out, e := cmd.StdoutPipe()
		if e != nil {
			t.Fatal(e)
		}
		cmd.Stderr = os.Stderr
		if e = cmd.Start(); e != nil {
			t.Fatal(e)
		}
		defer func() { in.Close(); _ = cmd.Wait() }()
		bridge := &desktopDesignBridge{pending: map[string]chan designToolResult{}, send: func(v any) error { return json.NewEncoder(in).Encode(v) }}
		go bridge.read(out)
		if _, e = bridge.call(t.Context(), row, nil, "design_session_open", json.RawMessage(`{"designType":"slide_deck","designTypeSource":"user","templateRef":null,"designSystemRef":null,"mediaModelId":null}`)); e != nil {
			t.Fatal(e)
		}
		if os.Getenv("DSH_WEB_DYNAMIC") == "pptx" {
			root, e := live.workspaces.sessionRoot(row)
			if e != nil {
				t.Fatal(e)
			}
			design := workspaceDesign(root, id)
			root.Close()
			if _, e = bridge.call(t.Context(), row, design, "design_test_style", json.RawMessage(`{}`)); e != nil {
				t.Fatal(e)
			}
		}
		host.designBridge = bridge
	}
	if goalMode {
		go host.superviseGoals(t.Context())
	}
	mux := http.NewServeMux()
	mux.Handle("/acp", host)
	if attachmentMode {
		image, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/aS8AAAAASUVORK5CYII=")
		if err != nil {
			t.Fatal(err)
		}
		if visionMode {
			var encoded bytes.Buffer
			if err := png.Encode(&encoded, imagepkg.NewRGBA(imagepkg.Rect(0, 0, 1, 1))); err != nil {
				t.Fatal(err)
			}
			image = encoded.Bytes()
		}
		samples := map[string][]byte{
			"notes.txt":   []byte("附件文本验收内容"),
			"photo.png":   image,
			"report.docx": officeTestArchive(t, map[string]string{"word/document.xml": `<p><t>办公文档验收内容</t></p>`}),
		}
		mux.HandleFunc("/fixture/attachment", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "invalid fixture attachment", http.StatusBadRequest)
				return
			}
			files := make([]map[string]string, 0, len(samples))
			for _, name := range []string{"notes.txt", "photo.png", "report.docx"} {
				id, err := newUUID()
				if err == nil {
					err = attachments.MkdirAll(id, 0700)
				}
				if err == nil {
					err = os.WriteFile(filepath.Join(attachments.Name(), id, name), samples[name], 0600)
				}
				if err != nil {
					http.Error(w, "fixture staging failed", http.StatusInternalServerError)
					return
				}
				uri := (&url.URL{Scheme: "file", Path: filepath.ToSlash(filepath.Join(attachments.Name(), id, name))}).String()
				files = append(files, map[string]string{"name": name, "uri": uri})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(files)
		})
	}
	if modelFailures {
		mux.HandleFunc("POST /fixture/park-previous-invocation", func(w http.ResponseWriter, r *http.Request) {
			const invocation = "77777777-7777-4777-8777-777777777777"
			slot, err := cfg.LeasePool.acquire(r.Context(), id)
			if err != nil {
				http.Error(w, "session unavailable", http.StatusConflict)
				return
			}
			defer func() { <-slot.gate }()
			row, err := live.load(id)
			if err != nil || row.State != "error" || len(row.Turns) == 0 {
				http.Error(w, "expected a failed turn", http.StatusConflict)
				return
			}
			slot.discard()
			journal := filepath.Join(root, "journal", id)
			record := dshLocalRecord{SessionID: id, OperationID: row.Turns[len(row.Turns)-1].Operation,
				CallID: invocation, ModelID: modelID, Protocol: selected.Protocol, State: "dispatched",
				Status: http.StatusServiceUnavailable, Body: []byte("{}")}
			raw, _ := json.Marshal(record)
			if err := os.WriteFile(filepath.Join(journal, "pending.json"), raw, 0600); err != nil {
				http.Error(w, "could not seed an unresolved invocation", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
		mux.HandleFunc("POST /fixture/hold-stream-receipt", func(w http.ResponseWriter, _ *http.Request) {
			holdStreamReceipt.Store(true)
			w.WriteHeader(http.StatusNoContent)
		})
		mux.HandleFunc("POST /fixture/model-receipt", func(w http.ResponseWriter, _ *http.Request) {
			if calls.Load() == 9 {
				deferredStreamReceipt.Store(true)
				releaseStreamReceipt.Do(func() { close(streamReceiptRelease) })
			} else {
				deferredModelReceipt.Store(true)
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
	if conversationReview {
		mux.HandleFunc("POST /fixture/release-first-step", func(w http.ResponseWriter, _ *http.Request) {
			if goalReleased.CompareAndSwap(false, true) {
				close(releaseGoal)
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
	mux.HandleFunc("/fixture", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := map[string]any{"sessionId": id, "goalHeld": goalHeld.Load(), "modelCalls": calls.Load(), "compactCalls": compactCalls.Load()}
		if conversationReview {
			status["stepCancelled"], status["feedbackReceived"] = stepCancelled.Load(), feedbackReceived.Load()
			if current, err := live.load(id); err == nil {
				status["turnCount"] = len(current.Turns)
			}
		}
		if modelFailures {
			status["reportFailures"] = reportFailures.Load()
			status["streamReceiptWaiting"] = streamReceiptWaiting.Load()
			_, archiveErr := os.Stat(filepath.Join(root, "journal", id, "77777777-7777-4777-8777-777777777777.json"))
			status["previousInvocationArchived"] = archiveErr == nil
			if current, err := live.load(id); err == nil && len(current.Turns) > 0 {
				turn := current.Turns[len(current.Turns)-1]
				status["failure"], status["settled"], status["turnCount"], status["ended"], status["answer"] = turn.FailureCode, turn.DSHModelFailureSettled, len(current.Turns), !turn.Ended.IsZero(), turn.Answer
				status["queued"], status["queuePaused"] = len(current.Queue.Messages), current.Queue.Paused
				status["model"] = turn.Model
			}
			_, err := os.Stat(filepath.Join(root, "journal", id, "pending.json"))
			status["pending"] = err == nil
		}
		if memberChat {
			status["designerHeld"], status["stopColdMember"] = memberModelHeld.Load(), stopColdMember
			if current, e := live.load(id); stopColdMember && e == nil && len(current.Turns) > 0 {
				turn := current.Turns[len(current.Turns)-1]
				status["memberTurnClosed"] = turn.DSHMemberID != "" && turn.Interrupted && !turn.Failed && !turn.Ended.IsZero()
				status["memberTurnDone"] = turn.DSHMemberID != "" && !turn.Interrupted && !turn.Failed && !turn.Ended.IsZero() && current.State == "idle"
			}
			status["stopOneMember"], status["limitOneMember"] = stopOneMember, limitOneMember
			status["memberModelCancelled"] = memberModelCancelled.Load()
			status["receiptRequested"], status["receiptVerified"] = memberReceiptRequested.Load(), memberReceiptVerified.Load()
		}
		if holdMember {
			status["memberStop"], status["memberFailure"] = true, failMember
			_, pendingErr := os.Stat(filepath.Join(root, "journal", id, "pending.json"))
			status["journalPending"] = pendingErr == nil
			status["withholdReceipt"] = withholdReceipt
			status["memberModelHeld"] = memberModelHeld.Load()
			status["memberModelCancelled"] = memberModelCancelled.Load()
			status["receiptRequested"], status["receiptVerified"] = memberReceiptRequested.Load(), memberReceiptVerified.Load()
			if current, err := live.load(id); err == nil && len(current.Turns) > 0 {
				turn := current.Turns[len(current.Turns)-1]
				status["parentInterrupted"] = turn.Interrupted
				status["stopObserved"], status["turnCount"] = turn.DSHStopObserved, len(current.Turns)
				status["parentState"], status["parentFailure"], status["parentEnded"] = current.State, turn.FailureCode, !turn.Ended.IsZero()
				for _, run := range turn.ToolRuns {
					if run.Call != nil && run.Call.Function.Name == "wait_agent" {
						status["parentWait"] = true
					}
				}
			}
		}
		if goalMode {
			if current, err := live.load(id); err == nil && current.Goal != nil {
				status["goalStatus"] = current.Goal.Status
				status["turnStatus"] = current.State
				status["context"] = current.workspaceMetadata()["context"]
				status["dshSessionId"] = current.DSHSessionID
				host.goals.mu.Lock()
				status["viewers"] = len(host.goals.viewers)
				host.goals.mu.Unlock()
			}
		}
		json.NewEncoder(w).Encode(status)
	})
	if memberChat {
		mux.HandleFunc("POST /fixture/evict", func(w http.ResponseWriter, r *http.Request) {
			if err := cfg.LeasePool.evictIfIdle(id); err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})
	}
	finishFixture := make(chan struct{})
	if goalHold || teamSwitch || expertEditor || filePanelMode || todoMode || memberMode || autoCompact {
		var finishOnce sync.Once
		mux.HandleFunc("POST /fixture/finish", func(w http.ResponseWriter, _ *http.Request) {
			if goalReleased.CompareAndSwap(false, true) {
				close(releaseGoal)
			}
			finishOnce.Do(func() { close(finishFixture) })
			w.WriteHeader(http.StatusNoContent)
		})
		release := func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			if goalReleased.CompareAndSwap(false, true) {
				close(releaseGoal)
			}
			w.WriteHeader(http.StatusNoContent)
		}
		mux.HandleFunc("/goal/release", release)
		if teamSwitch {
			mux.HandleFunc("/fixture/team-release", release)
		}
	}
	mux.HandleFunc("/tool-actions.mjs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join("..", "code", "web", "dsh", "tool-actions.mjs"))
	})
	if os.Getenv("DSH_WEB_RECOVERY") == "success" {
		mux.HandleFunc("/fixture/recovery-ready", func(w http.ResponseWriter, r *http.Request) {
			live.mu.Lock()
			defer live.mu.Unlock()
			current := live.sessions[id]
			if len(current.Turns) == 0 {
				http.Error(w, "missing turn", 409)
				return
			}
			i := len(current.Turns) - 1
			if len(current.Turns[i].ToolRuns) == 0 {
				http.Error(w, "missing write", 409)
				return
			}
			last := &current.Turns[i].ToolRuns[len(current.Turns[i].ToolRuns)-1]
			if last.Call == nil || last.Call.Function.Name != "write" {
				http.Error(w, "last call not write", 409)
				return
			}
			claim, e := readConnectorClaim(handle, connectorClaimPath(id, current.Turns[i].Operation, last.DSHCallID), id)
			if e != nil || claim.Kind != "workspace" || claim.State != "reconciled" {
				http.Error(w, "missing verified receipt", 409)
				return
			}
			last.Pending = true
			current.Turns[i].Failed = true
			current.Turns[i].FailureCode = "dsh_write_outcome_unknown"
			current.State = "error"
			if e = live.save(current); e != nil {
				http.Error(w, "save failed", 500)
				return
			}
			fmt.Fprint(w, "recovery fixture ready")
		})
	}
	if os.Getenv("DSH_WEB_DYNAMIC") == "connector" {
		mux.HandleFunc("/fixture/connector-status", func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]any{"dispatches": connectorCalls.Load(), "revoked": revoked.Load()})
		})
		mux.HandleFunc("/fixture/connector-revoke", func(w http.ResponseWriter, r *http.Request) { revoked.Store(true); fmt.Fprint(w, "revoked") })
	}
	if os.Getenv("DSH_WEB_DYNAMIC") == "canvas_write" {
		mux.HandleFunc("/fixture/canvas-recovery-ready", func(w http.ResponseWriter, r *http.Request) {
			live.mu.Lock()
			defer live.mu.Unlock()
			current := live.sessions[id]
			if len(current.Turns) == 0 {
				http.Error(w, "missing turn", 409)
				return
			}
			turn := &current.Turns[len(current.Turns)-1]
			if len(turn.ToolRuns) == 0 {
				http.Error(w, "missing tool", 409)
				return
			}
			run := &turn.ToolRuns[len(turn.ToolRuns)-1]
			if run.Call == nil || run.Call.Function.Name != designTextTool {
				http.Error(w, "wrong tool", 409)
				return
			}
			claim, e := readConnectorClaim(handle, connectorClaimPath(id, turn.Operation, run.DSHCallID), id)
			if e != nil || claim.Kind != "canvas" || claim.State != "reconciled" || claim.SDKSeq == nil {
				http.Error(w, "missing verified receipt", 409)
				return
			}
			run.Pending = true
			turn.Failed = true
			turn.FailureCode = "dsh_canvas_outcome_unknown"
			turn.DSHReconciled = false
			current.State = "error"
			if live.save(current) != nil {
				http.Error(w, "save failed", 500)
				return
			}
			fmt.Fprint(w, "canvas recovery fixture ready")
		})
		mux.HandleFunc("/fixture/canvas-status", func(w http.ResponseWriter, r *http.Request) {
			root, e := workspaces.sessionRoot(row)
			if e != nil {
				http.Error(w, e.Error(), 500)
				return
			}
			defer root.Close()
			var marker struct{ LocalDesignPath string }
			_ = json.Unmarshal(workspaceDesign(root, id), &marker)
			data, e := os.ReadFile(filepath.Join(row.CWD, filepath.FromSlash(marker.LocalDesignPath), "canvas.json"))
			if e != nil {
				http.Error(w, e.Error(), 500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(data)
		})
	}
	mux.HandleFunc("/artifact", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(row.CWD, "result.txt"))
	})
	// Serve the checked-in workbench with a test-only browser host endpoint.
	// No product bundle is changed on disk; authentication stays synthetic.
	webRoot := filepath.Join("..", "code", "web")
	bundle, err := os.ReadFile(filepath.Join(webRoot, "assets", "index-zDHqw1Zn.js"))
	if err != nil {
		t.Fatal(err)
	}
	patched := string(bundle)
	replaceOne := func(old, replacement string) {
		t.Helper()
		if strings.Count(patched, old) != 1 {
			t.Fatal("workbench browser fixture marker changed")
		}
		patched = strings.Replace(patched, old, replacement, 1)
	}
	old := "getRuntimeEndpoint: async () => null,"
	replaceOne(old, `getRuntimeEndpoint: async () => ({httpBase: location.origin, token: "fixture", connectionId: "11111111-1111-4111-8111-111111111111"}),`)
	if modelFailures {
		// The browser fixture has a real local ACP endpoint but no Electron shell.
		// Exercise the same automatic receipt check as that shell.
		replaceOne(`L && et.kind === "electron" && ts()?.transport !== "relay" &&
          e.error === Se.dshReconciliationRequired`, `L && ts()?.transport !== "relay" &&
          e.error === Se.dshReconciliationRequired`)
		replaceOne(`.includes(J?.error) && et.kind === "electron" && ts()?.transport !== "relay"`, `.includes(J?.error) && ts()?.transport !== "relay"`)
	}
	if attachmentMode {
		old = `      et.kind !== "electron" ||
      !J,`
		if strings.Count(patched, old) != 1 {
			t.Fatal("workbench attachment availability changed")
		}
		patched = strings.Replace(patched, old, `      !J,`, 1)
		old = "pickFiles: async () => [],"
		if strings.Count(patched, old) != 1 {
			t.Fatal("workbench file picker adapter changed")
		}
		patched = strings.Replace(patched, old, `pickFiles: async () => { const response = await fetch("/fixture/attachment", {method: "POST"}); if (!response.ok) throw new Error("fixture staging failed"); return response.json(); },`, 1)
		old = `previewAttachment: async () => ({
      kind: "unavailable",
      reason: "unsupported",
    }),`
		if strings.Count(patched, old) != 1 {
			t.Fatal("workbench attachment preview adapter changed")
		}
		patched = strings.Replace(patched, old, `previewAttachment: async (uri) => uri.endsWith("/photo.png") ? {kind: "image", dataUrl: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/aS8AAAAASUVORK5CYII="} : {kind: "unavailable", reason: "unsupported"},`, 1)
	}
	if os.Getenv("DSH_WEB_DYNAMIC") == "canvas" || os.Getenv("DSH_WEB_DYNAMIC") == "canvas_write" || os.Getenv("DSH_WEB_DYNAMIC") == "canvas_catalog" || os.Getenv("DSH_WEB_DYNAMIC") == "pptx" {
		// Mirror the private design workspace read/selection responses used by the
		// workbench. This is a synthetic browser-only adapter, not a canvas worker.
		old = "async function Jr(e, t, n) {"
		if strings.Count(patched, old) != 1 {
			t.Fatal("workbench design workspace adapter changed")
		}
		patched = strings.Replace(patched, old, old+`
  const canvasId = "eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee";
  if (e === "design_state") return {revision: 1, designType: "prototype", designTypeSource: "user", templateRef: null, designSystemRef: null, mediaModelId: null, styleDecision: "skipped", phase: "BRIEFING", designSessionId: t.sessionId, canvasId, localDesignPath: ".aibertai/design/" + canvasId, snapshot: {localVersion: 0, remoteVersion: 0, status: "local"}, provider: {status: "unavailable"}};
  if (e === "canvas_get") return {document: `+canvasDocument+`, generationTasks: []};
  if (e === "canvas_select") return {selection: {nodeIds: t.nodeIds || [], focusedNodeId: t.focusedNodeId}};
  if (e === "canvas_selected_sources") return [];
`, 1)
	}
	serveWorkbench := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webRoot, "index.html"))
	}
	mux.HandleFunc("/workbench", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/sessions/"+id, http.StatusFound) })
	mux.HandleFunc("/{$}", serveWorkbench)
	mux.HandleFunc("/login", serveWorkbench)
	mux.HandleFunc("/settings/", serveWorkbench)
	mux.HandleFunc("/sessions/", serveWorkbench)
	mux.HandleFunc("/assets/index-zDHqw1Zn.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write([]byte(patched))
	})
	if filePanelMode {
		mux.HandleFunc(localWorkspaceFileRoute, host.localWorkspaceFile)
	}
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(filepath.Join(webRoot, "assets")))))
	mux.Handle("/dsh/", http.StripPrefix("/dsh/", http.FileServer(http.Dir(filepath.Join(webRoot, "dsh")))))
	mux.HandleFunc("/brand-icon.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webRoot, "brand-icon.png"))
	})
	mux.HandleFunc("/brand-logo.png", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webRoot, "brand-logo.png"))
	})
	files := http.FileServer(http.Dir(filepath.Join("..", "..", "tests", "dsh_web")))
	mux.Handle("/", files)
	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Close()
	fmt.Printf("DSH_WEB_FIXTURE=http://%s/\n", listener.Addr())
	select {
	case <-t.Context().Done():
	case <-finishFixture:
	}
}

func dshFixtureMemberReply(req map[string]any, n int64, roots, designer *atomic.Int32, hold, chat bool) (map[string]any, string) {
	var userText strings.Builder
	messages, _ := req["messages"].([]any)
	for _, raw := range messages {
		msg, _ := raw.(map[string]any)
		if msg["role"] == "user" {
			content, _ := json.Marshal(msg["content"])
			userText.Write(content)
		}
	}
	var tool, args, answer string
	switch {
	case strings.Contains(userText.String(), "等待中的设计检查"):
		answer = "完成等待中的成员检查。"
	case strings.Contains(userText.String(), "检查技能文件"):
		if strings.Contains(userText.String(), "请补一条移动端规范") {
			answer = "## 移动端补充\n\n已记录最小触控区域。"
		} else if strings.Contains(userText.String(), "先等我确认画布") {
			answer = "等待本轮画布确认。"
		} else if strings.Contains(userText.String(), "请改用冷色") {
			answer = "## 配色补充\n\n已按你的追加要求改用冷色。"
		} else if designer.Add(1) == 1 {
			tool, args = "read", `{"file_path":"skills/visual-review/SKILL.md"}`
		} else {
			answer = "## 设计检查结论\n\n**层级已核对**。\n\n- 标题清晰\n- 正文可读"
		}
	case strings.Contains(userText.String(), "核对审稿交付"):
		answer = "## 校对检查结论\n\n**校对完成**。"
	case hold:
		switch roots.Add(1) {
		case 1:
			tool, args = "todo_write", `{"todos":[{"content":"整理待办资料","status":"in_progress"}]}`
		case 2:
			tool, args = "team_task_create", `{"subject":"共享设计检查","description":"核对交付物"}`
		case 3:
			tool, args = "team_task_update", `{"task_id":"task-1","expected_revision":1,"action":"claim"}`
		case 4:
			tool, args = "subagent", `{"name":"designer","description":"设计检查","prompt":"等待中的设计检查"}`
		case 5:
			tool, args = "wait_agent", `{"timeout_ms":60000}`
		default:
			answer = "等待中的父会话已经结束。"
		}
	case chat:
		switch roots.Add(1) {
		case 1:
			tool, args = "subagent", `{"name":"designer","description":"设计检查","prompt":"检查技能文件并总结结论。"}`
		case 2, 4:
			tool, args = "wait_agent", `{"timeout_ms":60000}`
		case 3:
			tool, args = "subagent", `{"name":"reviewer","description":"校对检查","prompt":"核对审稿交付。"}`
		default:
			answer = "主会话已收到成员结论。"
		}
	default:
		switch roots.Add(1) {
		case 1:
			tool, args = "subagent", `{"name":"designer","description":"设计检查","prompt":"检查技能文件并总结结论。"}`
		case 2:
			tool, args = "subagent_fork", `{"name":"reviewer","description":"校对检查","prompt":"核对审稿交付。"}`
		case 3:
			tool, args = "wait_agent", `{"timeout_ms":10000}`
		default:
			answer = "主会话已收到成员结论。"
		}
	}
	if tool == "" {
		return map[string]any{"role": "assistant", "content": answer}, "stop"
	}
	call := map[string]any{"index": 0, "id": fmt.Sprintf("member-%d", n), "type": "function", "function": map[string]string{"name": tool, "arguments": args}}
	return map[string]any{"role": "assistant", "tool_calls": []any{call}}, "tool_calls"
}
