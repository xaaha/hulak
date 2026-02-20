package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/utils"
)

type filterableList struct {
	items        []string
	lowerItems   []string
	filtered     []string
	cursor       int
	textInput    TextInput
	requireInput bool
}

func newFilterableList(
	items []string,
	prompt, placeholder string,
	requireInput bool,
) filterableList {
	lowerItems := make([]string, len(items))
	for i, item := range items {
		lowerItems[i] = strings.ToLower(item)
	}

	var filtered []string
	if !requireInput {
		filtered = items
	}

	return filterableList{
		items:      items,
		lowerItems: lowerItems,
		filtered:   filtered,
		textInput: NewFilterInput(
			TextInputOpts{Prompt: prompt, Placeholder: placeholder, MinWidth: 20},
		),
		requireInput: requireInput,
	}
}

func (f *filterableList) applyFilter() {
	val := f.textInput.Model.Value()
	if val == "" {
		if f.requireInput {
			f.filtered = nil
		} else {
			f.filtered = f.items
		}
	} else {
		f.filtered = make([]string, 0, len(f.items))
		lower := strings.ToLower(val)
		for i, item := range f.lowerItems {
			if strings.Contains(item, lower) {
				f.filtered = append(f.filtered, f.items[i])
			}
		}
	}
	f.cursor = ClampCursor(f.cursor, len(f.filtered)-1)
}

func (f *filterableList) clearFilter() {
	f.textInput.Model.Reset()
	f.applyFilter()
}

func (f filterableList) hasFilterValue() bool {
	return f.textInput.Model.Value() != ""
}

func (f filterableList) selectCurrent() (string, bool) {
	if len(f.filtered) == 0 || f.cursor >= len(f.filtered) {
		return "", false
	}
	return f.filtered[f.cursor], true
}

func (f *filterableList) updateInput(msg tea.Msg) tea.Cmd {
	prev := f.textInput.Model.Value()
	updated, cmd := f.textInput.Update(msg)
	f.textInput = updated
	if f.textInput.Model.Value() != prev {
		f.applyFilter()
	}
	return cmd
}

func (f filterableList) renderItems() string {
	if len(f.filtered) == 0 {
		if f.requireInput && !f.hasFilterValue() {
			return ""
		}
		return HelpStyle.Render(listPadding + "(no matches)")
	}

	lines := make([]string, 0, len(f.filtered))
	for i, item := range f.filtered {
		if i == f.cursor {
			lines = append(lines, SubtitleStyle.Render(utils.ChevronRight+KeySpace+item))
		} else {
			lines = append(lines, listPadding+item)
		}
	}
	return strings.Join(lines, "\n")
}
