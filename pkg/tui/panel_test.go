package tui

import (
	"strings"
	"testing"
)

func TestPanelCanRenderFalseBeforeResize(t *testing.T) {
	p := &Panel{Number: 2}
	if p.CanRender() {
		t.Error("expected CanRender false before Resize")
	}
}

func TestPanelCanRenderTrueAfterResize(t *testing.T) {
	p := &Panel{Number: 2}
	p.Resize(20, 10)
	if !p.CanRender() {
		t.Error("expected CanRender true after Resize")
	}
}

func TestPanelCanRenderFalseWhenTooSmall(t *testing.T) {
	p := &Panel{Number: 2}
	p.Resize(2, 2)
	if p.CanRender() {
		t.Error("expected CanRender false when outer matches border frame exactly")
	}
}

func TestPanelResizeUpdatesViewport(t *testing.T) {
	p := &Panel{Number: 1}
	p.Resize(30, 10)

	borderW := panelBorderStyle.GetHorizontalFrameSize()
	borderH := panelBorderStyle.GetVerticalFrameSize()

	if p.viewport.Width != 30-borderW {
		t.Errorf("viewport width = %d, want %d", p.viewport.Width, 30-borderW)
	}
	if p.viewport.Height != 10-borderH {
		t.Errorf("viewport height = %d, want %d", p.viewport.Height, 10-borderH)
	}

	p.Resize(40, 20)
	if p.viewport.Width != 40-borderW {
		t.Errorf("after re-resize, viewport width = %d, want %d", p.viewport.Width, 40-borderW)
	}
}

func TestPanelSetContentCaching(t *testing.T) {
	p := &Panel{Number: 1}
	p.Resize(30, 10)

	if !p.SetContent("first", "key-1") {
		t.Error("expected true on first SetContent")
	}
	if p.SetContent("second", "key-1") {
		t.Error("expected false on same-key SetContent")
	}
	if got := p.viewport.View(); !strings.Contains(got, "first") {
		t.Error("expected cached content to remain after same-key SetContent")
	}

	if !p.SetContent("third", "key-2") {
		t.Error("expected true on new-key SetContent")
	}
	if got := p.viewport.View(); !strings.Contains(got, "third") {
		t.Error("expected content to update with new cache key")
	}
}

func TestPanelSetContentEmptyKeyAlwaysUpdates(t *testing.T) {
	p := &Panel{Number: 1}
	p.Resize(30, 10)

	p.SetContent("aaa", "")
	p.SetContent("bbb", "")
	if got := p.viewport.View(); !strings.Contains(got, "bbb") {
		t.Error("expected empty cacheKey to always update content")
	}
}

func TestPanelScrollPercentBeforeResize(t *testing.T) {
	p := &Panel{Number: 1}
	if got := p.ScrollPercent(); got != 0 {
		t.Errorf("ScrollPercent before Resize = %f, want 0", got)
	}
}

func TestPanelUpdateBeforeResizeReturnsNil(t *testing.T) {
	p := &Panel{Number: 1}
	cmd := p.Update(nil)
	if cmd != nil {
		t.Error("Update before Resize should return nil cmd")
	}
}

func TestPanelViewContainsBorderTitle(t *testing.T) {
	p := &Panel{Number: 3}
	p.Resize(20, 5)
	p.SetContent("hello", "")

	view := p.View(false)
	if !strings.Contains(view, "[3]") {
		t.Errorf("expected border title [3] in view, got:\n%s", view)
	}
}

func TestPanelViewNoBorderTitleWhenNumberZero(t *testing.T) {
	p := &Panel{Number: 0}
	p.Resize(20, 5)
	p.SetContent("hello", "")

	view := p.View(false)
	if strings.Contains(view, "[") && strings.Contains(view, "]") {
		t.Errorf("expected no border title when Number=0, got:\n%s", view)
	}
}

func TestPanelViewStartsWithRoundedCorner(t *testing.T) {
	p := &Panel{Number: 2}
	p.Resize(20, 5)
	p.SetContent("test", "")

	view := p.View(true)
	runes := []rune(view)
	if len(runes) == 0 || string(runes[0]) != "╭" {
		t.Errorf("expected view to start with ╭, got %q", string(runes[0]))
	}
}

func TestInjectBorderTitleNoNewline(t *testing.T) {
	got := injectBorderTitle("no-newline-here", 1, ColorMuted)
	if got != "no-newline-here" {
		t.Error("expected no-op when input has no newline")
	}
}

func TestInjectBorderTitleTooNarrow(t *testing.T) {
	input := "╭──╮\n│  │\n╰──╯"
	got := injectBorderTitle(input, 1, ColorMuted)
	if got != input {
		t.Error("expected no-op when top line is too narrow for title")
	}
}
