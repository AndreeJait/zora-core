package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/outbound"
)

// ToolRegistryHTTP implements outbound.ToolRegistryClient via HTTP to zora-mcp-server.
type ToolRegistryHTTP struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

var _ outbound.ToolRegistryClient = (*ToolRegistryHTTP)(nil)

func NewToolRegistryHTTP(baseURL string, apiKey string) *ToolRegistryHTTP {
	return &ToolRegistryHTTP{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (t *ToolRegistryHTTP) SearchTools(ctx context.Context, query outbound.ToolSearchQuery) ([]entity.ToolContext, error) {
	body := map[string]any{
		"embedding": query.Embedding,
		"tags":      query.Tags,
		"user_id":   query.UserID,
		"limit":     query.Limit,
	}
	resp, err := t.doPost(ctx, "/api/v1/tools/search", body)
	if err != nil {
		return nil, fmt.Errorf("search tools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search tools: unexpected status %d", resp.StatusCode)
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("search tools decode envelope: %w", err)
	}

	var tools []entity.ToolContext
	if err := json.Unmarshal(envelope.Data, &tools); err != nil {
		return nil, fmt.Errorf("search tools decode data: %w", err)
	}

	return tools, nil
}

func (t *ToolRegistryHTTP) CallTool(ctx context.Context, name string, arguments string) (entity.ToolResult, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		args = map[string]any{"raw_input": arguments}
	}

	body := map[string]any{
		"name":      name,
		"arguments": args,
	}
	resp, err := t.doPost(ctx, "/api/v1/mcp/tools/call", body)
	if err != nil {
		return entity.ToolResult{}, fmt.Errorf("call tool: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return entity.ToolResult{
			Name:    name,
			Content: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(bodyBytes)),
			IsError: true,
		}, nil
	}

	var result struct {
		Data struct {
			Content string `json:"content"`
			IsError bool   `json:"is_error"`
		} `json:"data"`
	}
	if err := decodeJSON(resp.Body, &result); err != nil {
		return entity.ToolResult{}, fmt.Errorf("call tool decode: %w", err)
	}

	return entity.ToolResult{
		Name:    name,
		Content: result.Data.Content,
		IsError: result.Data.IsError,
	}, nil
}

func (t *ToolRegistryHTTP) ListTags(ctx context.Context) ([]outbound.TagInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.baseURL+"/api/v1/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("list tags: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if t.apiKey != "" {
		req.Header.Set("X-API-Key", t.apiKey)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list tags: unexpected status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var envelope struct {
		Data []outbound.TagInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("list tags decode: %w", err)
	}

	return envelope.Data, nil
}

func (t *ToolRegistryHTTP) doPost(ctx context.Context, path string, body any) (*http.Response, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if t.apiKey != "" {
		req.Header.Set("X-API-Key", t.apiKey)
	}

	return t.httpClient.Do(req)
}

func decodeJSON(r io.Reader, v any) error {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(r).Decode(&envelope); err != nil {
		// Try direct decode
		r.(io.ReadSeeker).Seek(0, io.SeekStart)
		return json.NewDecoder(r).Decode(v)
	}
	return json.Unmarshal(envelope.Data, v)
}