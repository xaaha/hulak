package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var panelBorderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())

// Panel is a reusable bordered viewport box for right-side content panels.
// Parent owns layout sizing; Panel owns viewport state, border rendering,
// and content caching. Not a full tea.Model — embed in a parent model
// and call Update/View explicitly.
type Panel struct {
	viewport viewport.Model
	ready    bool
	cacheKey string
	Number   int
	Footer   string
}

// Resize updates the panel's outer dimensions. Panel subtracts the border
// frame internally to size the viewport.
func (p *Panel) titleHeight() int {
	if p.Number > 0 {
		return 1
	}
	return 0
}

func (p *Panel) Resize(outerW, outerH int) {
	innerW := max(outerW-panelBorderStyle.GetHorizontalFrameSize(), 0)
	innerH := max(outerH-panelBorderStyle.GetVerticalFrameSize()-p.titleHeight(), 0)

	if !p.ready {
		p.viewport = viewport.New(innerW, innerH)
		p.viewport.MouseWheelEnabled = true
		p.viewport.SetHorizontalStep(2)
		p.ready = true
		return
	}

	p.viewport.Width = innerW
	p.viewport.Height = innerH
}

// CanRender returns false when outer dimensions are too small to fit
// the border frame, avoiding lipgloss rendering artifacts.
func (p *Panel) CanRender() bool {
	if !p.ready {
		return false
	}
	return p.viewport.Width > 0 && p.viewport.Height > 0
}

// SetContent updates viewport content. Returns true if the content was
// updated, false if the cacheKey matched and the update was skipped.
// Pass an empty cacheKey to force update every time.
func (p *Panel) SetContent(content, cacheKey string) bool {
	if cacheKey != "" && cacheKey == p.cacheKey {
		return false
	}
	p.cacheKey = cacheKey
	p.viewport.SetContent(content)
	return true
}

// SyncContent sets content and scrolls so cursorLine stays visible.
func (p *Panel) SyncContent(content string, cursorLine int) {
	p.cacheKey = ""
	SyncViewport(&p.viewport, content, cursorLine, DefaultScrollMargin)
}

// Update forwards messages (scroll, mouse) to the inner viewport.
// Parent should call this only when this panel is focused.
func (p *Panel) Update(msg tea.Msg) tea.Cmd {
	if !p.ready {
		return nil
	}
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return cmd
}

// GotoTop resets the viewport scroll position to the top.
func (p *Panel) GotoTop() {
	if p.ready {
		p.viewport.GotoTop()
	}
}

// GotoBottom sets the viewport scroll position to the bottom.
func (p *Panel) GotoBottom() {
	if p.ready {
		p.viewport.GotoBottom()
	}
}

// ScrollPercent returns the viewport's current scroll position as 0.0–1.0.
func (p *Panel) ScrollPercent() float64 {
	if !p.ready {
		return 0
	}
	return p.viewport.ScrollPercent()
}

// View renders the panel as a bordered box. When Number > 0, a styled
// [N] label appears inside the box at the bottom-right. Focused panels use
// ColorPrimary for the border and label, unfocused panels use ColorMuted.
func (p *Panel) View(focused bool) string {
	borderColor := ColorMuted
	if focused {
		borderColor = ColorPrimary
	}

	content := p.viewport.View()
	contentH := p.viewport.Height

	if p.Number > 0 {
		label := fmt.Sprintf("[%d]", p.Number)
		styledLabel := lipgloss.NewStyle().Foreground(borderColor).Render(label)
		labelLen := len([]rune(label))

		footerW := lipgloss.Width(p.Footer)
		gap := max(p.viewport.Width-footerW-labelLen, 1)
		labelLine := p.Footer + fmt.Sprintf("%*s%s", gap, "", styledLabel)

		content = content + "\n" + labelLine
		contentH += p.titleHeight()
	}

	style := panelBorderStyle.
		BorderForeground(borderColor).
		Width(p.viewport.Width).
		Height(contentH)

	return style.Render(content)
}
