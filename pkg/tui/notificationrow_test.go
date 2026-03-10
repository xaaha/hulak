package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestActionRowViewEmptyWhenNoVisibleContent(t *testing.T) {
	row := NewActionRow()
	row.SetBadge(ActionBadge{Label: "@", Key: KeyAt, Visible: false})

	if got := row.View(); got != "" {
		t.Fatalf("expected empty row, got %q", got)
	}
}

func TestActionRowViewShowsBadgeAndItems(t *testing.T) {
	row := NewActionRow(ActionItem{ID: "refresh", Label: "Refresh", Key: "r", Enabled: true})
	row.SetBadge(ActionBadge{
		Label:    "@",
		Key:      KeyAt,
		Severity: NotificationError,
		Visible:  true,
	})

	view := row.View()
	if !strings.Contains(view, "@") {
		t.Fatalf("expected badge in view, got %q", view)
	}
	if !strings.Contains(view, "Refresh") {
		t.Fatalf("expected action label in view, got %q", view)
	}
}

func TestActionRowHandleKeyReturnsBadgeWhenVisible(t *testing.T) {
	row := NewActionRow()
	row.SetBadge(ActionBadge{
		Label:    "@",
		Key:      KeyAt,
		Severity: NotificationWarn,
		Visible:  true,
	})

	id, handled := row.HandleKey(KeyAt)
	if !handled || id != "badge" {
		t.Fatalf("expected badge key handling, got handled=%v id=%q", handled, id)
	}
}

func TestActionRowHandleKeySkipsDisabledActions(t *testing.T) {
	row := NewActionRow(
		ActionItem{ID: "refresh", Label: "Refresh", Key: "r", Enabled: false},
		ActionItem{ID: "save", Label: "Save", Key: "s", Enabled: true},
	)

	if id, handled := row.HandleKey("r"); handled || id != "" {
		t.Fatalf("expected disabled action to be ignored, got handled=%v id=%q", handled, id)
	}
	if id, handled := row.HandleKey("s"); !handled || id != "save" {
		t.Fatalf("expected enabled action, got handled=%v id=%q", handled, id)
	}
}

func TestActionRowHandleMouseSupportsBadgeAndItems(t *testing.T) {
	row := NewActionRow(ActionItem{ID: "refresh", Label: "Refresh", Key: "r", Enabled: true})
	row.SetBadge(ActionBadge{
		Label:    "@",
		Key:      KeyAt,
		Severity: NotificationInfo,
		Visible:  true,
	})

	_ = ScanMouseZones(row.View())

	badgeX, badgeY, _, _ := waitForZoneBounds(t, row.badgeZoneID())
	if id, handled := row.HandleMouse(tea.MouseMsg{
		X:      badgeX,
		Y:      badgeY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	}); !handled || id != "badge" {
		t.Fatalf("expected badge mouse handling, got handled=%v id=%q", handled, id)
	}

	itemX, itemY, _, _ := waitForZoneBounds(t, row.itemZoneID("refresh"))
	if id, handled := row.HandleMouse(tea.MouseMsg{
		X:      itemX,
		Y:      itemY,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	}); !handled || id != "refresh" {
		t.Fatalf("expected item mouse handling, got handled=%v id=%q", handled, id)
	}
}

func TestActionRowHandleMouseIgnoresDisabledItems(t *testing.T) {
	row := NewActionRow(ActionItem{ID: "save", Label: "Save", Key: "s", Enabled: false})

	_ = ScanMouseZones(row.View())
	x, y, _, _ := waitForZoneBounds(t, row.itemZoneID("save"))

	if id, handled := row.HandleMouse(tea.MouseMsg{
		X:      x,
		Y:      y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	}); handled || id != "" {
		t.Fatalf("expected disabled item click to be ignored, got handled=%v id=%q", handled, id)
	}
}
