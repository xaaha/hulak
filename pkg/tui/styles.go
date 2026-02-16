package tui

import "github.com/charmbracelet/lipgloss"

// Adaptive colors that work in both light and dark terminal modes.
// AdaptiveColor{Light: "...", Dark: "..."} automatically picks
// the right color based on terminal background.
var (
	ColorPrimary   = lipgloss.AdaptiveColor{Light: "21", Dark: "75"}   // Blue
	ColorSecondary = lipgloss.AdaptiveColor{Light: "55", Dark: "141"}  // Purple
	ColorSuccess   = lipgloss.AdaptiveColor{Light: "22", Dark: "78"}   // Green
	ColorWarning   = lipgloss.AdaptiveColor{Light: "130", Dark: "214"} // Orange
	ColorError     = lipgloss.AdaptiveColor{Light: "124", Dark: "196"} // Red
	ColorMuted     = lipgloss.AdaptiveColor{Light: "240", Dark: "245"} // Gray
	ColorHighlight = lipgloss.AdaptiveColor{Light: "125", Dark: "212"} // Pink
)

var (
	// TitleStyle for section titles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// SubtitleStyle for secondary titles
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	// HelpStyle for help text at bottom
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// ErrorStyle for error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	// BorderStyle for boxes with borders
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted)
)

var BoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorMuted).
	Padding(1, 1)

// Layout split percentages for the two-column explorer layout.
const (
	LeftPanelPct  = 40
	RightPanelPct = 60
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
