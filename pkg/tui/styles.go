package tui

import "github.com/charmbracelet/lipgloss"

// Common colors used across TUI components
var (
	ColorPrimary   = lipgloss.Color("39")  // Blue
	ColorSecondary = lipgloss.Color("141") // Purple
	ColorSuccess   = lipgloss.Color("78")  // Green
	ColorWarning   = lipgloss.Color("214") // Orange
	ColorError     = lipgloss.Color("196") // Red
	ColorMuted     = lipgloss.Color("241") // Gray
	ColorHighlight = lipgloss.Color("212") // Pink
)

// Badge colors for endpoint tags (cycle through these)
var BadgeColors = []lipgloss.Color{
	lipgloss.Color("39"),  // Blue
	lipgloss.Color("78"),  // Green
	lipgloss.Color("214"), // Orange
	lipgloss.Color("141"), // Purple
	lipgloss.Color("212"), // Pink
	lipgloss.Color("87"),  // Cyan
	lipgloss.Color("221"), // Yellow
	lipgloss.Color("203"), // Coral
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
	FilterStyle  = lipgloss.NewStyle().Foreground(ColorMuted)
	FilterCursor = lipgloss.NewStyle().Foreground(ColorSecondary)
)

// RenderHelp creates a consistently styled help line
func RenderHelp(text string) string {
	return HelpStyle.Render(text)
}

// RenderError creates a consistently styled error message
func RenderError(text string) string {
	return ErrorStyle.Render(text)
}

// RenderBadge creates a colored badge for endpoint tags
func RenderBadge(text string, colorIndex int) string {
	color := BadgeColors[colorIndex%len(BadgeColors)]
	return lipgloss.NewStyle().
		Foreground(color).
		Background(lipgloss.Color("236")).
		Padding(0, 1).
		Render(text)
}

// StandardHelpText returns the standard help text for quit/cancel
func StandardHelpText() string {
	return RenderHelp("enter: select • esc: cancel • ctrl+c: quit")
}
