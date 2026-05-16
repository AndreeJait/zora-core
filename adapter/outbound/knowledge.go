package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/outbound"
)

// KnowledgeHTTP implements outbound.KnowledgeClient via HTTP to zora-knowledge.
type KnowledgeHTTP struct {
	baseURL    string
	httpClient *http.Client
}

var _ outbound.KnowledgeClient = (*KnowledgeHTTP)(nil)

func NewKnowledgeHTTP(baseURL string) *KnowledgeHTTP {
	return &KnowledgeHTTP{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (k *KnowledgeHTTP) SearchDocs(ctx context.Context, query outbound.KnowledgeSearchQuery) ([]entity.KnowledgeSnippet, error) {
	body := map[string]any{
		"embedding": query.Embedding,
		"tags":      query.Tags,
		"user_id":   query.UserID,
		"is_admin":  query.IsAdmin,
		"limit":     query.Limit,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, k.baseURL+"/search", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := k.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search docs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search docs: unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Data []entity.KnowledgeSnippet `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return result.Data, nil
}
