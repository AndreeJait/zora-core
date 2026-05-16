package outbound

import (
	"testing"
)

func TestParseTagResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "valid JSON array",
			input:    `["finance", "reporting", "golang"]`,
			expected: []string{"finance", "reporting", "golang"},
		},
		{
			name:     "JSON array with markdown fences",
			input:    "```json\n[\"finance\", \"reporting\"]\n```",
			expected: []string{"finance", "reporting"},
		},
		{
			name:     "JSON array with plain fences",
			input:    "```\n[\"finance\", \"reporting\"]\n```",
			expected: []string{"finance", "reporting"},
		},
		{
			name:     "mixed case tags are lowercased",
			input:    `["Finance", "REPORTING", "GoLang"]`,
			expected: []string{"finance", "reporting", "golang"},
		},
		{
			name:     "tags with whitespace are trimmed",
			input:    `[" finance ", " reporting "]`,
			expected: []string{"finance", "reporting"},
		},
		{
			name:     "empty strings are filtered",
			input:    `["finance", "", "reporting"]`,
			expected: []string{"finance", "reporting"},
		},
		{
			name:     "more than 5 tags are capped",
			input:    `["a", "b", "c", "d", "e", "f", "g"]`,
			expected: []string{"a", "b", "c", "d", "e"},
		},
		{
			name:     "empty response returns nil",
			input:    "",
			expected: nil,
		},
		{
			name:     "random text with embedded array",
			input:    "Here are the tags: [\"news\", \"sports\"] and that's it.",
			expected: []string{"news", "sports"},
		},
		{
			name:     "malformed JSON returns nil",
			input:    "this is not json at all",
			expected: nil,
		},
		{
			name:     "single tag",
			input:    `["finance"]`,
			expected: []string{"finance"},
		},
		{
			name:     "markdown with extra whitespace",
			input:    "  ```json\n  [\"cache\", \"performance\"]\n  ```  ",
			expected: []string{"cache", "performance"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTagResponse(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, result)
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("expected result[%d] = %q, got %q", i, tt.expected[i], result[i])
				}
			}
		})
	}
}