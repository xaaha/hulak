package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ActionItem struct {
	ID      string
	Label   string
	Key     string
	Enabled bool
}

type ActionBadge struct {
	Label    string
	Key      string
	Severity NotificationSeverity
	Visible  bool
}

// ActionRow renders a compact bottom-left row for future gql actions
// such as refresh, send, and save. It can also show an optional leading
// badge like the notification reopen marker.
type ActionRow struct {
	mouse MouseZone
	items []ActionItem
	badge ActionBadge
}

func NewActionRow(items ...ActionItem) ActionRow {
	row := ActionRow{mouse: NewMouseZone()}
	row.SetItems(items)
	return row
}

func (r *ActionRow) SetItems(items []ActionItem) {
	r.items = append(r.items[:0], items...)
}

func (r *ActionRow) SetBadge(badge ActionBadge) {
	r.badge = badge
}

func (r *ActionRow) View() string {
	parts := make([]string, 0, len(r.items)+1)
	if r.badge.Visible && strings.TrimSpace(r.badge.Label) != "" {
		parts = append(parts, r.mouse.Mark(r.badgeZoneID(), renderActionBadge(r.badge)))
	}
	for i := range r.items {
		item := r.items[i]
		if strings.TrimSpace(item.Label) == "" {
			continue
		}
		parts = append(parts, r.mouse.Mark(r.itemZoneID(item.ID), renderActionItem(item)))
	}
	return strings.Join(parts, "  ")
}

func (r *ActionRow) HandleKey(key string) (id string, handled bool) {
	if r.badge.Visible && r.badge.Key != "" && key == r.badge.Key {
		return "badge", true
	}
	for i := range r.items {
		item := r.items[i]
		if !item.Enabled || item.Key == "" || key != item.Key {
			continue
		}
		return item.ID, true
	}
	return "", false
}

func (r *ActionRow) HandleMouse(msg tea.MouseMsg) (id string, handled bool) {
	if !IsLeftClick(msg) {
		return "", false
	}
	if r.badge.Visible && Hit(r.badgeZoneID(), msg) {
		return "badge", true
	}
	for i := range r.items {
		item := r.items[i]
		if !item.Enabled {
			continue
		}
		if Hit(r.itemZoneID(item.ID), msg) {
			return item.ID, true
		}
	}
	return "", false
}

func (r *ActionRow) badgeZoneID() string {
	return r.mouse.ID("action-badge")
}

func (r *ActionRow) itemZoneID(id string) string {
	return r.mouse.ID("action", id)
}

func renderActionItem(item ActionItem) string {
	text := item.Label
	if item.Key != "" {
		text = item.Key + " " + text
	}

	style := MutedActionChipStyle
	if item.Enabled {
		style = ActionChipStyle
	}
	return style.Render(text)
}

func renderActionBadge(badge ActionBadge) string {
	bg := ColorPrimary
	fg := lipgloss.AdaptiveColor{Light: "255", Dark: "255"}
	switch badge.Severity {
	case NotificationError:
		bg = ColorError
	case NotificationWarn:
		bg = ColorWarn
		fg = lipgloss.AdaptiveColor{Light: "16", Dark: "16"}
	}

	text := badge.Label
	if badge.Key != "" && badge.Key != badge.Label {
		text = badge.Key
	}

	return NotificationBadgeBaseStyle.
		Foreground(fg).
		Background(bg).
		Render(text)
}
