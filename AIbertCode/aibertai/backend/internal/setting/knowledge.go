package setting

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

const (
	DocumentVisionKey   = "document_vision"
	RerankerKey         = "knowledge_reranker"
	EmbeddingDefaultKey = "knowledge_embedding"
)

type KnowledgeEndpoint struct {
	BaseURL string `json:"base_url"`
	ModelID string `json:"model_id"`
	APIKey  string `json:"api_key"`
}

func (s *Service) DocumentVision(ctx context.Context) (KnowledgeEndpoint, error) {
	return s.knowledgeEndpoint(ctx, DocumentVisionKey)
}

func (s *Service) Reranker(ctx context.Context) (KnowledgeEndpoint, error) {
	return s.knowledgeEndpoint(ctx, RerankerKey)
}

func (s *Service) knowledgeEndpoint(ctx context.Context, key string) (KnowledgeEndpoint, error) {
	var config KnowledgeEndpoint
	value, err := s.GetValue(ctx, key)
	if errors.Is(err, ErrNotFound) {
		return config, nil
	}
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(value, &config)
	return config, err
}

func (s *Service) prepareKnowledgeEndpoint(ctx context.Context, key string, value json.RawMessage) (json.RawMessage, error) {
	var config KnowledgeEndpoint
	if err := json.Unmarshal(value, &config); err != nil {
		return nil, errors.New("模型配置格式无效")
	}
	operation := "chat/completions"
	if key == RerankerKey {
		operation = "rerank"
	}
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.BaseURL = strings.TrimSuffix(config.BaseURL, "/"+operation)
	config.ModelID = strings.TrimSpace(config.ModelID)
	config.APIKey = strings.TrimSpace(config.APIKey)
	address, err := url.Parse(config.BaseURL)
	if err != nil || (address.Scheme != "http" && address.Scheme != "https") || address.Hostname() == "" ||
		address.User != nil || len(config.BaseURL) > 2048 || strings.ContainsAny(config.BaseURL, "?#") {
		return nil, errors.New("请输入有效的模型 API 地址，例如 https://api.example.com/v1")
	}
	if config.ModelID == "" || len(config.ModelID) > 256 {
		return nil, errors.New("请输入有效的模型 ID")
	}
	if config.APIKey == "" {
		previous, err := s.knowledgeEndpoint(ctx, key)
		if err != nil {
			return nil, err
		}
		if previous.BaseURL == config.BaseURL {
			config.APIKey = previous.APIKey
		}
	}
	if config.APIKey == "" || len(config.APIKey) > 8192 || strings.ContainsFunc(config.APIKey, unicode.IsControl) {
		return nil, errors.New("首次配置或修改 API 地址时，请输入模型 API 密钥")
	}
	return json.Marshal(config)
}

func (s *Service) WithEmbeddingDefaultCheck(check func(context.Context, string) error) *Service {
	s.embeddingDefaultCheck = check
	return s
}

func (s *Service) prepareEmbeddingDefault(ctx context.Context, value json.RawMessage) (json.RawMessage, error) {
	var selection struct {
		ModelID string `json:"model_id"`
	}
	if err := json.Unmarshal(value, &selection); err != nil {
		return nil, errors.New("向量模型配置格式无效")
	}
	selection.ModelID = strings.TrimSpace(selection.ModelID)
	if selection.ModelID != "" {
		id, err := uuid.Parse(selection.ModelID)
		if err != nil || s.embeddingDefaultCheck == nil {
			return nil, errors.New("请选择已启用的系统向量模型")
		}
		selection.ModelID = id.String()
		if err := s.embeddingDefaultCheck(ctx, selection.ModelID); err != nil {
			return nil, err
		}
	}
	return json.Marshal(selection)
}
