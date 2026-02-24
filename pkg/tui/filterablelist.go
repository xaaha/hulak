package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xaaha/hulak/pkg/utils"
)

// FilterableList is a headless data component that manages a string list with
// text-based filtering, cursor tracking, and item rendering. It does not implement
// tea.Model; embed it in a Bubble Tea model (like SelectorModel or apicaller.helperPane)
// to build interactive selectors on top.
type FilterableList struct {
	items        []string
	lowerItems   []string
	Filtered     []string
	Cursor       int
	TextInput    TextInput
	requireInput bool
}

func NewFilterableList(
	items []string,
	prompt, placeholder string,
	requireInput bool,
) FilterableList {
	lowerItems := make([]string, len(items))
	for i, item := range items {
		lowerItems[i] = strings.ToLower(item)
	}

	var filtered []string
	if !requireInput {
		filtered = items
	}

	return FilterableList{
		items:      items,
		lowerItems: lowerItems,
		Filtered:   filtered,
		TextInput: NewFilterInput(
			TextInputOpts{Prompt: prompt, Placeholder: placeholder, MinWidth: 20},
		),
		requireInput: requireInput,
	}
}

func (f *FilterableList) applyFilter() {
	val := f.TextInput.Model.Value()
	if val == "" {
		if f.requireInput {
			f.Filtered = nil
		} else {
			f.Filtered = f.items
		}
	} else {
		f.Filtered = make([]string, 0, len(f.items))
		lower := strings.ToLower(val)
		for i, item := range f.lowerItems {
			if strings.Contains(item, lower) {
				f.Filtered = append(f.Filtered, f.items[i])
			}
		}
	}
	f.Cursor = ClampCursor(f.Cursor, len(f.Filtered)-1)
}

func (f *FilterableList) ClearFilter() {
	f.TextInput.Model.Reset()
	f.applyFilter()
}

func (f *FilterableList) HasFilterValue() bool {
	return f.TextInput.Model.Value() != ""
}

func (f *FilterableList) SelectCurrent() (string, bool) {
	if len(f.Filtered) == 0 || f.Cursor >= len(f.Filtered) {
		return "", false
	}
	return f.Filtered[f.Cursor], true
}

func (f *FilterableList) UpdateInput(msg tea.Msg) tea.Cmd {
	prev := f.TextInput.Model.Value()
	_, cmd := f.TextInput.Update(msg)
	if f.TextInput.Model.Value() != prev {
		f.applyFilter()
	}
	return cmd
}

// RenderItems renders the filtered list with a cursor indicator on the
// selected item. It returns the rendered content and the line number of
// the cursor, which callers can pass to [SyncViewport] for scroll tracking.
func (f *FilterableList) RenderItems() (string, int) {
	return f.RenderItemsWidth(0)
}

func (f *FilterableList) RenderItemsWidth(maxWidth int) (string, int) {
	if len(f.Filtered) == 0 {
		if f.requireInput && !f.HasFilterValue() {
			return "", 0
		}
		return HelpStyle.Render(listPadding + "(no matches)"), 0
	}

	selectedPrefix := utils.ChevronRight + KeySpace
	selectedAvail := maxWidth - lipgloss.Width(selectedPrefix)
	listAvail := maxWidth - lipgloss.Width(listPadding)

	cursorLine := 0
	lines := make([]string, 0, len(f.Filtered))
	for i, item := range f.Filtered {
		display := item
		if maxWidth > 0 {
			avail := listAvail
			if i == f.Cursor {
				avail = selectedAvail
			}
			display = truncateWithEllipsis(item, avail)
		}

		if i == f.Cursor {
			cursorLine = len(lines)
			lines = append(lines, SubtitleStyle.Render(selectedPrefix+display))
		} else {
			lines = append(lines, listPadding+display)
		}
	}
	return strings.Join(lines, "\n"), cursorLine
}

func truncateWithEllipsis(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	if maxWidth <= len(utils.Ellipsis) {
		return strings.Repeat(".", maxWidth)
	}

	target := maxWidth - len(utils.Ellipsis)
	var b strings.Builder
	used := 0

	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if used+rw > target {
			break
		}
		b.WriteRune(r)
		used += rw
	}

	return b.String() + utils.Ellipsis
}
