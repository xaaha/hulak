package tui

import (
	"strings"
	"testing"
	"time"
)

func TestEstimateNotificationTTLUsesReadingTimeBounds(t *testing.T) {
	if got := estimateNotificationTTL("short message"); got != notificationMinTTL {
		t.Fatalf("short message ttl = %v, want %v", got, notificationMinTTL)
	}

	long := strings.Repeat("word ", 100)
	if got := estimateNotificationTTL(long); got != notificationMaxTTL {
		t.Fatalf("long message ttl = %v, want %v", got, notificationMaxTTL)
	}
}

func TestNotificationCenterEnqueueStoresLastAndShowsVisible(t *testing.T) {
	now := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	n := NewNotificationCenter()
	n.now = func() time.Time { return now }

	cmd := n.Enqueue(NotificationError, "schema fetch failed")
	if cmd == nil {
		t.Fatal("expected expiry command")
	}
	if !n.Visible() {
		t.Fatal("expected notification to be visible")
	}
	if n.last == nil || n.last.Message != "schema fetch failed" {
		t.Fatal("expected last notification to be stored")
	}
}

func TestNotificationCenterUpdateExpiresActiveNotification(t *testing.T) {
	now := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	n := NewNotificationCenter()
	n.now = func() time.Time { return now }
	n.Enqueue(NotificationWarn, "temporary failure")

	now = now.Add(notificationMinTTL + time.Second)
	n.Update(notificationTickMsg{id: n.activeID})

	if n.Visible() {
		t.Fatal("expected active notification to expire")
	}
	if n.last == nil {
		t.Fatal("expected last notification to remain available")
	}
}

func TestNotificationCenterToggleLastReopensDismissedMessage(t *testing.T) {
	now := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	n := NewNotificationCenter()
	n.now = func() time.Time { return now }
	n.Enqueue(NotificationError, "latest schema error")

	now = now.Add(notificationMinTTL + time.Second)
	n.Update(notificationTickMsg{id: n.activeID})

	if handled := n.ToggleLast(); !handled {
		t.Fatal("expected toggle to succeed when last notification exists")
	}
	if !n.Visible() {
		t.Fatal("expected toggled last notification to be visible")
	}
	if got := n.CopyText(); got != "latest schema error" {
		t.Fatalf("CopyText() = %q", got)
	}
}

func TestNotificationCenterHandleKeySupportsToggleAndCopy(t *testing.T) {
	n := NewNotificationCenter()
	n.now = func() time.Time { return time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC) }
	n.Enqueue(NotificationInfo, "schema loaded")

	handled, copied := n.HandleKey(KeyYank)
	if !handled {
		t.Fatal("expected ctrl+y to be handled when notification is visible")
	}
	if copied != "schema loaded" {
		t.Fatalf("copied text = %q", copied)
	}

	n.active = nil
	handled, _ = n.HandleKey(KeyAt)
	if !handled || !n.Visible() {
		t.Fatal("expected @ to reopen last notification")
	}
}

func TestNotificationCenterRenderPlacesNotificationAtBottomRight(t *testing.T) {
	n := NewNotificationCenter()
	n.now = func() time.Time { return time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC) }
	n.Enqueue(NotificationError, "failed to refresh schema from endpoint")

	view := n.Render(50, 8)
	lines := strings.Split(view, "\n")
	if len(lines) != 8 {
		t.Fatalf("expected 8 lines, got %d", len(lines))
	}
	if !strings.Contains(view, "ERROR") {
		t.Fatalf("expected severity label in view, got:\n%s", view)
	}

	firstContent := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstContent = i
			break
		}
	}
	if firstContent < 0 {
		t.Fatal("expected rendered notification content")
	}
	if firstContent == 0 {
		t.Fatalf("expected vertical bottom placement, got:\n%s", view)
	}

	lastContentLine := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			lastContentLine = lines[i]
			break
		}
	}
	if lastContentLine == "" {
		t.Fatal("expected non-empty content near bottom")
	}
	if strings.HasPrefix(lastContentLine, "╰") {
		t.Fatalf("expected right-aligned content, got line: %q", lastContentLine)
	}
}

func TestNotificationCenterRenderTruncatesLargeErrorsAndShowsCopyHint(t *testing.T) {
	n := NewNotificationCenter()
	n.now = func() time.Time { return time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC) }

	long := strings.Repeat("endpoint failure ", 80)
	n.Enqueue(NotificationError, long)

	view := n.Render(52, 10)
	if !strings.Contains(view, "Ctrl+y copy full error") {
		t.Fatalf("expected copy hint for truncated error, got:\n%s", view)
	}
	if !strings.Contains(view, "...") {
		t.Fatalf("expected truncated preview with ellipsis, got:\n%s", view)
	}
	if got := n.CopyText(); got != strings.TrimSpace(long) {
		t.Fatalf("expected CopyText to keep full message")
	}
}

func TestTruncateNotificationMessageKeepsFullCopySeparateFromPreview(t *testing.T) {
	long := strings.Repeat("schema fetch failed on endpoint ", 50)
	preview, truncated := truncateNotificationMessage(long, 24)

	if !truncated {
		t.Fatal("expected long message to be truncated")
	}
	if preview == long {
		t.Fatal("expected preview to differ from full message")
	}
	if !strings.Contains(preview, "...") {
		t.Fatalf("expected ellipsis in preview, got %q", preview)
	}
}

func TestNotificationCenterRenderEmptyWhenHidden(t *testing.T) {
	n := NewNotificationCenter()
	view := n.Render(20, 3)

	if strings.TrimSpace(view) != "" {
		t.Fatalf("expected blank render for hidden notification, got %q", view)
	}
}
