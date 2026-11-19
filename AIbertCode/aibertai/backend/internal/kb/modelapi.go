package kb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"aibertai.local/backend/internal/model"
	"aibertai.local/backend/internal/setting"
)

var knowledgeModelClient = &http.Client{
	Timeout:       45 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

type knowledgeModelStatus int

func (status knowledgeModelStatus) Error() string { return fmt.Sprintf("upstream HTTP %d", status) }

func callKnowledgeModel(ctx context.Context, config setting.KnowledgeEndpoint, operation string, input, output any) error {
	endpoint, err := model.CompatibleEndpoint(config.BaseURL, operation)
	if err != nil {
		return err
	}
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	response, err := knowledgeModelClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return knowledgeModelStatus(response.StatusCode)
	}
	const maxResponseBytes = 2 << 20
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxResponseBytes {
		return errors.New("model response exceeds size limit")
	}
	return json.Unmarshal(data, output)
}
