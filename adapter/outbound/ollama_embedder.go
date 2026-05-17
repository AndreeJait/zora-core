package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AndreeJait/zora-core/port/outbound"
)

// OllamaEmbedder implements outbound.Embedder by calling the Ollama /api/embed endpoint.
type OllamaEmbedder struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

var _ outbound.Embedder = (*OllamaEmbedder)(nil)

// OllamaEmbedderConfig holds configuration for the Ollama embedder.
type OllamaEmbedderConfig struct {
	BaseURL string
	Model   string
}

func NewOllamaEmbedder(cfg OllamaEmbedderConfig) *OllamaEmbedder {
	return &OllamaEmbedder{
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// resolveBaseURL returns the Ollama API base URL for the given model.
// Cloud models (containing "-cloud") use the Ollama cloud endpoint with API key auth.
// Local models use the configured base URL.
func (o *OllamaEmbedder) resolveBaseURL(model string) string {
	if strings.Contains(model, "-cloud") {
		return "https://ollama.com"
	}
	return o.baseURL
}

type ollamaEmbedRequest struct {
	Model  string   `json:"model"`
	Input  []string `json:"input"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

func (o *OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	reqBody := ollamaEmbedRequest{
		Model: o.model,
		Input: texts,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.resolveBaseURL(o.model)+"/api/embed", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embed: unexpected status %d", resp.StatusCode)
	}

	var result ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return result.Embeddings, nil
}
