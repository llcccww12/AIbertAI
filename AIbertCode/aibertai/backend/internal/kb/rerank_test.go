package kb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"aibertai.local/backend/internal/resource"
	"aibertai.local/backend/internal/setting"
)

func TestKnowledgeSearchUsesConfiguredReranker(t *testing.T) {
	pool, actor := kbTestPool(t)
	ctx := t.Context()
	id := resource.ID()
	if _, err := pool.Exec(ctx, `INSERT INTO knowledge_bases(id,name,owner_user_id,created_by_user_id,ownership_type)
		VALUES($1,'Reranking',$2,$2,'user')`, id, actor); err != nil {
		t.Fatal(err)
	}
	settings := setting.NewService(setting.NewPostgres(pool))
	service := NewService(resource.NewStore(pool), nil).WithReranker(settings.Reranker)
	for _, body := range []string{"apple orchard", "pear orchard"} {
		if _, err := service.CreateDocument(ctx, actor, DocumentInput{KbID: id, Title: body, Content: body}); err != nil {
			t.Fatal(err)
		}
		if worked, err := service.runLearningJob(ctx); !worked || err != nil {
			t.Fatalf("learning: worked=%v, err=%v", worked, err)
		}
	}
	query := SearchInput{Query: "orchard", KbIDs: []string{id}, TopK: 2}
	original, err := service.SearchWithResponse(ctx, actor, query)
	if err != nil || len(original.Items) != 2 {
		t.Fatalf("initial search: %+v, %v", original, err)
	}
	target := original.Items[1].String("content")
	var invalid atomic.Bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var data struct {
			Model     string   `json:"model"`
			Query     string   `json:"query"`
			Documents []string `json:"documents"`
			TopN      int      `json:"top_n"`
		}
		if req.URL.Path != "/v2/rerank" || req.Header.Get("Authorization") != "Bearer rerank-test-token" ||
			json.NewDecoder(req.Body).Decode(&data) != nil || data.Model != "custom-reranker" ||
			data.Query != query.Query || len(data.Documents) != 2 || data.TopN != 2 {
			http.Error(w, "unexpected rerank request", http.StatusBadRequest)
			return
		}
		index := slices.Index(data.Documents, target)
		if invalid.Load() {
			index = len(data.Documents)
		}
		fmt.Fprintf(w, `{"results":[{"index":%d,"relevance_score":0.97}]}`, index)
	}))
	defer upstream.Close()
	config := json.RawMessage(fmt.Sprintf(`{"base_url":%q,"model_id":"custom-reranker","api_key":"rerank-test-token"}`, upstream.URL+"/v2/rerank"))
	if _, err := settings.Put(ctx, setting.RerankerKey, config, 1, actor); err != nil {
		t.Fatal(err)
	}
	read, err := settings.AdminGet(ctx, setting.RerankerKey)
	if err != nil || strings.Contains(string(read.Value), "rerank-test-token") {
		t.Fatalf("reranker secret was returned: %v", err)
	}
	reordered, err := service.SearchWithResponse(ctx, actor, query)
	if err != nil || len(reordered.Items) != 2 || reordered.Items[0].String("content") != target ||
		reordered.Items[0]["rerank_score"] != 0.97 || reordered.Degraded {
		t.Fatalf("reranked search: %+v, %v", reordered, err)
	}
	invalid.Store(true)
	fallback, err := service.SearchWithResponse(ctx, actor, query)
	if err != nil || len(fallback.Items) != 2 || fallback.Items[0].String("chunk_id") != original.Items[0].String("chunk_id") ||
		!fallback.Degraded || !slices.Contains(fallback.Warnings, "reranker_failed") {
		t.Fatalf("reranker fallback: %+v, %v", fallback, err)
	}
}

func TestConfiguredEmbeddingDefaultRespectsExplicitBinding(t *testing.T) {
	pool, actor := kbTestPool(t)
	ctx := t.Context()
	service := NewService(resource.NewStore(pool), nil)
	settings := setting.NewService(setting.NewPostgres(pool)).WithEmbeddingDefaultCheck(service.ValidateEmbeddingDefault)
	first, preferred := resource.ID(), resource.ID()
	seedEmbeddingModel(t, pool, actor, first)
	seedEmbeddingModel(t, pool, actor, preferred)
	value := json.RawMessage(fmt.Sprintf(`{"model_id":%q}`, preferred))
	if _, err := settings.Put(ctx, setting.EmbeddingDefaultKey, value, 1, actor); err != nil {
		t.Fatal(err)
	}
	for requested, want := range map[string]string{"": preferred, first: first} {
		result, dimension, err := service.resolveEmbeddingModel(ctx, pool, requested)
		if err != nil || result != want || dimension != 3 {
			t.Fatalf("embedding selection %q: got %q (%d), want %q, %v", requested, result, dimension, want, err)
		}
	}
	unknown := json.RawMessage(fmt.Sprintf(`{"model_id":%q}`, resource.ID()))
	if _, err := settings.Put(ctx, setting.EmbeddingDefaultKey, unknown, 1, actor); err == nil {
		t.Fatal("expected rejection of unknown model")
	}
}
