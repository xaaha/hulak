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

// SetContent updates viewport content. Skips the update if cacheKey
// matches the previous call, avoiding redundant re-renders.
// Pass an empty cacheKey to force update every time.
func (p *Panel) SetContent(content, cacheKey string) {
	if cacheKey != "" && cacheKey == p.cacheKey {
		return
	}
	p.cacheKey = cacheKey
	p.viewport.SetContent(content)
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

// View renders the panel as a bordered box with a number title (e.g. ╴2╶)
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

// injectBorderTitle splices a ╴N╶ label into the top border line of a
// lipgloss-rendered box. It replaces runes at positions 1..3 (the first
// three ─ after the ╭ corner), so the box must be at least 5 runes wide.
func injectBorderTitle(box string, number int, color lipgloss.TerminalColor) string {
	newlineIdx := strings.IndexByte(box, '\n')
	if newlineIdx < 0 {
		return box
	}

	topLine := []rune(box[:newlineIdx])
	// Need at least: ╭ + 3 replacement runes + ╮ = 5 runes
	if len(topLine) < 5 {
		return box
	}

	numStyle := lipgloss.NewStyle().Foreground(color)
	label := fmt.Sprintf("╴%d╶", number)
	styledLabel := numStyle.Render(label)

	// Build new top line: corner + styled label + remaining border
	var b strings.Builder
	b.WriteRune(topLine[0]) // ╭
	for _, r := range styledLabel {
		b.WriteRune(r)
	}
	// Skip the original ─ runes that the label replaces.
	// len(label) counts the visible runes (╴, digit(s), ╶).
	skip := len([]rune(label))
	if 1+skip < len(topLine) {
		for _, r := range topLine[1+skip:] {
			b.WriteRune(r)
		}
	}

	return b.String() + box[newlineIdx:]
}
