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

	// HelpStyle for inline help text (muted gray)
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// HelpBarStyle for the bottom help bar — more noticeable than
	// HelpStyle but not as prominent as body text.
	HelpBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "243", Dark: "250"}).
			Italic(true)

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

// Layout constants. Every height/width/percentage that affects how
// the screen is carved up lives here so adding a new row or panel
// does not require hunting through component files.

// The viewport in the left panel fills whatever is left after
// subtracting the fixed rows (search, status, badges, filter hint)
// from contentHeight. contentHeight itself already excludes
// HelpBarHeight so the help bar sits outside both columns.

const (
	// Column widths
	LeftPanelPct       = 30
	MinLeftPanelWidth  = 26
	MinRightPanelWidth = 32

	// Right panel vertical split (percentage of contentHeight)
	DetailTopPct = 40

	// Fixed-height rows (lines). Subtract these from the total
	// vertical budget before giving the remainder to scrollable
	// content areas.
	SearchBoxHeight = 3 // top border + input + bottom border
	StatusRowHeight = 1 // "N/M operations" line
	HelpBarHeight   = 1 // full-width help bar below both columns
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
