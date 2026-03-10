package tui

import (
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	notificationMinTTL       = 4 * time.Second
	notificationMaxTTL       = 12 * time.Second
	notificationWordDuration = 300 * time.Millisecond
	notificationBaseTTL      = 2 * time.Second
	notificationMaxWidth     = 72
	notificationMinWidth     = 28
	notificationMaxLines     = 6
	notificationMaxRunes     = 320
)

type NotificationSeverity string

const (
	NotificationInfo  NotificationSeverity = "info"
	NotificationWarn  NotificationSeverity = "warn"
	NotificationError NotificationSeverity = "error"
)

type Notification struct {
	Severity  NotificationSeverity
	Message   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type notificationTickMsg struct {
	id int
}

// NotificationCenter manages one active toast plus the last dismissed message.
// Enqueue/Update integrate with Bubble Tea; Render paints a bottom-right block
// within the caller-provided area.
type NotificationCenter struct {
	active *Notification
	last   *Notification

	expanded bool
	nextID   int
	activeID int

	now func() time.Time
}

func NewNotificationCenter() NotificationCenter {
	return NotificationCenter{
		now: time.Now,
	}
}

// Enqueue shows a notification and schedules expiry based on estimated reading time.
func (n *NotificationCenter) Enqueue(severity NotificationSeverity, message string) tea.Cmd {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}

	now := n.nowTime()
	ttl := estimateNotificationTTL(message)
	notice := &Notification{
		Severity:  severity,
		Message:   message,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}

	n.active = notice
	n.last = cloneNotification(notice)
	n.expanded = false
	n.nextID++
	n.activeID = n.nextID

	return n.expiryCmd(n.activeID, ttl)
}

func (n *NotificationCenter) Update(msg tea.Msg) tea.Cmd {
	tick, ok := msg.(notificationTickMsg)
	if !ok || tick.id != n.activeID || n.active == nil {
		return nil
	}
	if !n.nowTime().Before(n.active.ExpiresAt) {
		n.active = nil
	}
	return nil
}

// HandleKey handles notification-local interactions.
// For ctrl+y it returns the displayed message so the parent can call CopyToClipboard.
func (n *NotificationCenter) HandleKey(key string) (handled bool, copyText string) {
	switch key {
	case KeyAt:
		return n.ToggleLast(), ""
	case KeyYank:
		text := n.CopyText()
		return text != "", text
	default:
		return false, ""
	}
}

func (n *NotificationCenter) ToggleLast() bool {
	if n.last == nil {
		return false
	}
	n.expanded = !n.expanded
	return true
}

func (n *NotificationCenter) CopyText() string {
	if displayed := n.displayed(); displayed != nil {
		return displayed.Message
	}
	return ""
}

func (n *NotificationCenter) Visible() bool {
	return n.displayed() != nil
}

func (n *NotificationCenter) Render(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	displayed := n.displayed()
	if displayed == nil {
		return lipgloss.Place(width, height, lipgloss.Right, lipgloss.Bottom, "")
	}

	cardMaxWidth := min(max(width-2, notificationMinWidth), notificationMaxWidth)
	cardMinWidth := min(notificationMinWidth, max(width-2, 1))

	titleStyle, borderColor := notificationStyles(displayed.Severity)
	bodyWidth := max(cardMaxWidth-4, 1)
	bodyText, truncated := truncateNotificationMessage(displayed.Message, bodyWidth)
	footerText := notificationFooter(displayed.Severity, truncated)

	header := titleStyle.Render(strings.ToUpper(string(displayed.Severity)))
	body := lipgloss.NewStyle().Width(bodyWidth).Render(bodyText)
	content := header + "\n" + body
	if footerText != "" {
		footer := HelpStyle.Render(footerText)
		content += "\n" + footer
	}
	cardWidth := max(min(lipgloss.Width(content)+4, cardMaxWidth), cardMinWidth)

	card := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(cardWidth - 2).
		Render(content)

	return lipgloss.Place(width, height, lipgloss.Right, lipgloss.Bottom, card)
}

func (n *NotificationCenter) displayed() *Notification {
	if n.active != nil && n.nowTime().Before(n.active.ExpiresAt) {
		return n.active
	}
	if n.active != nil && !n.nowTime().Before(n.active.ExpiresAt) {
		n.active = nil
	}
	if n.expanded && n.last != nil {
		return n.last
	}
	return nil
}

func (n *NotificationCenter) nowTime() time.Time {
	if n.now != nil {
		return n.now()
	}
	return time.Now()
}

func (n *NotificationCenter) expiryCmd(id int, ttl time.Duration) tea.Cmd {
	return tea.Tick(ttl, func(time.Time) tea.Msg {
		return notificationTickMsg{id: id}
	})
}

func notificationStyles(severity NotificationSeverity) (lipgloss.Style, lipgloss.AdaptiveColor) {
	switch severity {
	case NotificationError:
		color := lipgloss.AdaptiveColor{Light: "160", Dark: "203"}
		return lipgloss.NewStyle().Bold(true).Foreground(color), color
	case NotificationWarn:
		color := lipgloss.AdaptiveColor{Light: "130", Dark: "214"}
		return lipgloss.NewStyle().Bold(true).Foreground(color), color
	default:
		color := ColorPrimary
		return lipgloss.NewStyle().Bold(true).Foreground(color), color
	}
}

func notificationFooter(severity NotificationSeverity, truncated bool) string {
	if truncated {
		if severity == NotificationError {
			return "Ctrl+y copy full error"
		}
		return "Ctrl+y copy full message"
	}
	return ""
}

func estimateNotificationTTL(message string) time.Duration {
	words := countWords(message)
	if words == 0 {
		return notificationMinTTL
	}

	ttl := notificationBaseTTL + time.Duration(words)*notificationWordDuration
	if ttl < notificationMinTTL {
		return notificationMinTTL
	}
	if ttl > notificationMaxTTL {
		return notificationMaxTTL
	}
	return ttl
}

func countWords(message string) int {
	fields := strings.FieldsFunc(message, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	return len(fields)
}

func cloneNotification(n *Notification) *Notification {
	if n == nil {
		return nil
	}
	cp := *n
	return &cp
}

func truncateNotificationMessage(message string, width int) (string, bool) {
	if width <= 0 {
		return "", len(message) > 0
	}

	runes := []rune(message)
	truncated := false
	if len(runes) > notificationMaxRunes {
		message = string(runes[:notificationMaxRunes]) + "..."
		truncated = true
	}

	wrapped := lipgloss.NewStyle().Width(width).Render(message)
	lines := strings.Split(wrapped, "\n")
	if len(lines) <= notificationMaxLines {
		if truncated {
			lines[len(lines)-1] = ensurePreviewEllipsis(lines[len(lines)-1], width)
			return strings.Join(lines, "\n"), true
		}
		return message, truncated
	}

	lines = lines[:notificationMaxLines]
	lines[len(lines)-1] = ensurePreviewEllipsis(lines[len(lines)-1], width)
	return strings.Join(lines, "\n"), true
}

func ensurePreviewEllipsis(line string, width int) string {
	line = strings.TrimRight(line, " ")
	if strings.Contains(line, "...") {
		return line
	}
	if width <= 3 {
		return strings.Repeat(".", max(width, 0))
	}
	return truncateWithEllipsis(line+" ...", width)
}
