package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PanelSearch provides reusable search-input state, match navigation,
// key handling, and footer rendering. The caller owns match computation;
// PanelSearch owns the input, index list, cursor, and UI chrome.
type PanelSearch struct {
	active       bool
	input        TextInput
	matchIndices []int
	matchCursor  int
}

// NewPanelSearch creates a PanelSearch with up/down keys unbound
// so the host panel can use them for navigation.
func NewPanelSearch() PanelSearch {
	return PanelSearch{
		input: NewFilterInput(TextInputOpts{
			Prompt:      "",
			Placeholder: "search…",
			MinWidth:    20,
		}),
	}
}

// ── State queries ────────────────────────────────────────────

// Active returns true while the search input is open.
func (ps *PanelSearch) Active() bool { return ps.active }

// Query returns the current raw search text.
func (ps *PanelSearch) Query() string { return ps.input.Model.Value() }

// SetQuery replaces the current search text programmatically.
func (ps *PanelSearch) SetQuery(value string) { ps.input.Model.SetValue(value) }

// MatchCount returns how many matches are currently tracked.
func (ps *PanelSearch) MatchCount() int { return len(ps.matchIndices) }

// CurrentMatch returns the index stored at the current match cursor,
// or -1 when there are no matches.
func (ps *PanelSearch) CurrentMatch() int {
	if len(ps.matchIndices) == 0 {
		return -1
	}
	return ps.matchIndices[ps.matchCursor]
}

// ── Lifecycle ────────────────────────────────────────────────

// Start opens the search input and resets all match state.
func (ps *PanelSearch) Start() {
	ps.active = true
	ps.input.Model.SetValue("")
	ps.input.Model.Focus()
	ps.matchIndices = nil
	ps.matchCursor = 0
}

// Stop closes the search input and clears matches.
func (ps *PanelSearch) Stop() {
	ps.active = false
	ps.input.Model.Blur()
	ps.matchIndices = nil
}

// ── Match management (caller-driven) ────────────────────────

// SetMatches replaces the match list and resets the cursor to 0.
func (ps *PanelSearch) SetMatches(indices []int) {
	ps.matchIndices = indices
	ps.matchCursor = 0
}

// CycleNext moves to the next match, wrapping around.
func (ps *PanelSearch) CycleNext() {
	if len(ps.matchIndices) == 0 {
		return
	}
	ps.matchCursor = (ps.matchCursor + 1) % len(ps.matchIndices)
}

// CyclePrev moves to the previous match, wrapping around.
func (ps *PanelSearch) CyclePrev() {
	if len(ps.matchIndices) == 0 {
		return
	}
	ps.matchCursor--
	if ps.matchCursor < 0 {
		ps.matchCursor = len(ps.matchIndices) - 1
	}
}

// ── Key handling ─────────────────────────────────────────────

// HandleKey processes key messages while search is active.
// stopped=true means search ended (confirmed=true for Enter, false for Esc).
// The caller should re-run match computation after non-stop returns.
func (ps *PanelSearch) HandleKey(msg tea.KeyMsg) (stopped bool, confirmed bool, cmd tea.Cmd) {
	switch msg.String() {
	case KeyEnter:
		ps.Stop()
		return true, true, nil
	case KeyCancel:
		ps.Stop()
		return true, false, nil
	case KeyUp, KeyCtrlP:
		ps.CyclePrev()
		return false, false, nil
	case KeyDown, KeyCtrlN:
		ps.CycleNext()
		return false, false, nil
	}
	_, cmd = ps.input.Update(msg)
	return false, false, cmd
}

// ── Rendering ────────────────────────────────────────────────

// statusText returns "N/M" or "no matches" depending on state.
func (ps *PanelSearch) statusText() string {
	if ps.input.Model.Value() == "" {
		return ""
	}
	if len(ps.matchIndices) == 0 {
		return "no matches"
	}
	return fmt.Sprintf("%d/%d", ps.matchCursor+1, len(ps.matchIndices))
}

// Footer renders "Search(/) [input] N/M" or "" when inactive.
func (ps *PanelSearch) Footer() string {
	if !ps.active {
		return ""
	}
	label := lipgloss.NewStyle().Foreground(ColorPrimary).Render("Search(/)")
	input := ps.input.Model.View()
	result := label + " " + input
	if status := ps.statusText(); status != "" {
		result += "  " + HelpStyle.Render(status)
	}
	return result
}
