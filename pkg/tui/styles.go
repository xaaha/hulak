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

// Badge colors for endpoint tags (cycle through these)
var BadgeColors = []lipgloss.AdaptiveColor{
	{Light: "21", Dark: "39"},   // Blue
	{Light: "22", Dark: "78"},   // Green
	{Light: "130", Dark: "214"}, // Orange
	{Light: "55", Dark: "141"},  // Purple
	{Light: "125", Dark: "212"}, // Pink
	{Light: "30", Dark: "87"},   // Cyan
	{Light: "136", Dark: "221"}, // Yellow
	{Light: "124", Dark: "203"}, // Coral
}

// Common styles
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

	// SuccessStyle for success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// SelectedStyle for selected items
	SelectedStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	// BorderStyle for boxes with borders
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted)
	// FilterStyle for the "Filter:" prompt - no background, just readable text
	FilterStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)
	// FilterTextStyle for the text user types in filter
	FilterTextStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)
	// FilterCursor style
	FilterCursor = lipgloss.NewStyle().
			Foreground(ColorPrimary)
)

var BoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorMuted).
	Padding(1, 1)

// RenderHelp creates a consistently styled help line
func RenderHelp(text string) string {
	return HelpStyle.Render(text)
}

// RenderError creates a consistently styled error message
func RenderError(text string) string {
	return ErrorStyle.Render(text)
}

// RenderBadge creates a colored badge with the given foreground color.
func RenderBadge(text string, color lipgloss.AdaptiveColor) string {
	bgColor := lipgloss.AdaptiveColor{Light: "254", Dark: "236"}
	return lipgloss.NewStyle().
		Foreground(color).
		Background(bgColor).
		Padding(0, 1).
		Render(text)
}

// StandardHelpText returns the standard help text for quit/cancel
func StandardHelpText() string {
	return RenderHelp("enter: select • esc: cancel • ctrl+c: quit")
}
