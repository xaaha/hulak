package gqlexplorer

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
)

func (f *formItem) HandleKey(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	switch f.kind {
	case formItemToggle:
		f.toggle, cmd = f.toggle.Update(msg)
	case formItemTextInput:
		_, cmd = f.input.Update(msg)
	case formItemDropdown:
		f.dropdown, cmd = f.dropdown.Update(msg)
	}
	return cmd
}

// CursorUp moves the inner cursor up, clamping at 0.
func (df *DetailForm) CursorUp() {
	df.cursor = tui.MoveCursorUp(df.cursor)
	df.FocusCurrent()
}

// CursorDown moves the inner cursor down, clamping at the last item.
func (df *DetailForm) CursorDown() {
	df.cursor = tui.MoveCursorDown(df.cursor, len(df.items)-1)
	df.FocusCurrent()
}

func (df *DetailForm) CursorToTop() {
	df.cursor = 0
	df.FocusCurrent()
}

func (df *DetailForm) CursorToBottom() {
	df.cursor = len(df.items) - 1
	df.FocusCurrent()
}

// Search

func (df *DetailForm) IsSearching() bool { return df.search.Active() }

func (df *DetailForm) StartSearch() {
	df.preSearchCursor = df.cursor
	df.search.Start()
}

func (df *DetailForm) StopSearch(confirm bool) {
	df.search.Stop()
	if !confirm {
		df.cursor = df.preSearchCursor
		df.FocusCurrent()
	}
}

func (df *DetailForm) updateSearchMatches() {
	query := strings.ToLower(df.search.Query())
	var indices []int
	if query != "" {
		for i := range df.items {
			if strings.Contains(strings.ToLower(df.items[i].name), query) {
				indices = append(indices, i)
			}
		}
	}
	df.search.SetMatches(indices)
	df.syncSearchCursor()
}

func (df *DetailForm) syncSearchCursor() {
	if m := df.search.CurrentMatch(); m >= 0 {
		df.cursor = m
		df.FocusCurrent()
	}
}

func (df *DetailForm) SearchFooter() string {
	return df.search.Footer()
}

func (df *DetailForm) HandleSearchKey(msg tea.KeyMsg) tea.Cmd {
	stopped, confirmed, cmd := df.search.HandleKey(msg)
	if stopped {
		if !confirmed {
			df.cursor = df.preSearchCursor
			df.FocusCurrent()
		}
		return cmd
	}
	switch msg.String() {
	case tui.KeyUp, tui.KeyCtrlP, tui.KeyDown, tui.KeyCtrlN:
		df.syncSearchCursor()
	default:
		df.updateSearchMatches()
	}
	return cmd
}

// HandleKey routes a key message to the currently focused item.
func (df *DetailForm) HandleKey(msg tea.KeyMsg) tea.Cmd {
	if df.cursor < 0 || df.cursor >= len(df.items) {
		return nil
	}
	item := &df.items[df.cursor]
	key := msg.String()
	if item.kind == formItemToggle && key == tui.KeyEnter {
		msg = tea.KeyMsg{Type: tea.KeySpace}
		key = tui.KeySpace
	}

	// ── Argument items: Space toggles the enabled checkbox ──
	if !item.isField && key == tui.KeySpace && !item.ConsumesTextInput() {
		var cmd tea.Cmd
		newValue := !item.enabled
		if item.kind == formItemToggle {
			cmd = item.HandleKey(msg)
			newValue = item.toggle.Value
		}
		if item.listItem || item.name == item.argName {
			df.setArgEnabled(item.argName, newValue)
		} else {
			item.enabled = newValue
		}
		// A toggle's enabled state drives hasMeaningfulListValue directly, so
		// list rows must resync here too, or a boolean list/nested list can
		// never grow past its first element via keyboard.
		if item.listItem {
			df.syncListArgRows(item.argName)
		}
		if item.listBoundary != "" {
			df.syncNestedListBoundary(item.listBoundary)
		}
		return cmd
	}

	// Argument text inputs: Enter activates/deactivates editing
	if !item.isField && item.kind == formItemTextInput && key == tui.KeyEnter {
		if item.input.Model.Focused() {
			item.input.Model.Blur()
		} else {
			item.input.Model.Focus()
		}
		return nil
	}

	// Argument text inputs: Esc exits editing
	if !item.isField && item.kind == formItemTextInput &&
		key == tui.KeyCancel && item.input.Model.Focused() {
		item.input.Model.Blur()
		return nil
	}

	// Pass through to widget
	cmd := item.HandleKey(msg)

	// Both checks can apply to the same item (a list nested inside a
	// top-level list argument's element belongs to both), so run
	// independently rather than else-if.
	if !item.isField && item.listItem {
		df.syncListArgRows(item.argName)
	}
	if !item.isField && item.listBoundary != "" {
		df.syncNestedListBoundary(item.listBoundary)
	}

	// Field toggles: expand/collapse children on Space
	if key == tui.KeySpace && item.expandable && item.kind == formItemToggle {
		df.toggleExpand(df.cursor)
	}

	return cmd
}

// ConsumesTextInput returns true if the focused item is a text input
// or expanded dropdown that should capture typed characters.
func (df *DetailForm) ConsumesTextInput() bool {
	if df.cursor >= 0 && df.cursor < len(df.items) {
		return df.items[df.cursor].ConsumesTextInput()
	}
	return false
}

func (df *DetailForm) toggleExpand(idx int) {
	item := &df.items[idx]
	if !item.expandable || item.kind != formItemToggle {
		return
	}

	if item.toggle.Value {
		base := ExtractBaseType(item.typeHint)
		ot, ok := resolveType(df.objectTypes, df.endpoint, base)
		if !ok {
			return
		}
		childDepth := item.depth + 1
		children := make([]formItem, 0, len(ot.Fields))
		for _, f := range ot.Fields {
			child := newFieldFormItem(f, false)
			child.depth = childDepth
			childBase := ExtractBaseType(f.Type)
			if childDepth < maxObjectTypeDepth {
				if _, ok := resolveType(df.objectTypes, df.endpoint, childBase); ok {
					child.expandable = true
				}
			}
			children = append(children, child)
		}
		tail := make([]formItem, len(df.items[idx+1:]))
		copy(tail, df.items[idx+1:])
		df.items = append(df.items[:idx+1], children...)
		df.items = append(df.items, tail...)
	} else {
		start := idx + 1
		end := start
		for end < len(df.items) && df.items[end].depth > item.depth {
			end++
		}
		if end > start {
			df.items = append(df.items[:start], df.items[end:]...)
			if df.cursor >= end {
				df.cursor -= (end - start)
			} else if df.cursor > idx {
				df.cursor = idx
			}
		}
	}
}

func (df *DetailForm) hasExpandedDropdown() bool {
	if df.cursor >= 0 && df.cursor < len(df.items) {
		item := &df.items[df.cursor]
		return item.kind == formItemDropdown && item.dropdown.Expanded()
	}
	return false
}

func (df *DetailForm) itemZoneID(prefix string, index int) string {
	return prefix + ":item:" + strconv.Itoa(index)
}

func (df *DetailForm) HandleMouse(prefix string, msg tea.MouseMsg) bool {
	for i := range df.items {
		id := df.itemZoneID(prefix, i)
		if !tui.Hit(id, msg) {
			continue
		}

		df.cursor = i
		df.FocusCurrent()
		item := &df.items[i]

		switch item.kind {
		case formItemToggle:
			_ = item.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
			if !item.isField {
				if item.listItem || item.name == item.argName {
					df.setArgEnabled(item.argName, item.toggle.Value)
				} else {
					item.enabled = item.toggle.Value
				}
				if item.listItem {
					df.syncListArgRows(item.argName)
				}
				if item.listBoundary != "" {
					df.syncNestedListBoundary(item.listBoundary)
				}
			}
			if item.expandable {
				df.toggleExpand(i)
			}
		case formItemTextInput:
			if !item.isField {
				item.enabled = true
			}
			item.input.Model.Focus()
			if item.listItem {
				df.syncListArgRows(item.argName)
			}
			if item.listBoundary != "" {
				df.syncNestedListBoundary(item.listBoundary)
			}
		case formItemDropdown:
			if !item.dropdown.Expanded() {
				if !item.isField {
					item.enabled = true
				}
				item.dropdown.Focus()
				item.dropdown.Expand()
				return true
			}

			_, relY := tui.ZonePos(id, msg)
			optionIndex := relY
			if optionIndex < 0 || optionIndex >= len(item.dropdown.Options) {
				return true
			}
			item.dropdown.Select(optionIndex)
			item.dropdown.Focus()
			if !item.isField {
				item.enabled = true
			}
			if item.listItem {
				df.syncListArgRows(item.argName)
			}
			if item.listBoundary != "" {
				df.syncNestedListBoundary(item.listBoundary)
			}
		}
		return true
	}
	return false
}
