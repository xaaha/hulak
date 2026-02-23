package tui

import "github.com/charmbracelet/lipgloss"

// Adaptive colors that work in both light and dark terminal modes.
// AdaptiveColor{Light: "...", Dark: "..."} automatically picks
// the right color based on terminal background.
var (
	ColorPrimary   = lipgloss.AdaptiveColor{Light: "21", Dark: "75"}   // Blue
	ColorSecondary = lipgloss.AdaptiveColor{Light: "55", Dark: "141"}  // Purple
	ColorMuted     = lipgloss.AdaptiveColor{Light: "240", Dark: "245"} // Gray
)

var (
	// TitleStyle for section titles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// MutedTitleStyle for unfocused section titles
	MutedTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorMuted)

	// SubtitleStyle for secondary titles
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	// HelpStyle for help text at bottom
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// BorderStyle for boxes with borders
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted)

	// FocusedInputStyle for the actively focused input field
	FocusedInputStyle = BorderStyle.Padding(0, 1).
				BorderForeground(ColorPrimary)

	// InputStyle for unfocused/locked input fields
	InputStyle = BorderStyle.Padding(0, 1)
)

// BoxStyle for full-screen content containers with rounded border and padding.
var BoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorMuted).
	Padding(1, 1)

// Two-column layout split ratio. All width-dependent UI
// (search input, badges, text wrapping) should derive from these
// so the split ratio is defined in one place.
const (
	LeftPanelPct       = 30
	DetailTopHeight    = 40
	DetailFocusBoxW    = 50
	DetailFocusBoxH    = 100
	MinLeftPanelWidth  = 26
	MinRightPanelWidth = 32
)

// RenderBadge creates a colored badge with the given foreground color.
func RenderBadge(text string, color lipgloss.AdaptiveColor) string {
	bgColor := lipgloss.AdaptiveColor{Light: "254", Dark: "236"}
	return lipgloss.NewStyle().
		Foreground(color).
		Background(bgColor).
		Padding(0, 1).
		Render(text)
}
