package tui

import "github.com/charmbracelet/lipgloss"

// OverlayCenter places overlay in the center of the canvas.
func OverlayCenter(overlay string, width, height int) string {
	if width <= 0 || height <= 0 || overlay == "" {
		return ""
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, overlay)
}
