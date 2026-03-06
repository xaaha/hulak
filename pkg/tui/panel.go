package tui

import (
	"fmt"
	"strings"

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
}

// Resize updates the panel's outer dimensions. Panel subtracts the border
// frame internally to size the viewport.
func (p *Panel) Resize(outerW, outerH int) {
	innerW := max(outerW-panelBorderStyle.GetHorizontalFrameSize(), 0)
	innerH := max(outerH-panelBorderStyle.GetVerticalFrameSize(), 0)

	if !p.ready {
		p.viewport = viewport.New(innerW, innerH)
		p.viewport.MouseWheelEnabled = true
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

// ScrollPercent returns the viewport's current scroll position as 0.0–1.0.
func (p *Panel) ScrollPercent() float64 {
	if !p.ready {
		return 0
	}
	return p.viewport.ScrollPercent()
}

// View renders the panel as a bordered box with a number title (e.g. [2])
// injected into the top border. Focused panels use ColorPrimary for the
// border, unfocused panels use ColorMuted.
func (p *Panel) View(focused bool) string {
	borderColor := ColorMuted
	if focused {
		borderColor = ColorPrimary
	}

	style := panelBorderStyle.
		BorderForeground(borderColor).
		Width(p.viewport.Width).
		Height(p.viewport.Height)

	box := style.Render(p.viewport.View())

	if p.Number > 0 {
		box = injectBorderTitle(box, p.Number, borderColor)
	}

	return box
}

// injectBorderTitle splices a [N] label into the top border line of a
// lipgloss-rendered box. It searches for the actual border characters
// by string value (skipping any ANSI escape prefixes) and replaces
// consecutive ─ dashes after the ╭ corner.
func injectBorderTitle(box string, number int, color lipgloss.TerminalColor) string {
	newlineIdx := strings.IndexByte(box, '\n')
	if newlineIdx < 0 {
		return box
	}

	topLine := box[:newlineIdx]

	// Find the ╭ corner character, skipping any ANSI prefix.
	cornerIdx := strings.Index(topLine, "╭")
	if cornerIdx < 0 {
		return box
	}

	// Find the first ─ after the corner.
	afterCorner := cornerIdx + len("╭")
	dashIdx := strings.Index(topLine[afterCorner:], "─")
	if dashIdx < 0 {
		return box
	}
	dashStart := afterCorner + dashIdx

	// Count how many consecutive ─ bytes are available.
	dashLen := len("─")
	nDashes := 0
	pos := dashStart
	for pos+dashLen <= len(topLine) && topLine[pos:pos+dashLen] == "─" {
		nDashes++
		pos += dashLen
	}

	label := fmt.Sprintf("[%d]", number)
	labelRunes := len([]rune(label))
	// Need enough dashes to replace with the label.
	if nDashes < labelRunes {
		return box
	}

	styledLabel := lipgloss.NewStyle().Foreground(color).Render(label)

	replaceEnd := dashStart + labelRunes*dashLen
	return topLine[:dashStart] + styledLabel + topLine[replaceEnd:] + box[newlineIdx:]
}
