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
	if badge := r.BadgeView(); badge != "" {
		parts = append(parts, badge)
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

func (r *ActionRow) BadgeView() string {
	if !r.badge.Visible || strings.TrimSpace(r.badge.Label) == "" {
		return ""
	}
	return r.mouse.Mark(r.badgeZoneID(), renderActionBadge(r.badge))
}

func (r *ActionRow) ViewList() string {
	lines := make([]string, 0, len(r.items)+1)
	for i := range r.items {
		item := r.items[i]
		if strings.TrimSpace(item.Label) == "" {
			continue
		}
		lines = append(lines, r.mouse.Mark(r.itemZoneID(item.ID), renderActionItem(item)))
	}
	if badge := r.BadgeListView(); badge != "" {
		lines = append(lines, badge)
	}
	return strings.Join(lines, "\n")
}

func (r *ActionRow) BadgeListView() string {
	if !r.badge.Visible || strings.TrimSpace(r.badge.Label) == "" {
		return ""
	}
	return r.mouse.Mark(r.badgeZoneID(), renderActionBadgeLine(r.badge))
}

func (r *ActionRow) ViewColumn(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	visible := make([]ActionItem, 0, len(r.items))
	for i := range r.items {
		if strings.TrimSpace(r.items[i].Label) != "" {
			visible = append(visible, r.items[i])
		}
	}
	if len(visible) == 0 {
		return ""
	}

	gaps := len(visible) - 1
	availableHeight := max(height-gaps, len(visible))
	entries := make([]LayoutEntry, len(visible))
	for i := range visible {
		entries[i] = LayoutEntry{Weight: 1, MinSize: 1}
	}
	heights := DistributeSpace(availableHeight, entries)

	lines := make([]string, 0, len(visible))
	for i := range visible {
		item := visible[i]
		lines = append(lines, r.mouse.Mark(
			r.itemZoneID(item.ID),
			renderActionItemBlock(item, width, heights[i]),
		))
	}
	return strings.Join(lines, "\n")
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
	return TitleStyle.Render(item.Label)
}

func renderActionItemBlock(item ActionItem, width, height int) string {
	color := ColorMuted
	if item.Enabled {
		color = ColorPrimary
	}
	return RenderChipBlock(item.Label, ChipVariantButton, color, width, height)
}

func renderActionBadge(badge ActionBadge) string {
	bg := ColorPrimary
	switch badge.Severity {
	case NotificationError:
		bg = ColorError
	case NotificationWarn:
		bg = ColorWarn
	}

	text := badge.Label
	if badge.Key != "" && badge.Key != badge.Label {
		text = badge.Key
	}

	return RenderChip(text, ChipVariantSolid, bg)
}

func renderActionBadgeLine(badge ActionBadge) string {
	var style lipgloss.Style
	switch badge.Severity {
	case NotificationError:
		style = lipgloss.NewStyle().Foreground(ColorError)
	case NotificationWarn:
		style = lipgloss.NewStyle().Foreground(ColorWarn)
	default:
		style = lipgloss.NewStyle().Foreground(ColorPrimary)
	}
	return style.Render(badge.Label)
}
