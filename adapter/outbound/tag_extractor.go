package outbound

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/AndreeJait/go-utility/v2/llmw"
	"github.com/AndreeJait/go-utility/v2/logw"
)

const tagSystemPromptFreeform = `You are a tag extraction engine. Given a user task, extract 1-5 relevant tags that categorize the task.
Return ONLY a JSON array of lowercase strings, e.g. ["finance", "reporting"].
No explanation, no markdown, no code fences.`

const tagSystemPromptConstrained = `You are a tag extraction engine. Given a user task, select the most relevant tags from the available tags listed below.

AVAILABLE TAGS:
%s

Rules:
- Select 1-5 tags from the available tags above
- Return ONLY a JSON array of the selected tag names, e.g. ["finance", "reporting"]
- Do NOT invent new tags — only select from the available tags
- No explanation, no markdown, no code fences`

// TagExtractorAdapter implements TagExtractor using an LLM.
type TagExtractorAdapter struct {
	llm llmw.LLM
}

// NewTagExtractorAdapter creates a new TagExtractorAdapter.
func NewTagExtractorAdapter(llm llmw.LLM) *TagExtractorAdapter {
	return &TagExtractorAdapter{llm: llm}
}

// ExtractTags extracts structured tags from the given task text.
// When availableTags is non-empty, the LLM is constrained to select from those known tags.
// Failures are non-fatal: a warning is logged and nil tags are returned.
func (te *TagExtractorAdapter) ExtractTags(ctx context.Context, taskText string, availableTags []string) ([]string, error) {
	systemPrompt := tagSystemPromptFreeform
	if len(availableTags) > 0 {
		systemPrompt = fmt.Sprintf(tagSystemPromptConstrained, strings.Join(availableTags, ", "))
	}

	resp, err := te.llm.Chat(ctx, []llmw.Message{
		{Role: llmw.RoleSystem, Content: systemPrompt},
		{Role: llmw.RoleUser, Content: taskText},
	})
	if err != nil {
		return nil, fmt.Errorf("tag extraction LLM call: %w", err)
	}

	tags := parseTagResponse(resp.Message.Content)
	logw.CtxInfof(ctx, "tag_extractor: extracted tags %v for task %q", tags, truncate(taskText, 50))
	return tags, nil
}

// parseTagResponse extracts a JSON array of strings from the LLM response.
// It handles markdown code fences and various formatting quirks.
func parseTagResponse(raw string) []string {
	content := strings.TrimSpace(raw)

	// Strip markdown code fences if present
	if strings.HasPrefix(content, "```") {
		// Remove opening fence (e.g. ```json or ```)
		newlineIdx := strings.Index(content, "\n")
		if newlineIdx != -1 {
			content = content[newlineIdx+1:]
		}
		// Remove closing fence
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
		content = strings.TrimSpace(content)
	}

	var tags []string
	if err := json.Unmarshal([]byte(content), &tags); err != nil {
		// Try to find a JSON array within the content
		start := strings.Index(content, "[")
		end := strings.LastIndex(content, "]")
		if start != -1 && end > start {
			if err := json.Unmarshal([]byte(content[start:end+1]), &tags); err != nil {
				return nil
			}
		} else {
			return nil
		}
	}

	// Normalize tags: lowercase, trim whitespace
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		normalized := strings.ToLower(strings.TrimSpace(t))
		if normalized != "" {
			result = append(result, normalized)
		}
	}

	// Cap at 5 tags
	if len(result) > 5 {
		result = result[:5]
	}

	return result
}

// truncate returns the first n characters of s, appending "..." if truncated.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}