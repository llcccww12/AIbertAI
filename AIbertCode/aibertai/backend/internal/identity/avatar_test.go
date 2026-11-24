package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
)

type avatarStorageStub struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func (s *avatarStorageStub) Put(_ context.Context, key string, data []byte, _ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = data
	return nil
}

func (s *avatarStorageStub) Get(_ context.Context, key string) (io.ReadCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.objects[key]
	if !ok {
		return nil, errors.New("missing")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (s *avatarStorageStub) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, key)
	return nil
}

func TestAgentAvatarUploadAndClear(t *testing.T) {
	pool := emailDatabase(t)
	storage := &avatarStorageStub{objects: map[string][]byte{}}
	s := newTestService(pool, nil, "http://localhost").WithProfileStorage(storage)
	var user User
	if err := pool.QueryRow(t.Context(), `INSERT INTO users(name,email,avatar_url) VALUES('员工','staff@example.test','https://sso.example/a.png') RETURNING id,name,email`).Scan(&user.ID, &user.Name, &user.Email); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey{}, user)))
		})
	})
	s.RegisterAgent(router)
	call := func(method string, body []byte, want int) avatarResult {
		t.Helper()
		req := httptest.NewRequest(method, "/me/avatar", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s: status=%d want=%d body=%s", method, rec.Code, want, rec.Body.String())
		}
		var out avatarResult
		if want == http.StatusOK {
			if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}

	call(http.MethodPut, []byte("not an image"), http.StatusBadRequest)

	var img bytes.Buffer
	if err := png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	uploaded := call(http.MethodPut, img.Bytes(), http.StatusOK)
	if !strings.HasPrefix(uploaded.CustomAvatarURL, "http://localhost/api/auth/v1/avatars/"+user.ID+"/") || uploaded.User.AvatarURL != uploaded.CustomAvatarURL {
		t.Fatalf("upload result = %+v", uploaded)
	}
	if len(storage.objects) != 1 {
		t.Fatalf("stored objects = %d", len(storage.objects))
	}

	cleared := call(http.MethodDelete, nil, http.StatusOK)
	if cleared.CustomAvatarURL != "" || cleared.User.AvatarURL != "https://sso.example/a.png" {
		t.Fatalf("clear result = %+v", cleared)
	}
	if len(storage.objects) != 0 {
		t.Fatalf("old avatar object not removed: %d", len(storage.objects))
	}
}

type avatarResult struct {
	User            User   `json:"user"`
	CustomAvatarURL string `json:"custom_avatar_url"`
}
