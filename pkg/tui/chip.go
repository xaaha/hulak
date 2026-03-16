package tui

import "github.com/charmbracelet/lipgloss"

type ChipVariant string

const (
	ChipVariantBadge  ChipVariant = "badge"
	ChipVariantButton ChipVariant = "button"
	ChipVariantSolid  ChipVariant = "solid"
)

// RenderChip renders compact UI labels used for passive badges,
// clickable buttons, and solid markers like the @ notification badge.
func RenderChip(text string, variant ChipVariant, color lipgloss.TerminalColor) string {
	switch variant {
	case ChipVariantButton:
		fg := lipgloss.TerminalColor(lipgloss.Color("15")) // bright white
		if color == ColorWarn {
			fg = lipgloss.Color("0") // black
		}
		return ActionChipStyle.
			Foreground(fg).
			Background(color).
			Render(text)
	case ChipVariantSolid:
		fg := lipgloss.TerminalColor(lipgloss.Color("15")) // bright white
		if color == ColorWarn {
			fg = lipgloss.Color("0") // black
		}
		return NotificationBadgeBaseStyle.
			Foreground(fg).
			Background(color).
			Render(text)
	default:
		return lipgloss.NewStyle().
			Foreground(color).
			Padding(0, 1).
			Render(text)
	}
}

func RenderChipBlock(text string, variant ChipVariant, color lipgloss.TerminalColor, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	switch variant {
	case ChipVariantButton:
		fg := lipgloss.TerminalColor(lipgloss.Color("15")) // bright white
		bg := color
		if color == ColorWarn {
			fg = lipgloss.Color("0") // black
		}
		if color == ColorMuted {
			fg = ColorMuted
			bg = lipgloss.Color("7") // white / light gray
		}
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Foreground(fg).
			Background(bg).
			Align(lipgloss.Center, lipgloss.Center).
			Render(text)
	default:
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Render(RenderChip(text, variant, color))
	}
}
