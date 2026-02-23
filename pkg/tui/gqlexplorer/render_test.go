package gqlexplorer

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/utils"
)

func TestTruncateToWidthTruncatesLongText(t *testing.T) {
	got := truncateToWidth("abcdefghijklmnopqrstuvwxyz", 12)
	if lipgloss.Width(got) != 12 {
		t.Fatalf("truncated width = %d, want 12", lipgloss.Width(got))
	}
	if got != "abcdefghi"+utils.Ellipsis {
		t.Fatalf("unexpected truncated string: %q", got)
	}
}

func TestTruncateToWidthReturnsInputWhenFits(t *testing.T) {
	const input = "hulak"
	if got := truncateToWidth(input, 12); got != input {
		t.Fatalf("width >= input should return input, got %q", got)
	}
}

func TestTruncateToWidthHandlesZeroAndNegativeWidths(t *testing.T) {
	if got := truncateToWidth("hello", 0); got != "" {
		t.Fatalf("width 0 got %q, want empty", got)
	}
	if got := truncateToWidth("hello", -1); got != "" {
		t.Fatalf("width -1 got %q, want empty", got)
	}
}
