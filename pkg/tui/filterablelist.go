package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/utils"
)

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

func (f FilterableList) HasFilterValue() bool {
	return f.TextInput.Model.Value() != ""
}

func (f FilterableList) SelectCurrent() (string, bool) {
	if len(f.Filtered) == 0 || f.Cursor >= len(f.Filtered) {
		return "", false
	}
	return f.Filtered[f.Cursor], true
}

func (f *FilterableList) UpdateInput(msg tea.Msg) tea.Cmd {
	prev := f.TextInput.Model.Value()
	updated, cmd := f.TextInput.Update(msg)
	f.TextInput = updated
	if f.TextInput.Model.Value() != prev {
		f.applyFilter()
	}
	return cmd
}

func (f FilterableList) RenderItems() string {
	if len(f.Filtered) == 0 {
		if f.requireInput && !f.HasFilterValue() {
			return ""
		}
		return HelpStyle.Render(listPadding + "(no matches)")
	}

	lines := make([]string, 0, len(f.Filtered))
	for i, item := range f.Filtered {
		if i == f.Cursor {
			lines = append(lines, SubtitleStyle.Render(utils.ChevronRight+KeySpace+item))
		} else {
			lines = append(lines, listPadding+item)
		}
	}
	return strings.Join(lines, "\n")
}
