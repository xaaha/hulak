package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func waitForZoneBounds(t *testing.T, id string) (int, int, int, int) {
	t.Helper()
	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		startX, startY, endX, endY, ok := ZoneBounds(id)
		if ok {
			return startX, startY, endX, endY
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("zone %q was not registered", id)
	return 0, 0, 0, 0
}

func TestNewMouseZonePrefixesAreUnique(t *testing.T) {
	first := NewMouseZone()
	second := NewMouseZone()

	id1 := first.ID("action")
	id2 := second.ID("action")
	if id1 == id2 {
		t.Fatalf("expected unique IDs, got %q", id1)
	}
}

func TestMouseZoneIDJoinsParts(t *testing.T) {
	z := NewMouseZone()
	id := z.ID("button", "save")

	if !strings.Contains(id, "button:save") {
		t.Fatalf("expected joined suffix in ID, got %q", id)
	}
}

func TestScanMouseZonesRegistersMarkedRegionWithoutChangingVisibleText(t *testing.T) {
	z := NewMouseZone()
	id := z.ID("row")
	view := z.Mark(id, "hello")

	scanned := ScanMouseZones(view)
	if scanned != "hello" {
		t.Fatalf("expected scan to preserve visible output, got %q", scanned)
	}

	startX, startY, endX, endY := waitForZoneBounds(t, id)
	if startX != 0 || startY != 0 {
		t.Fatalf("expected zone to start at origin, got (%d,%d)", startX, startY)
	}
	if endX != 4 || endY != 0 {
		t.Fatalf("expected zone to cover 5 cells on one line, got (%d,%d)", endX, endY)
	}
}

func TestHitAndZonePos(t *testing.T) {
	z := NewMouseZone()
	id := z.ID("row")
	_ = ScanMouseZones("prefix " + z.Mark(id, "hello"))

	startX, startY, _, _ := waitForZoneBounds(t, id)
	msg := tea.MouseMsg{
		X:      startX + 2,
		Y:      startY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	}

	if !Hit(id, msg) {
		t.Fatal("expected hit inside zone bounds")
	}
	x, y := ZonePos(id, msg)
	if x != 2 || y != 0 {
		t.Fatalf("expected relative position (2,0), got (%d,%d)", x, y)
	}
}

func TestHitReturnsFalseOutsideZone(t *testing.T) {
	z := NewMouseZone()
	id := z.ID("row")
	_ = ScanMouseZones(z.Mark(id, "hello"))
	_, _, endX, endY := waitForZoneBounds(t, id)

	msg := tea.MouseMsg{
		X:      endX + 1,
		Y:      endY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	}

	if Hit(id, msg) {
		t.Fatal("expected no hit outside zone bounds")
	}
	x, y := ZonePos(id, msg)
	if x != -1 || y != -1 {
		t.Fatalf("expected relative position (-1,-1) outside bounds, got (%d,%d)", x, y)
	}
}

func TestIsLeftClick(t *testing.T) {
	if !IsLeftClick(tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}) {
		t.Fatal("expected left-button release to be recognized as click")
	}
	if IsLeftClick(tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionPress}) {
		t.Fatal("did not expect press to be recognized as click")
	}
	if IsLeftClick(tea.MouseMsg{Button: tea.MouseButtonRight, Action: tea.MouseActionRelease}) {
		t.Fatal("did not expect right-button release to be recognized as click")
	}
}
