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
func RenderChip(text string, variant ChipVariant, color lipgloss.AdaptiveColor) string {
	switch variant {
	case ChipVariantButton:
		fg := lipgloss.AdaptiveColor{Light: "255", Dark: "255"}
		if color == ColorWarn {
			fg = lipgloss.AdaptiveColor{Light: "16", Dark: "16"}
		}
		return ActionChipStyle.
			Foreground(fg).
			Background(color).
			Render(text)
	case ChipVariantSolid:
		fg := lipgloss.AdaptiveColor{Light: "255", Dark: "255"}
		if color == ColorWarn {
			fg = lipgloss.AdaptiveColor{Light: "16", Dark: "16"}
		}
		return NotificationBadgeBaseStyle.
			Foreground(fg).
			Background(color).
			Render(text)
	default:
		bgColor := lipgloss.AdaptiveColor{Light: "254", Dark: "236"}
		return lipgloss.NewStyle().
			Foreground(color).
			Background(bgColor).
			Padding(0, 1).
			Render(text)
	}
}

func RenderChipBlock(text string, variant ChipVariant, color lipgloss.AdaptiveColor, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	switch variant {
	case ChipVariantButton:
		fg := lipgloss.AdaptiveColor{Light: "255", Dark: "255"}
		bg := color
		if color == ColorWarn {
			fg = lipgloss.AdaptiveColor{Light: "16", Dark: "16"}
		}
		if color == ColorMuted {
			fg = ColorMuted
			bg = lipgloss.AdaptiveColor{Light: "252", Dark: "238"}
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
