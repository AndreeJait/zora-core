package graph

import (
	"testing"

	"github.com/AndreeJait/zora-core/domain/entity"
)

func TestFilterToolsByScore(t *testing.T) {
	tools := []entity.ToolContext{
		{Name: "tool-a", Score: 0.9},
		{Name: "tool-b", Score: 0.5},
		{Name: "tool-c", Score: 0.3},
		{Name: "tool-d", Score: 0.1},
	}

	tests := []struct {
		name     string
		minScore float64
		expected int
	}{
		{"no filter (0)", 0, 4},
		{"filter at 0.3", 0.3, 3},
		{"filter at 0.5", 0.5, 2},
		{"filter at 0.9", 0.9, 1},
		{"filter at 1.0", 1.0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterToolsByScore(tools, tt.minScore)
			if len(result) != tt.expected {
				t.Errorf("expected %d tools, got %d", tt.expected, len(result))
			}
		})
	}

	t.Run("nil input", func(t *testing.T) {
		result := filterToolsByScore(nil, 0.5)
		if len(result) != 0 {
			t.Errorf("expected empty slice, got %d items", len(result))
		}
	})

	t.Run("negative minScore returns all", func(t *testing.T) {
		result := filterToolsByScore(tools, -1)
		if len(result) != 4 {
			t.Errorf("expected 4 tools, got %d", len(result))
		}
	})
}

func TestFilterDocsByScore(t *testing.T) {
	docs := []entity.KnowledgeSnippet{
		{DocID: "doc-a", Score: 0.85},
		{DocID: "doc-b", Score: 0.45},
		{DocID: "doc-c", Score: 0.2},
	}

	tests := []struct {
		name     string
		minScore float64
		expected int
	}{
		{"no filter (0)", 0, 3},
		{"filter at 0.4", 0.4, 2},
		{"filter at 0.85", 0.85, 1},
		{"filter at 1.0", 1.0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterDocsByScore(docs, tt.minScore)
			if len(result) != tt.expected {
				t.Errorf("expected %d docs, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestBudgetTools(t *testing.T) {
	tools := []entity.ToolContext{
		{Name: "tool-a", Description: "Short desc"},                          // ~3 tokens
		{Name: "tool-b", Description: "A longer description for tool b"},    // ~8 tokens
		{Name: "tool-c", Description: "Another tool with a very long description that takes up many tokens in the prompt context window"}, // ~20 tokens
	}

	t.Run("no budget returns all", func(t *testing.T) {
		result := budgetTools(tools, 0)
		if len(result) != 3 {
			t.Errorf("expected 3 tools, got %d", len(result))
		}
	})

	t.Run("negative budget returns all", func(t *testing.T) {
		result := budgetTools(tools, -1)
		if len(result) != 3 {
			t.Errorf("expected 3 tools, got %d", len(result))
		}
	})

	t.Run("large budget returns all", func(t *testing.T) {
		result := budgetTools(tools, 1000)
		if len(result) != 3 {
			t.Errorf("expected 3 tools, got %d", len(result))
		}
	})

	t.Run("small budget truncates", func(t *testing.T) {
		result := budgetTools(tools, 10)
		if len(result) == 0 {
			t.Error("expected at least 1 tool")
		}
	})

	t.Run("nil input returns nil", func(t *testing.T) {
		result := budgetTools(nil, 100)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestBudgetDocs(t *testing.T) {
	docs := []entity.KnowledgeSnippet{
		{DocID: "doc-a", Content: "Short content"},
		{DocID: "doc-b", Content: "Medium length content for testing"},
		{DocID: "doc-c", Content: "Very long content that should be truncated when the budget is small"},
	}

	t.Run("no budget returns all", func(t *testing.T) {
		result := budgetDocs(docs, 0)
		if len(result) != 3 {
			t.Errorf("expected 3 docs, got %d", len(result))
		}
	})

	t.Run("large budget returns all", func(t *testing.T) {
		result := budgetDocs(docs, 1000)
		if len(result) != 3 {
			t.Errorf("expected 3 docs, got %d", len(result))
		}
	})

	t.Run("small budget truncates", func(t *testing.T) {
		result := budgetDocs(docs, 5)
		if len(result) == 0 {
			t.Error("expected at least 1 doc")
		}
	})

	t.Run("nil input returns nil", func(t *testing.T) {
		result := budgetDocs(nil, 100)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestRoughTokenEstimate(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"ab", 0},
		{"abcd", 1},
		{"abcdefgh", 2},
		{"abcdefghijklmnop", 4},
	}

	for _, tt := range tests {
		result := roughTokenEstimate(tt.input)
		if result != tt.expected {
			t.Errorf("roughTokenEstimate(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}