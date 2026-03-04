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
