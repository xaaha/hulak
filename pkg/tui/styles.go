package tui

import "github.com/charmbracelet/lipgloss"

// Semantic colors using basic ANSI (0-15). These are defined by the
// terminal's own color scheme, so they automatically adapt when the
// user switches between dark and light themes.
var (
	ColorPrimary   = lipgloss.Color("4") // blue
	ColorSecondary = lipgloss.Color("5") // magenta
	ColorMuted     = lipgloss.Color("8") // gray (bright black)
	ColorWarn      = lipgloss.Color("3") // yellow
	ColorError     = lipgloss.Color("1") // red
	ColorSuccess   = lipgloss.Color("2") // green
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
			Foreground(ColorMuted).
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

	// ActionChipStyle renders compact action controls.
	ActionChipStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	// NotificationBadgeBaseStyle renders the @ reopen badge.
	NotificationBadgeBaseStyle = lipgloss.NewStyle().
					Bold(true).
					Padding(0, 1)
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
func RenderBadge(text string, color lipgloss.TerminalColor) string {
	return RenderChip(text, ChipVariantBadge, color)
}
