package tui

import "testing"

func TestTruncateWithEllipsis(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
	}{
		{name: "no truncation", input: "hello", maxWidth: 10, want: "hello"},
		{name: "truncate", input: "hello world", maxWidth: 8, want: "hello..."},
		{name: "very small width", input: "hello", maxWidth: 2, want: ".."},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateWithEllipsis(tc.input, tc.maxWidth)
			if got != tc.want {
				t.Fatalf("truncateWithEllipsis(%q, %d) = %q, want %q", tc.input, tc.maxWidth, got, tc.want)
			}
		})
	}
}

func TestRenderItemsWidthTruncatesRows(t *testing.T) {
	f := NewFilterableList([]string{"averyverylongitemname"}, "Select: ", "", false)
	content, _ := f.RenderItemsWidth(12)

	if !containsDots(content) {
		t.Fatalf("expected truncated content to contain ellipsis, got %q", content)
	}
}

func containsDots(s string) bool {
	for i := 0; i+2 < len(s); i++ {
		if s[i] == '.' && s[i+1] == '.' && s[i+2] == '.' {
			return true
		}
	}
	return false
}
