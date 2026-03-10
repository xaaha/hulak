package tui

import "github.com/charmbracelet/lipgloss"

// OverlayCenter places overlay in the center of the base canvas.
func OverlayCenter(base, overlay string, width, height int) string {
	if width <= 0 || height <= 0 || overlay == "" {
		return base
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay)
}
