package kb

import (
	"context"
	"errors"
	"sort"
	"time"

	"aibertai.local/backend/internal/resource"
)

// Only candidates from authorized search results reach the reranker.
func (s *Service) rerankCandidates(ctx context.Context, actor, query string, items []resource.Object) ([]resource.Object, error) {
	if s.reranker == nil || len(items) < 2 {
		return items, nil
	}
	config, err := s.reranker(ctx)
	if err != nil || config.APIKey == "" || config.BaseURL == "" || config.ModelID == "" {
		return items, err
	}
	count := min(defaultSearchCandidate, len(items))
	if err := s.checkSearchPermissions(ctx, actor, items[:count]); err != nil {
		return items, err
	}
	documents := make([]string, count)
	for i, item := range items[:count] {
		documents[i] = item.String("content")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var response struct {
		Results []struct {
			Index *int     `json:"index"`
			Score *float64 `json:"relevance_score"`
		} `json:"results"`
	}
	err = callKnowledgeModel(ctx, config, "rerank", map[string]any{
		"model":     config.ModelID,
		"query":     query,
		"documents": documents,
		"top_n":     count,
	}, &response)
	if err != nil {
		return items, err
	}

	scored := make(map[int]float64, len(response.Results))
	if len(response.Results) == 0 || len(response.Results) > count {
		return items, errors.New("invalid reranker results")
	}
	for _, result := range response.Results {
		if result.Index == nil || result.Score == nil || *result.Index < 0 || *result.Index >= count {
			return items, errors.New("invalid reranker result")
		}
		if _, exists := scored[*result.Index]; exists {
			return items, errors.New("duplicate reranker result")
		}
		scored[*result.Index] = *result.Score
	}
	order := make([]int, count)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		left, leftOK := scored[order[i]]
		right, rightOK := scored[order[j]]
		return leftOK && (!rightOK || left > right)
	})
	ranked := make([]resource.Object, 0, len(items))
	for _, index := range order {
		item := items[index]
		if score, ok := scored[index]; ok {
			item["rerank_score"] = score
		}
		ranked = append(ranked, item)
	}
	return append(ranked, items[count:]...), nil
}
