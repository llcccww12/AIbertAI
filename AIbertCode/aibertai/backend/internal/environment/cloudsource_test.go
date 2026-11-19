package environment

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"aibertai.local/backend/internal/apikey"
	"aibertai.local/backend/internal/endpoint"
	"aibertai.local/backend/internal/identity"
	"aibertai.local/backend/internal/relay"
	"aibertai.local/backend/internal/resource"
	"aibertai.local/backend/internal/session"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/go-chi/chi/v5"
)

// Run on Linux with a writable /workspace, the repository migrations, and an
// isolated Postgres database. The model catalog and upstream are fixtures.
func TestCloudSourceRuntimeDocker(t *testing.T) {
	binary := os.Getenv("AIBERTAI_TEST_CLOUD_BINARY")
	if binary == "" {
		t.Skip("set AIBERTAI_TEST_CLOUD_BINARY in the Linux cloud integration environment")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	s, user, host := fixture(t)
	server := httptest.NewUnstartedServer(nil)
	origin := "http://" + server.Listener.Addr().String()
	serverID := resource.ID()
	ids := identity.NewService(s.pool, nil, origin)
	devices := endpoint.NewExecution(s.pool, ids)
	_, signingKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(signingKey)
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(t.TempDir(), "relay.pem")
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := relay.NewService(s.pool, ids, devices, slog.Default(), relay.Config{
		ServerID: serverID, PublicURL: origin, FilesOrigin: origin, SigningKeyFile: keyFile,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Shutdown(context.Background()); server.Close() })
	ids.WithDeviceAuthorization(serverID, r.DeviceAuthorizationKey(), devices).
		WithRuntimeMetadata(r.Metadata()).WithRuntimeAccess(s)
	s.WithRuntime(origin, serverID, ids).WithReadiness(r)
	if _, err := s.pool.Exec(ctx, `UPDATE hosts SET report=report || '{"runtime_protocol":1,"runtime_update_protocol":3}'::jsonb WHERE id=$1`, host); err != nil {
		t.Fatal(err)
	}
	env, err := s.Create(ctx, user, Creation{Name: "Cloud source fixture", RequestKey: "cloud-source-1"})
	if err != nil {
		t.Fatal(err)
	}

	const toolResult = "cloud workspace fixture 3d6ff0"
	if err := os.WriteFile("/workspace/cloud-fixture.txt", []byte(toolResult), 0600); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	r.Register(router)
	router.Get("/.well-known/oauth-authorization-server", ids.OAuthMetadata)
	router.Mount("/oauth", ids.OAuthRouter())
	model := resource.ID()
	if _, err := s.pool.Exec(ctx, `INSERT INTO models(id,ownership_type,owner_user_id,display_name,model_id,protocol,base_url,api_key,advanced_config)
		VALUES($1,'system',$2,'Cloud fixture','model-fixture','openai_chat_completions',$3,'fixture','{}')`, model, user, origin); err != nil {
		t.Fatal(err)
	}
	keys := apikey.NewService(apikey.NewPostgres(s.pool))
	sessions := session.NewService(s.pool).WithHostAuthorizer(devices)
	var permitReports atomic.Bool
	router.Route("/api/v1", func(routes chi.Router) {
		routes.Use(ids.RequireAgent)
		ids.RegisterAgent(routes)
		keys.RegisterAgent(routes)
		sessions.RegisterAgent(routes.With(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost && !permitReports.Load() {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				next.ServeHTTP(w, r)
			})
		}))
		routes.Get("/models", func(w http.ResponseWriter, req *http.Request) {
			resource.JSON(req.Context(), w, 200, map[string]any{"total_count": 1, "models": []map[string]string{{"id": model, "kind": "text", "protocol": "openai_chat_completions"}}})
		})
	})
	var toolObserved atomic.Bool
	modelWaiting := make(chan struct{})
	router.Post("/v1/chat/completions", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Messages []struct{ Role, Content string }
		}
		rawKey, ok := strings.CutPrefix(req.Header.Get("Authorization"), "Bearer ")
		owner, keyErr := keys.Authenticate(req.Context(), rawKey, apikey.ScopeModelInvoke)
		if !ok || keyErr != nil || owner != user || json.NewDecoder(req.Body).Decode(&body) != nil || len(body.Messages) == 0 {
			w.WriteHeader(400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		last := body.Messages[len(body.Messages)-1]
		if last.Role == "user" && last.Content == "Pause the model for cancellation." {
			close(modelWaiting)
			<-req.Context().Done()
			return
		}
		if last.Role != "tool" {
			fmt.Fprint(w, `{"choices":[{"finish_reason":"tool_calls","message":{"tool_calls":[{"id":"fixture-call","type":"function","function":{"name":"read_text_file","arguments":"{\"path\":\"cloud-fixture.txt\"}"}}]}}]}`)
		} else if last.Content == toolResult {
			toolObserved.Store(true)
			fmt.Fprint(w, `{"choices":[{"finish_reason":"stop","message":{"content":"fixture confirmed"}}],"usage":{"prompt_tokens":5,"completion_tokens":2}}`)
		} else {
			t.Errorf("remote workspace returned unexpected tool data: %q", last.Content)
			w.WriteHeader(400)
		}
	})
	server.Config.Handler = router
	server.Start()

	state := t.TempDir()
	archiveID := resource.ID()
	archive := filepath.Join(state, "runtime", "sessions", archiveID)
	if err := os.MkdirAll(archive, 0700); err != nil {
		t.Fatal(err)
	}
	replay := []byte(`{"version":11,"from_seq":1,"to_seq":1,"message":{"id":"legacy-message","role":"assistant","parts":[{"type":"text","text":"preserved cloud message"},{"type":"file","name":"sample.txt","uri":"fixture"}]},"toolCalls":{"legacy-tool":{"output":"preserved cloud tool output"}}}` + "\n")
	meta, err := json.Marshal(map[string]any{"id": archiveID, "status": "idle", "schema_version": 8, "replay_version": 11,
		"replay_bytes": len(replay), "message_count": 1, "replay_applied_seq": 1, "last_event_seq": 1, "view_seq": 1, "replay_generation": "fixture"})
	if err != nil {
		t.Fatal(err)
	}
	for name, payload := range map[string][]byte{"meta.json": meta, "replay.jsonl": replay} {
		if err := os.WriteFile(filepath.Join(archive, name), payload, 0600); err != nil {
			t.Fatal(err)
		}
	}
	command := func(action string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, binary, append([]string{action, "--state-dir", state}, args...)...)
	}
	status := func() (result struct {
		MachineID, PublicKey string
		Authorized           bool
	}) {
		t.Helper()
		raw, err := command("status").CombinedOutput()
		if err != nil || json.Unmarshal(raw, &result) != nil {
			t.Fatalf("runtime status: %v %s", err, raw)
		}
		return result
	}
	authorize := func(target View) RuntimeRequest {
		t.Helper()
		login := command("login", "--device-code", "--server", origin, "--server-id", serverID, "--user", user)
		var loginErrors bytes.Buffer
		login.Stderr = &loginErrors
		stdout, err := login.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := login.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = login.Process.Kill() })
		var approval struct{ Event, UserCode string }
		if err := json.NewDecoder(bufio.NewReader(stdout)).Decode(&approval); err != nil || approval.Event != "authorize" || approval.UserCode == "" {
			_ = login.Wait()
			t.Fatalf("runtime login did not request approval: %v %s", err, loginErrors.String())
		}
		before := status()
		if before.Authorized {
			t.Fatal("runtime was authorized without backend approval")
		}
		approve := RuntimeRequest{EnvironmentID: target.ID, InstanceID: target.InstanceID,
			Generation: target.Generation, MachineID: before.MachineID, PublicKey: before.PublicKey, UserCode: approval.UserCode}
		if err := s.approveRuntime(ctx, host, approve); err != nil {
			t.Fatal(err)
		}
		if err := login.Wait(); err != nil {
			t.Fatalf("finish device grant: %v %s", err, loginErrors.String())
		}
		if after := status(); !after.Authorized || after.MachineID != before.MachineID || after.PublicKey != before.PublicKey {
			t.Fatal("persisted device identity does not match approval")
		}
		return approve
	}
	initial := authorize(env.Environment)
	before := status()

	var grant string
	if err := s.pool.QueryRow(ctx, "INSERT INTO oauth_grants(user_id,client_id,scopes,expires_at) VALUES($1,'monkeyai-desktop',$2,now()+interval '1 day') RETURNING id",
		user, []string{"device:connect", "session:read", "session:control", "files:read", "files:write"}).Scan(&grant); err != nil {
		t.Fatal(err)
	}
	bearer := resource.ID()
	tokenHash := sha256.Sum256([]byte(bearer))
	refreshHash := sha256.Sum256([]byte(bearer + "refresh"))
	if _, err := s.pool.Exec(ctx, `INSERT INTO oauth_tokens(user_id,client_id,grant_id,access_token_hash,refresh_token_hash,access_expires_at,refresh_expires_at)
		VALUES($1,'monkeyai-desktop',$2,$3,$4,now()+interval '1 hour',now()+interval '1 day')`,
		user, grant, hex.EncodeToString(tokenHash[:]), hex.EncodeToString(refreshHash[:])); err != nil {
		t.Fatal(err)
	}

	serve := func() func() {
		t.Helper()
		cmd := command("serve")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = cmd.Process.Kill() })
		return func() {
			t.Helper()
			_ = cmd.Process.Signal(syscall.SIGTERM)
			if err := cmd.Wait(); err != nil {
				t.Fatalf("runtime shutdown: %v", err)
			}
		}
	}
	connect := func() (*websocket.Conn, string) {
		t.Helper()
		var ticket relay.ConnectionResponse
		var detail string
		for range 25 {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, origin+"/api/v1/endpoints/"+before.MachineID+"/connections",
				strings.NewReader(`{"scopes":["session:read","session:control","files:read","files:write"],"supported_protocols":{"acp":[1],"runtime_api":[1]}}`))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer "+bearer)
			response, err := server.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			data, _ := io.ReadAll(response.Body)
			response.Body.Close()
			if response.StatusCode == 200 && json.Unmarshal(data, &ticket) == nil && ticket.Ticket != "" {
				break
			}
			detail = string(data)
			time.Sleep(250 * time.Millisecond)
		}
		if ticket.Ticket == "" {
			t.Fatalf("runtime did not become reachable: %s", detail)
		}
		ws, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(ticket.HTTPBase, "http")+"acp",
			&websocket.DialOptions{Subprotocols: []string{"monkeyai.v1", "monkeyai.token." + ticket.Ticket}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { ws.CloseNow() })
		cloudACP(t, ctx, ws, 1, "initialize", map[string]int{"protocolVersion": 1}, nil)
		return ws, ticket.Ticket
	}
	t.Log("device approval and sealed identity survived a separate process")
	stop := serve()
	ws, _ := connect()
	syncState(t, s, host, observe(env.Environment, "running"))
	var created struct{ SessionID string }
	cloudACP(t, ctx, ws, 2, "session/new", map[string]any{"_meta": map[string]string{"operationId": resource.ID()}}, &created)
	prompt := map[string]any{"sessionId": created.SessionID, "_meta": map[string]string{"operationId": resource.ID()},
		"prompt": []map[string]string{{"type": "text", "text": "Read cloud-fixture.txt once."}}}
	answer := cloudACP(t, ctx, ws, 3, "session/prompt", prompt, nil)
	if !toolObserved.Load() || answer != "fixture confirmed" {
		t.Fatalf("remote tool did not complete: %q", answer)
	}
	ws.CloseNow()
	t.Log("remote prompt requested permission and returned the real workspace file")
	stop()
	stopped, err := s.Change(ctx, &user, env.Environment.ID, "stopped", Change{Generation: env.Environment.Generation, RequestKey: "cloud-stop"})
	if err != nil {
		t.Fatal(err)
	}
	syncState(t, s, host, observe(stopped.Environment, "stopped"))
	// Represent an exported device key and the untouched state of an already
	// bound environment. Move the previous executor's fixture credentials
	// aside to require an actual, independent authorization after migration.
	backup := t.TempDir()
	const identityFile = "aibertai-cloud-identity.json"
	for _, name := range []string{identityFile, "source-credentials.enc.json"} {
		if err := os.Rename(filepath.Join(state, name), filepath.Join(backup, name)); err != nil {
			t.Fatal(err)
		}
	}
	var oldIdentity struct{ DataKey string }
	raw, err := os.ReadFile(filepath.Join(backup, identityFile))
	if err != nil || json.Unmarshal(raw, &oldIdentity) != nil {
		t.Fatalf("fixture identity could not be loaded: %v", err)
	}
	seed, err := base64.StdEncoding.DecodeString(oldIdentity.DataKey)
	if err != nil {
		t.Fatal(err)
	}
	export := filepath.Join(backup, "fixture-original-ed25519.seed")
	if err := os.WriteFile(export, seed, 0600); err != nil {
		t.Fatal(err)
	}
	const version = "26.1105.2"
	s.WithSourceRuntimeProvisioning().WithRuntimeCatalog(aiBertSourceCatalog{officialRuntimeCatalog{runtimeCatalogFixture{latest: version}}})
	raw, err = command("prepare-cloud-migration", "--server", origin, "--server-id", serverID, "--user", user,
		"--machine", before.MachineID, "--public-key", before.PublicKey, "--key-file", export,
		"--environment", env.Environment.ID, "--generation", fmt.Sprint(stopped.Environment.Generation), "--version", version).CombinedOutput()
	var input RuntimeUpdate
	if err != nil || json.Unmarshal(raw, &input) != nil || input.IdentityProof == "" {
		t.Fatalf("prepare migration: %v", err)
	}
	invalid := input
	invalid.Version = "26.1105.1"
	if _, err := s.UpdateRuntime(ctx, user, env.Environment.ID, invalid); err == nil {
		t.Fatal("migration proof could be reused for another release")
	}
	upgrade, err := s.UpdateRuntime(ctx, user, env.Environment.ID, input)
	if err != nil || upgrade.Environment.InstanceID != stopped.Environment.InstanceID {
		t.Fatalf("in-place migration rejected: %v", err)
	}
	if retried, err := s.UpdateRuntime(ctx, user, env.Environment.ID, input); err != nil || retried.Operation.ID != upgrade.Operation.ID {
		t.Fatalf("migration request retry changed operation: %v", err)
	}
	installed := observe(upgrade.Environment, "stopped")
	installed.RuntimeVersion = version
	syncState(t, s, host, installed)
	started, err := s.Change(ctx, &user, env.Environment.ID, "running", Change{Generation: upgrade.Environment.Generation, RequestKey: "cloud-restart"})
	if err != nil {
		t.Fatal(err)
	}
	rebound := authorize(started.Environment)
	if rebound.MachineID != initial.MachineID || rebound.PublicKey != initial.PublicKey {
		t.Fatal("migration changed the original machine identity")
	}
	t.Log("migration preserved the registered public key and device ID after reauthorization")
	permitReports.Store(true)
	stop = serve()
	ws, ticket := connect()
	restored := cloudACP(t, ctx, ws, 2, "session/load", map[string]string{"sessionId": created.SessionID, "cwd": "/workspace"}, nil)
	if restored != answer {
		t.Fatalf("restored transcript differs: %q", restored)
	}
	for id, expected := range map[string][]string{created.SessionID: {toolResult}, archiveID: {"preserved cloud message", "sample.txt", "preserved cloud tool output"}} {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, origin+"/api/v1/endpoints/"+before.MachineID+"/proxy/read/session",
			strings.NewReader(fmt.Sprintf(`{"method":"_monkeyai/session/messages","params":{"sessionId":%q,"limit":20}}`, id)))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+ticket)
		req.Header.Set("Content-Type", "application/json")
		response, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		history, _ := io.ReadAll(response.Body)
		response.Body.Close()
		for _, want := range expected {
			if response.StatusCode != 200 || !bytes.Contains(history, []byte(want)) {
				t.Fatalf("restored history missing %q: HTTP %d %s", want, response.StatusCode, history)
			}
		}
	}
	if actual, err := os.ReadFile(filepath.Join(archive, "replay.jsonl")); err != nil || !bytes.Equal(actual, replay) {
		t.Fatalf("legacy data changed: %v", err)
	}
	if contents, err := os.ReadFile("/workspace/cloud-fixture.txt"); err != nil || string(contents) != toolResult {
		t.Fatalf("migration changed the original workspace: %v", err)
	}
	var reported bool
	for range 30 {
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM session_turn_tools WHERE session_id=$1 AND name='read_text_file')`, created.SessionID).Scan(&reported); err != nil {
			t.Fatal(err)
		}
		if reported {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !reported {
		t.Fatal("unconfirmed remote tool execution was not reported after restart")
	}
	var preview struct{ Content string }
	cloudACP(t, ctx, ws, 30, "_monkeyai/session/files/preview", map[string]string{
		"sessionId": created.SessionID, "path": "cloud-fixture.txt",
	}, &preview)
	if preview.Content != toolResult {
		t.Fatal("cloud session file preview did not return the authorized workspace file")
	}
	toolObserved.Store(false)
	prompt["_meta"] = map[string]string{"operationId": resource.ID()}
	prompt["prompt"] = []map[string]string{{"type": "text", "text": "Read the same file after migration."}}
	if repeated := cloudACP(t, ctx, ws, 3, "session/prompt", prompt, nil); repeated != answer || !toolObserved.Load() {
		t.Fatal("migrated session did not execute a new remote read in the existing workspace")
	}
	readState, err := http.NewRequestWithContext(ctx, http.MethodPost, origin+"/api/v1/endpoints/"+before.MachineID+"/proxy/read/session",
		strings.NewReader(fmt.Sprintf(`{"method":"_monkeyai/session/state/get","params":{"sessionId":%q}}`, created.SessionID)))
	if err != nil {
		t.Fatal(err)
	}
	readState.Header.Set("Authorization", "Bearer "+ticket)
	stateResponse, err := server.Client().Do(readState)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		Result struct {
			SessionID string
			State     struct {
				Files, Tasks, Plan, ResourceIDs []json.RawMessage
				ResourcePreferences             map[string]json.RawMessage
			}
		}
	}
	decodeErr := json.NewDecoder(stateResponse.Body).Decode(&snapshot)
	stateResponse.Body.Close()
	stateSnapshot := snapshot.Result.State
	if stateResponse.StatusCode != 200 || decodeErr != nil || snapshot.Result.SessionID != created.SessionID ||
		stateSnapshot.Files == nil || stateSnapshot.Tasks == nil || stateSnapshot.Plan == nil ||
		stateSnapshot.ResourceIDs == nil || stateSnapshot.ResourcePreferences == nil {
		t.Fatalf("remote session state did not match the renderer contract: HTTP %d %v", stateResponse.StatusCode, decodeErr)
	}
	prompt["_meta"] = map[string]string{"operationId": resource.ID()}
	prompt["prompt"] = []map[string]string{{"type": "text", "text": "Pause the model for cancellation."}}
	if err := wsjson.Write(ctx, ws, map[string]any{"jsonrpc": "2.0", "id": 31, "method": "session/prompt", "params": prompt}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-modelWaiting:
	case <-time.After(10 * time.Second):
		t.Fatal("test model never received the pending request")
	}
	cancelRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, origin+"/api/v1/endpoints/"+before.MachineID+"/proxy/write/session/cancel",
		strings.NewReader(fmt.Sprintf(`{"sessionId":%q}`, created.SessionID)))
	if err != nil {
		t.Fatal(err)
	}
	cancelRequest.Header.Set("Authorization", "Bearer "+ticket)
	cancelResponse, err := server.Client().Do(cancelRequest)
	if err != nil {
		t.Fatal(err)
	}
	cancelBody, _ := io.ReadAll(cancelResponse.Body)
	cancelResponse.Body.Close()
	if cancelResponse.StatusCode != 200 || !bytes.Contains(cancelBody, []byte(`"stopped":true`)) {
		t.Fatalf("remote cancellation was blocked by the active prompt: HTTP %d %s", cancelResponse.StatusCode, cancelBody)
	}
	ws.CloseNow()
	stop()
	t.Log("recovered the conversation and pending report and executed another tool after migration, with the legacy archive unchanged")
}

func cloudACP(t *testing.T, ctx context.Context, ws *websocket.Conn, id int, method string, params any, result any) string {
	t.Helper()
	request := map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}
	if err := wsjson.Write(ctx, ws, request); err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	approved := false
	for {
		var reply struct {
			ID, Result, Error json.RawMessage
			Method            string
			Params            struct {
				Update struct {
					SessionUpdate string
					Content       struct{ Text string }
				}
			}
		}
		if err := wsjson.Read(ctx, ws, &reply); err != nil {
			t.Fatalf("%s: %v", method, err)
		}
		if reply.Method == "session/request_permission" {
			if method != "session/prompt" || approved {
				t.Fatalf("unexpected tool permission request for %s", method)
			}
			approved = true
			permission := map[string]any{"jsonrpc": "2.0", "id": reply.ID,
				"result": map[string]any{"outcome": map[string]string{"outcome": "selected", "optionId": "allow"}}}
			if err := wsjson.Write(ctx, ws, permission); err != nil {
				t.Fatal(err)
			}
		}
		if reply.Method == "session/update" && reply.Params.Update.SessionUpdate == "agent_message_chunk" {
			text.WriteString(reply.Params.Update.Content.Text)
		}
		if string(reply.ID) != fmt.Sprint(id) {
			continue
		}
		if len(reply.Error) != 0 || (result != nil && json.Unmarshal(reply.Result, result) != nil) {
			t.Fatalf("%s: %s", method, reply.Error)
		}
		if method == "session/prompt" && !approved {
			t.Fatal("remote tool executed without the approval step")
		}
		return text.String()
	}
}
