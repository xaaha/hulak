package tui

// FocusRing tracks which panel is focused and whether the user is in
// typing mode. Index -1 means the left panel (search/list) is focused;
// 0+ indexes into the panels slice.
type FocusRing struct {
	panels []*Panel
	index  int
	typing bool
}

// NewFocusRing creates a ring starting on the left panel (index -1).
func NewFocusRing(panels []*Panel) FocusRing {
	return FocusRing{panels: panels, index: -1}
}

// LeftFocused returns true when the left panel (search/list) has focus.
func (f *FocusRing) LeftFocused() bool {
	return f.index == -1
}

// IsFocused returns true if the given panel is the currently focused one.
func (f *FocusRing) IsFocused(p *Panel) bool {
	if f.index < 0 || f.index >= len(f.panels) {
		return false
	}
	return f.panels[f.index] == p
}

// Next advances focus to the next panel, wrapping from the last panel
// back to the left panel (-1). Exits typing mode on switch.
func (f *FocusRing) Next() {
	f.typing = false
	f.index++
	if f.index >= len(f.panels) {
		f.index = -1
	}
}

// Prev moves focus to the previous panel, wrapping from the left panel
// back to the last panel. Exits typing mode on switch.
func (f *FocusRing) Prev() {
	f.typing = false
	f.index--
	if f.index < -1 {
		f.index = len(f.panels) - 1
	}
}

// FocusByNumber jumps to the panel with the given Number field.
// Number 1 focuses the left panel. Returns false if no match found.
// Exits typing mode on switch.
func (f *FocusRing) FocusByNumber(num int) bool {
	if num == 1 {
		f.typing = false
		f.index = -1
		return true
	}
	for i, p := range f.panels {
		if p.Number == num {
			f.typing = false
			f.index = i
			return true
		}
	}
	return false
}

func (f *FocusRing) Typing() bool {
	return f.typing
}

func (f *FocusRing) SetTyping(v bool) {
	f.typing = v
}

// TODO-gql: gqlexplorer.handleKey duplicates Esc/Tab/Enter/number-key logic
// instead of delegating to HandleKey. Consolidate in Phase 3 when multiple
// editable panels make the duplication painful.
//
// HandleKey processes focus-related keys. Returns two bools:
//   - consumed: true if the key was handled (caller should not process it further)
//   - quit: true if Esc was pressed while already not typing (caller should exit)
func (f *FocusRing) HandleKey(key string) (consumed, quit bool) {
	switch key {
	case KeyTab:
		f.Next()
		return true, false

	case KeyEnter:
		if !f.typing {
			f.typing = true
			return true, false
		}
		return false, false

	case KeyCancel:
		if f.typing {
			f.typing = false
			return true, false
		}
		return true, true

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if !f.typing {
			num := int(key[0] - '0')
			f.FocusByNumber(num)
			return true, false
		}
		return false, false

	default:
		return false, false
	}
}
