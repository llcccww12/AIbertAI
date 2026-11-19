package kb

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"aibertai.local/backend/internal/identity"
	"aibertai.local/backend/internal/resource"
	"aibertai.local/backend/internal/setting"
	"github.com/go-chi/chi/v5"
)

func TestAdminDocumentVisionImport(t *testing.T) {
	pool, actor := kbTestPool(t)
	const token = "vision-import-admin-session"
	const apiKey = "vision-test-key"
	digest := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(t.Context(), `INSERT INTO browser_sessions(token_hash,user_id,authentication_method,expires_at)
		VALUES($1,$2,'oauth',now()+interval '1 hour')`, hex.EncodeToString(digest[:]), actor); err != nil {
		t.Fatal(err)
	}
	kbID := resource.ID()
	if _, err := pool.Exec(t.Context(), `INSERT INTO knowledge_bases(id,name,owner_user_id,created_by_user_id,ownership_type)
		VALUES($1,'Document vision',$2,$2,'system')`, kbID, actor); err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Model    string `json:"model"`
			Messages []struct {
				Content []struct {
					ImageURL struct{ URL string } `json:"image_url"`
				} `json:"content"`
			} `json:"messages"`
		}
		if r.URL.Path != "/v1/chat/completions" ||
			r.Header.Get("Authorization") != "Bearer "+apiKey ||
			json.NewDecoder(r.Body).Decode(&input) != nil ||
			input.Model != "vision-test" || len(input.Messages) != 1 || len(input.Messages[0].Content) != 2 {
			http.Error(w, "unexpected vision request", http.StatusBadRequest)
			return
		}
		var text string
		switch imageURL := input.Messages[0].Content[1].ImageURL.URL; {
		case strings.HasPrefix(imageURL, "data:image/png;base64,"):
			text = "Extracted PNG text"
		case strings.HasPrefix(imageURL, "data:image/jpeg;base64,"):
			text = "Extracted PDF page"
		default:
			http.Error(w, "missing document image", http.StatusBadRequest)
			return
		}
		_, _ = fmt.Fprintf(w, `{"choices":[{"finish_reason":"stop","message":{"content":%q}}]}`, text)
	}))
	t.Cleanup(upstream.Close)

	settings := setting.NewService(setting.NewPostgres(pool))
	service := NewService(resource.NewStore(pool), nil).WithDocumentVision(settings.DocumentVision)
	router := chi.NewRouter()
	router.Use(identity.NewService(pool, nil, "http://example.test").RequireAdmin)
	settings.RegisterAdmin(router)
	service.RegisterAdmin(router)
	request := func(method, path, contentType string, body io.Reader) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, path, body)
		r.Header.Set("Content-Type", contentType)
		r.AddCookie(&http.Cookie{Name: "monkeyai_session", Value: token})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	upload := func(name string, data []byte) *httptest.ResponseRecorder {
		t.Helper()
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		part, err := form.CreateFormFile("file", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = part.Write(data); err != nil {
			t.Fatal(err)
		}
		if err = form.Close(); err != nil {
			t.Fatal(err)
		}
		return request(http.MethodPost, "/kb/"+kbID+"/documents", form.FormDataContentType(), &body)
	}
	png := avatarImage(t, "png")
	if w := upload("test.png", png); w.Code != http.StatusBadRequest {
		t.Fatalf("expected vision configuration prompt, got %d %s", w.Code, w.Body)
	}
	settingPath := "/settings/document_vision"
	save := func(base, key string) *httptest.ResponseRecorder {
		body := fmt.Sprintf(`{"schema_version":1,"value":{"base_url":%q,"model_id":"vision-test","api_key":%q}}`, base, key)
		return request(http.MethodPut, settingPath, "application/json", strings.NewReader(body))
	}
	for _, w := range []*httptest.ResponseRecorder{
		save(upstream.URL+"/v1/chat/completions/", apiKey),
		save(upstream.URL+"/v1", ""),
		request(http.MethodGet, settingPath, "", nil),
		request(http.MethodGet, "/settings", "", nil),
	} {
		if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "api_key") ||
			strings.Contains(w.Body.String(), apiKey) || !strings.Contains(w.Body.String(), "vision-test") {
			t.Fatalf("expected saved model without key, got %d %s", w.Code, w.Body)
		}
	}
	if w := save("https://other-model.example/v1", ""); w.Code != http.StatusBadRequest {
		t.Fatalf("changing providers without a new key should fail: %d %s", w.Code, w.Body)
	}
	agent, err := settings.AgentConfig(t.Context())
	if err != nil || agent.Settings[setting.DocumentVisionKey] != nil {
		t.Fatalf("document vision settings appeared in agent configuration: %v", err)
	}
	checkImport := func(t *testing.T, name, text string, data []byte) {
		t.Helper()
		w := upload(name, data)
		if w.Code != http.StatusAccepted {
			t.Fatalf("import %s: %d %s", name, w.Code, w.Body)
		}
		var result resource.Object
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		doc, err := service.GetDocument(t.Context(), result.String("id"))
		if err != nil || doc.String("content") != text || doc.String("file_name") != name {
			t.Fatalf("unexpected stored document: %v, %v", doc, err)
		}
	}
	checkImport(t, "test.png", "Extracted PNG text", png)
	t.Run("multi-page PDF", func(t *testing.T) {
		if _, err := exec.LookPath("pdftoppm"); err != nil {
			t.Skip("Poppler is required for PDF import")
		}
		checkImport(t, "test.pdf", "Extracted PDF page\n\nExtracted PDF page", twoPagePDF())
	})
}

func twoPagePDF() []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << >> >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 100 100] /Resources << >> >>",
	}
	var pdf strings.Builder
	pdf.WriteString("%PDF-1.4\n")
	offsets := make([]int, 0, len(objects))
	for index, object := range objects {
		offsets = append(offsets, pdf.Len())
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return []byte(pdf.String())
}
