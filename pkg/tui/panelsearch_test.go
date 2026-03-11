package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPanelSearchStartStop(t *testing.T) {
	ps := NewPanelSearch()
	if ps.Active() {
		t.Fatal("expected inactive before Start")
	}

	ps.Start()
	if !ps.Active() {
		t.Fatal("expected active after Start")
	}

	ps.Stop()
	if ps.Active() {
		t.Fatal("expected inactive after Stop")
	}
}

func TestPanelSearchCurrentMatchEmpty(t *testing.T) {
	ps := NewPanelSearch()
	if got := ps.CurrentMatch(); got != -1 {
		t.Errorf("CurrentMatch() = %d, want -1 with no matches", got)
	}
}

func TestPanelSearchSetMatchesAndCycle(t *testing.T) {
	ps := NewPanelSearch()
	ps.SetMatches([]int{10, 20, 30})

	if got := ps.MatchCount(); got != 3 {
		t.Fatalf("MatchCount() = %d, want 3", got)
	}
	if got := ps.CurrentMatch(); got != 10 {
		t.Errorf("CurrentMatch() = %d, want 10 (first)", got)
	}

	ps.CycleNext()
	if got := ps.CurrentMatch(); got != 20 {
		t.Errorf("after CycleNext, CurrentMatch() = %d, want 20", got)
	}

	ps.CycleNext()
	ps.CycleNext()
	if got := ps.CurrentMatch(); got != 10 {
		t.Errorf("after wrapping CycleNext, CurrentMatch() = %d, want 10", got)
	}

	ps.CyclePrev()
	if got := ps.CurrentMatch(); got != 30 {
		t.Errorf("after CyclePrev wrap, CurrentMatch() = %d, want 30", got)
	}
}

func TestPanelSearchCycleNoMatches(t *testing.T) {
	ps := NewPanelSearch()
	ps.CycleNext()
	ps.CyclePrev()
	if got := ps.CurrentMatch(); got != -1 {
		t.Errorf("cycling with no matches should keep CurrentMatch() = -1, got %d", got)
	}
}

func TestPanelSearchHandleKeyEnter(t *testing.T) {
	ps := NewPanelSearch()
	ps.Start()

	stopped, confirmed, _ := ps.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !stopped || !confirmed {
		t.Errorf("Enter: stopped=%v confirmed=%v, want true/true", stopped, confirmed)
	}
	if ps.Active() {
		t.Error("expected inactive after Enter")
	}
}

func TestPanelSearchHandleKeyEsc(t *testing.T) {
	ps := NewPanelSearch()
	ps.Start()

	stopped, confirmed, _ := ps.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})
	if !stopped || confirmed {
		t.Errorf("Esc: stopped=%v confirmed=%v, want true/false", stopped, confirmed)
	}
	if ps.Active() {
		t.Error("expected inactive after Esc")
	}
}

func TestPanelSearchHandleKeyNav(t *testing.T) {
	ps := NewPanelSearch()
	ps.Start()
	ps.SetMatches([]int{5, 15})

	stopped, _, _ := ps.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if stopped {
		t.Error("Down should not stop search")
	}
	if got := ps.CurrentMatch(); got != 15 {
		t.Errorf("after Down, CurrentMatch() = %d, want 15", got)
	}

	stopped, _, _ = ps.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if stopped {
		t.Error("Up should not stop search")
	}
	if got := ps.CurrentMatch(); got != 5 {
		t.Errorf("after Up, CurrentMatch() = %d, want 5", got)
	}
}

func TestPanelSearchFooterInactive(t *testing.T) {
	ps := NewPanelSearch()
	if got := ps.Footer(); got != "" {
		t.Errorf("Footer() when inactive = %q, want empty", got)
	}
}

func TestPanelSearchFooterActive(t *testing.T) {
	ps := NewPanelSearch()
	ps.Start()

	footer := ps.Footer()
	if !strings.Contains(footer, "Search(/)") {
		t.Errorf("Footer() missing Search(/) label, got: %s", footer)
	}
}

func TestPanelSearchStopClearsMatches(t *testing.T) {
	ps := NewPanelSearch()
	ps.Start()
	ps.SetMatches([]int{1, 2, 3})
	ps.Stop()

	if got := ps.MatchCount(); got != 0 {
		t.Errorf("MatchCount() after Stop = %d, want 0", got)
	}
}

func TestPanelSearchStartResetsState(t *testing.T) {
	ps := NewPanelSearch()
	ps.Start()
	ps.SetMatches([]int{1, 2, 3})
	ps.CycleNext()
	ps.Stop()

	ps.Start()
	if got := ps.Query(); got != "" {
		t.Errorf("Query() after re-Start = %q, want empty", got)
	}
	if got := ps.MatchCount(); got != 0 {
		t.Errorf("MatchCount() after re-Start = %d, want 0", got)
	}
}
