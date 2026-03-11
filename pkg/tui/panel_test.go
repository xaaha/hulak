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

	wantW := 30 - borderW
	wantH := 10 - borderH - p.titleHeight()

	if p.viewport.Width != wantW {
		t.Errorf("viewport width = %d, want %d", p.viewport.Width, wantW)
	}
	if p.viewport.Height != wantH {
		t.Errorf("viewport height = %d, want %d", p.viewport.Height, wantH)
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

func TestPanelViewContainsLabelInsideBox(t *testing.T) {
	p := &Panel{Number: 3}
	p.Resize(20, 5)
	p.SetContent("hello", "")

	view := p.View(false)
	if !strings.Contains(view, "[3]") {
		t.Errorf("expected label [3] in view, got:\n%s", view)
	}

	bottomBorderIdx := strings.LastIndex(view, "╰")
	labelIdx := strings.Index(view, "[3]")
	if labelIdx > bottomBorderIdx {
		t.Error("expected label inside box (before bottom border), but it appeared after")
	}

	lines := strings.Split(view, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		idx := strings.Index(lines[i], "[3]")
		if idx < 0 {
			continue
		}
		before := lines[i][:idx]
		if !strings.Contains(before, "  ") {
			t.Errorf("expected leading spaces before right-aligned label, got line: %q", lines[i])
		}
		break
	}
}

func TestPanelViewShowsBottomLeftLabel(t *testing.T) {
	p := &Panel{Number: 4, Label: "Variables"}
	p.Resize(28, 5)
	p.SetContent("", "")

	view := p.View(false)
	if !strings.Contains(view, "Variables") {
		t.Fatalf("expected bottom-left label in view, got:\n%s", view)
	}
	if !strings.Contains(view, "[4]") {
		t.Fatalf("expected panel number in view, got:\n%s", view)
	}
}

func TestPanelFooterOverridesGenericLabel(t *testing.T) {
	p := &Panel{Number: 2, Label: "Variables", Footer: "Search(/)"}
	p.Resize(32, 5)
	p.SetContent("", "")

	view := p.View(false)
	if !strings.Contains(view, "Search(/)") {
		t.Fatalf("expected footer text in view, got:\n%s", view)
	}
	if strings.Contains(view, "Variables") {
		t.Fatalf("expected generic label to be suppressed when footer is set, got:\n%s", view)
	}
}

func TestPanelViewNoLabelWhenNumberZero(t *testing.T) {
	p := &Panel{Number: 0}
	p.Resize(20, 5)
	p.SetContent("hello", "")

	view := p.View(false)
	if strings.Contains(view, "[") && strings.Contains(view, "]") {
		t.Errorf("expected no label when Number=0, got:\n%s", view)
	}
}

func TestPanelViewAlwaysStartsWithBorder(t *testing.T) {
	tests := []struct {
		name   string
		number int
	}{
		{"without_label", 0},
		{"with_label", 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Panel{Number: tc.number}
			p.Resize(20, 5)
			p.SetContent("test", "")

			view := p.View(true)
			if !strings.HasPrefix(view, "╭") {
				t.Errorf("expected view to start with ╭, got %q", view[:20])
			}
		})
	}
}

func TestPanelTitleHeight(t *testing.T) {
	tests := []struct {
		name  string
		panel Panel
		want  int
	}{
		{"empty", Panel{}, 0},
		{"number", Panel{Number: 2}, 1},
		{"footer", Panel{Footer: "Search(/)"}, 1},
		{"label", Panel{Label: "Variables"}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &tc.panel
			if got := p.titleHeight(); got != tc.want {
				t.Errorf("titleHeight() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPanelHeaderHeight(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   int
	}{
		{"empty", "", 0},
		{"single_line", "Status: 200", 1},
		{"two_lines", "Line1\nLine2", 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Panel{Header: tc.header}
			if got := p.headerHeight(); got != tc.want {
				t.Errorf("headerHeight() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestPanelSetHeaderRecalcsViewportHeight(t *testing.T) {
	p := &Panel{Number: 1}
	p.Resize(30, 10)

	borderH := panelBorderStyle.GetVerticalFrameSize()
	hNoHeader := 10 - borderH - p.titleHeight()
	if p.viewport.Height != hNoHeader {
		t.Fatalf("before SetHeader: viewport height = %d, want %d", p.viewport.Height, hNoHeader)
	}

	p.SetHeader("Header Line")
	hWithHeader := 10 - borderH - p.titleHeight() - p.headerHeight()
	if p.viewport.Height != hWithHeader {
		t.Errorf("after SetHeader: viewport height = %d, want %d", p.viewport.Height, hWithHeader)
	}

	p.SetHeader("")
	if p.viewport.Height != hNoHeader {
		t.Errorf("after clearing header: viewport height = %d, want %d", p.viewport.Height, hNoHeader)
	}
}

func TestPanelViewContainsHeader(t *testing.T) {
	p := &Panel{Number: 5}
	p.Resize(40, 10)
	p.SetHeader("200 OK  1.2s")
	p.SetContent("response body", "")

	view := p.View(true)
	if !strings.Contains(view, "200 OK  1.2s") {
		t.Errorf("expected header text in view, got:\n%s", view)
	}
	if !strings.Contains(view, "response body") {
		t.Errorf("expected body text in view, got:\n%s", view)
	}
}
