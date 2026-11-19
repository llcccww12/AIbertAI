package agentconfig

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"aibertai.local/backend/internal/apikey"
	"aibertai.local/backend/internal/expert"
	"aibertai.local/backend/internal/identity"
	"aibertai.local/backend/internal/mcp"
	"aibertai.local/backend/internal/resource"
	"aibertai.local/backend/internal/web"
	"github.com/go-chi/chi/v5"
)

type importTestBilling struct{}

func (importTestBilling) Begin(context.Context, mcp.Invocation) (string, error) {
	return resource.ID(), nil
}
func (importTestBilling) Start(context.Context, string) error                        { return nil }
func (importTestBilling) Finish(context.Context, string, mcp.InvocationResult) error { return nil }

func TestDesktopLegacyImport(t *testing.T) {
	binary := os.Getenv("AIBERTAI_NATIVE_IMPORT_TEST_BINARY")
	if binary == "" {
		t.Skip("需要编译桌面端测试二进制")
	}
	resources, user, _ := catalogDatabase(t)
	t.Setenv("MONKEYAI_MCP_ALLOWED_CIDRS", "127.0.0.0/8")
	var calls, sequence atomic.Int32
	var sessions sync.Map
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-legacy-header" {
			http.Error(w, "missing fixture credential", 401)
			return
		}
		if r.URL.Path == "/" {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			id := fmt.Sprint(sequence.Add(1))
			out := make(chan []byte, 8)
			sessions.Store(id, out)
			defer sessions.Delete(id)
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "event: endpoint\ndata: /rpc?sid=%s\n\n", id)
			flusher := w.(http.Flusher)
			flusher.Flush()
			for {
				select {
				case data := <-out:
					fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
					flusher.Flush()
				case <-r.Context().Done():
					return
				}
			}
		}
		stream, exists := sessions.Load(r.URL.Query().Get("sid"))
		if r.Method != http.MethodPost || !exists {
			http.NotFound(w, r)
			return
		}
		var request struct {
			ID     json.RawMessage
			Method string
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "bad request", 400)
			return
		}
		result := resource.Object{}
		switch request.Method {
		case "initialize":
			result["protocolVersion"] = "2024-11-05"
		case "notifications/initialized":
			w.WriteHeader(202)
			return
		case "tools/list":
			result["tools"] = []resource.Object{{"name": "check_status", "inputSchema": resource.Object{"type": "object"}}}
		case "tools/call":
			calls.Add(1)
			result["content"] = []resource.Object{{"type": "text", "text": "LEGACY_MCP_OK"}}
		default:
			http.Error(w, "unknown request", 400)
			return
		}
		data, _ := json.Marshal(resource.Object{"jsonrpc": "2.0", "id": request.ID, "result": result})
		stream.(chan []byte) <- data
		w.WriteHeader(http.StatusAccepted)
	}))
	defer upstream.Close()

	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte("desktop-import-test")))
	catalogExec(t, resources, `INSERT INTO oauth_tokens(user_id,client_id,access_token_hash,refresh_token_hash,access_expires_at,refresh_expires_at) VALUES($1,'test',$2,$2,now()+interval '1 hour',now()+interval '2 hours')`, user, tokenHash)
	router := chi.NewRouter()
	router.Use(web.Middleware)
	keys := apikey.NewService(apikey.NewPostgres(resources.store.Pool))
	router.Route("/api/v1", func(r chi.Router) {
		r.Use(identity.NewService(resources.store.Pool, nil, "http://localhost").RequireAgent)
		keys.RegisterAgent(r)
		resources.mcp.RegisterAgent(r)
		expert.NewService(resources.store).RegisterAgent(r)
		resources.RegisterAgent(r)
	})
	resources.mcp.RegisterGateway(router, keys, importTestBilling{})
	srv := httptest.NewServer(router)
	defer srv.Close()
	cmd := exec.CommandContext(t.Context(), binary, "-test.run=^TestDesktopLegacyImportIntegration$", "-test.v")
	cmd.Env = append(os.Environ(), "AIBERTAI_IMPORT_TEST_BACKEND="+srv.URL, "AIBERTAI_IMPORT_TEST_MCP="+upstream.URL)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("桌面运行时导入验收失败: %v\n%s", err, output)
	} else {
		t.Log(strings.TrimSpace(string(output)))
	}
	if calls.Load() != 1 {
		t.Fatalf("MCP 上游实际调用次数 = %d", calls.Load())
	}
	for _, table := range []string{"experts", "connectors"} {
		var n int
		err := resources.store.Pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE owner_user_id=$1", user).Scan(&n)
		if err != nil || n != 1 {
			t.Fatalf("%s 数据库记录数 = %d: %v", table, n, err)
		}
	}
	var credentials int
	if err := resources.store.Pool.QueryRow(t.Context(), "SELECT count(*) FROM connector_credentials WHERE user_id=$1", user).Scan(&credentials); err != nil || credentials != 1 {
		t.Fatalf("个人连接凭证记录数 = %d: %v", credentials, err)
	}
}
