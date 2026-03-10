package tui

import (
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

var mouseZoneOnce sync.Once

// MouseZone provides stable, collision-free zone IDs for one component or model.
// Actionable views should mark click targets with IDs derived from the same zone.
type MouseZone struct {
	prefix string
}

// NewMouseZone initializes the shared zone manager and returns a unique prefix
// that can be used to create stable IDs for one component instance.
func NewMouseZone() MouseZone {
	ensureMouseZoneManager()
	return MouseZone{prefix: zone.NewPrefix()}
}

func ensureMouseZoneManager() {
	mouseZoneOnce.Do(zone.NewGlobal)
}

// ID builds a stable namespaced ID for a clickable/focusable region.
func (z MouseZone) ID(parts ...string) string {
	if len(parts) == 0 {
		return z.prefix
	}
	return z.prefix + strings.Join(parts, ":")
}

// Mark wraps an actionable view region with a zone marker.
func (z MouseZone) Mark(id, view string) string {
	ensureMouseZoneManager()
	return zone.Mark(id, view)
}

// Scan registers all marked regions at the root view without affecting layout.
func ScanMouseZones(view string) string {
	ensureMouseZoneManager()
	return zone.Scan(view)
}

// IsLeftClick reports whether msg is a left-button release event.
func IsLeftClick(msg tea.MouseMsg) bool {
	return msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionRelease
}

// Hit reports whether the given mouse event landed inside the provided zone.
func Hit(id string, msg tea.MouseMsg) bool {
	ensureMouseZoneManager()
	return zone.Get(id).InBounds(msg)
}

// ZoneBounds returns the stored bounds for the given zone ID.
func ZoneBounds(id string) (startX, startY, endX, endY int, ok bool) {
	ensureMouseZoneManager()
	info := zone.Get(id)
	if info == nil || info.IsZero() {
		return 0, 0, 0, 0, false
	}
	return info.StartX, info.StartY, info.EndX, info.EndY, true
}

// ZonePos reports the mouse position relative to the given zone.
func ZonePos(id string, msg tea.MouseMsg) (int, int) {
	ensureMouseZoneManager()
	return zone.Get(id).Pos(msg)
}
