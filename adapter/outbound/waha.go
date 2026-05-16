package outbound

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/zora-core/config"
	portOutbound "github.com/AndreeJait/zora-core/port/outbound"
)

const (
	maxRetries     = 3
	initialBackoff = 1 * time.Second
	maxBackoff     = 10 * time.Second
)

type wahaClient struct {
	baseURL string
	apiKey  string
	session string
	client  *http.Client
}

// NewWahaClient creates a new WahaClient using the application config.
func NewWahaClient(cfg *config.AppConfig) portOutbound.WahaClient {
	return &wahaClient{
		baseURL: cfg.WAHA.BaseURL,
		apiKey:  cfg.WAHA.APIKey,
		session: cfg.WAHA.Session,
		client:  &http.Client{},
	}
}

func (c *wahaClient) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-Api-Key", c.apiKey)
	}

	return c.client.Do(req)
}

func (c *wahaClient) SendText(ctx context.Context, chatID, text string) error {
	payload := map[string]any{
		"session": c.session,
		"chatId":  chatID,
		"text":    text,
	}

	return c.sendWithRetry(ctx, "/api/sendText", payload, chatID)
}

func (c *wahaClient) SendReaction(ctx context.Context, chatID, messageID, emoji string) error {
	payload := map[string]any{
		"session":   c.session,
		"chatId":    chatID,
		"messageId": messageID,
		"reaction":  emoji,
	}

	resp, err := c.doRequest(ctx, http.MethodPut, "/api/reaction", payload)
	if err != nil {
		return fmt.Errorf("failed to send WAHA reaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send WAHA reaction: status=%d body=%s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *wahaClient) StartTyping(ctx context.Context, chatID string) error {
	payload := map[string]any{
		"session": c.session,
		"chatId":  chatID,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/startTyping", payload)
	if err != nil {
		return fmt.Errorf("failed to start WAHA typing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to start WAHA typing: status=%d body=%s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (c *wahaClient) StopTyping(ctx context.Context, chatID string) error {
	payload := map[string]any{
		"session": c.session,
		"chatId":  chatID,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/stopTyping", payload)
	if err != nil {
		return fmt.Errorf("failed to stop WAHA typing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to stop WAHA typing: status=%d body=%s", resp.StatusCode, string(respBody))
	}
	return nil
}

type lidResponse struct {
	Lid string `json:"lid"`
	PN  string `json:"pn"`
}

func (c *wahaClient) ResolveLID(ctx context.Context, lid string) (string, error) {
	// Strip @lid suffix for the URL path
	lidClean := strings.TrimSuffix(lid, "@lid")
	path := fmt.Sprintf("/api/%s/lids/%s", c.session, lidClean)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", fmt.Errorf("failed to resolve LID: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LID resolution failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result lidResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode LID response: %w", err)
	}

	if result.PN == "" {
		return "", nil
	}

	// Strip @c.us suffix from phone number
	phone := strings.TrimSuffix(result.PN, "@c.us")
	return phone, nil
}

func (c *wahaClient) SendSeen(ctx context.Context, chatID string) error {
	payload := map[string]any{
		"session": c.session,
		"chatId":  chatID,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/sendSeen", payload)
	if err != nil {
		return fmt.Errorf("failed to send WAHA seen: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send WAHA seen: status=%d body=%s", resp.StatusCode, string(respBody))
	}
	return nil
}

// sendTextResponse represents the WAHA /api/sendText response.
type sendTextResponse struct {
	ID  string `json:"id"`
	Key struct {
		ID string `json:"id"`
	} `json:"key"`
}

func (c *wahaClient) SendTextReply(ctx context.Context, chatID, text, replyTo string) (string, error) {
	payload := map[string]any{
		"session":  c.session,
		"chatId":   chatID,
		"text":     text,
		"reply_to": replyTo,
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Min(float64(initialBackoff)*math.Pow(2, float64(attempt-1)), float64(maxBackoff)))
			logw.Infof("retrying WAHA sendTextReply to %s (attempt %d/%d, backoff %v)", chatID, attempt+1, maxRetries+1, backoff)
			select {
			case <-ctx.Done():
				return "", fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		resp, err := c.doRequest(ctx, http.MethodPost, "/api/sendText", payload)
		if err != nil {
			lastErr = fmt.Errorf("failed to send WAHA reply: %w", err)
			logw.Infof("WAHA sendTextReply to %s failed (attempt %d/%d): %v", chatID, attempt+1, maxRetries+1, err)
			continue
		}

		if resp.StatusCode >= 500 {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("failed to send WAHA reply: status=%d body=%s", resp.StatusCode, string(respBody))
			logw.Infof("WAHA sendTextReply to %s got server error (attempt %d/%d): status=%d", chatID, attempt+1, maxRetries+1, resp.StatusCode)
			continue
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return "", fmt.Errorf("failed to send WAHA reply: status=%d body=%s", resp.StatusCode, string(respBody))
		}

		// Parse response to extract message ID
		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			logw.CtxWarningf(ctx, "WAHA sendTextReply: failed to read response body: %v", readErr)
			if attempt > 0 {
				logw.Infof("WAHA sendTextReply to %s succeeded on attempt %d/%d", chatID, attempt+1, maxRetries+1)
			}
			return "", nil
		}

		var result sendTextResponse
		msgID := ""
		if json.Unmarshal(respBody, &result) == nil {
			if result.Key.ID != "" {
				msgID = result.Key.ID
			} else if result.ID != "" {
				msgID = result.ID
			}
		}

		if attempt > 0 {
			logw.Infof("WAHA sendTextReply to %s succeeded on attempt %d/%d", chatID, attempt+1, maxRetries+1)
		}
		return msgID, nil
	}

	return "", lastErr
}

func (c *wahaClient) sendWithRetry(ctx context.Context, path string, payload any, chatID string) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Min(float64(initialBackoff)*math.Pow(2, float64(attempt-1)), float64(maxBackoff)))
			logw.Infof("retrying WAHA send to %s (attempt %d/%d, backoff %v)", chatID, attempt+1, maxRetries+1, backoff)

			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		resp, err := c.doRequest(ctx, http.MethodPost, path, payload)
		if err != nil {
			lastErr = fmt.Errorf("failed to send WAHA message: %w", err)
			logw.Infof("WAHA send to %s failed (attempt %d/%d): %v", chatID, attempt+1, maxRetries+1, err)
			continue
		}

		if resp.StatusCode >= 500 {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("failed to send WAHA message: status=%d body=%s", resp.StatusCode, string(respBody))
			logw.Infof("WAHA send to %s got server error (attempt %d/%d): status=%d", chatID, attempt+1, maxRetries+1, resp.StatusCode)
			continue
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("failed to send WAHA message: status=%d body=%s", resp.StatusCode, string(respBody))
		}

		resp.Body.Close()
		if attempt > 0 {
			logw.Infof("WAHA send to %s succeeded on attempt %d/%d", chatID, attempt+1, maxRetries+1)
		}
		return nil
	}

	return lastErr
}
