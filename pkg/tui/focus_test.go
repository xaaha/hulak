package tui

import "testing"

func newTestRing() (*FocusRing, *Panel, *Panel) {
	p2 := &Panel{Number: 2}
	p3 := &Panel{Number: 3}
	fr := NewFocusRing([]*Panel{p2, p3})
	return &fr, p2, p3
}

func TestNewFocusRingStartsOnLeft(t *testing.T) {
	fr, _, _ := newTestRing()
	if !fr.LeftFocused() {
		t.Error("expected LeftFocused after creation")
	}
}

func TestNextCyclesThroughAllPanels(t *testing.T) {
	fr, p2, p3 := newTestRing()

	fr.Next()
	if !fr.IsFocused(p2) {
		t.Error("expected panel 2 focused after first Next")
	}

	fr.Next()
	if !fr.IsFocused(p3) {
		t.Error("expected panel 3 focused after second Next")
	}

	fr.Next()
	if !fr.LeftFocused() {
		t.Error("expected wrap back to left after third Next")
	}
}

func TestNextClearsTyping(t *testing.T) {
	fr, _, _ := newTestRing()
	fr.SetTyping(true)
	fr.Next()
	if fr.Typing() {
		t.Error("expected typing cleared after Next")
	}
}

func TestFocusByNumberLeft(t *testing.T) {
	fr, _, _ := newTestRing()
	fr.Next()

	ok := fr.FocusByNumber(1)
	if !ok || !fr.LeftFocused() {
		t.Error("expected FocusByNumber(1) to return to left panel")
	}
}

func TestFocusByNumberPanel(t *testing.T) {
	fr, _, p3 := newTestRing()

	ok := fr.FocusByNumber(3)
	if !ok || !fr.IsFocused(p3) {
		t.Error("expected FocusByNumber(3) to focus panel 3")
	}
}

func TestFocusByNumberNotFound(t *testing.T) {
	fr, _, _ := newTestRing()

	ok := fr.FocusByNumber(9)
	if ok {
		t.Error("expected false for non-existent panel number")
	}
	if !fr.LeftFocused() {
		t.Error("focus should not change when number not found")
	}
}

func TestFocusByNumberClearsTyping(t *testing.T) {
	fr, _, _ := newTestRing()
	fr.SetTyping(true)
	fr.FocusByNumber(2)
	if fr.Typing() {
		t.Error("expected typing cleared after FocusByNumber")
	}
}

func TestIsFocusedFalseWhenOnLeft(t *testing.T) {
	fr, p2, _ := newTestRing()
	if fr.IsFocused(p2) {
		t.Error("expected IsFocused false when left panel is focused")
	}
}

func TestHandleKeyTab(t *testing.T) {
	fr, p2, _ := newTestRing()

	consumed, quit := fr.HandleKey(KeyTab)
	if !consumed || quit {
		t.Error("Tab should be consumed, not quit")
	}
	if !fr.IsFocused(p2) {
		t.Error("expected focus to advance after Tab")
	}
}

func TestHandleKeyEnterStartsTyping(t *testing.T) {
	fr, _, _ := newTestRing()

	consumed, quit := fr.HandleKey(KeyEnter)
	if !consumed || quit {
		t.Error("Enter (not typing) should be consumed, not quit")
	}
	if !fr.Typing() {
		t.Error("expected typing mode after Enter")
	}
}

func TestHandleKeyEnterPassthroughWhenTyping(t *testing.T) {
	fr, _, _ := newTestRing()
	fr.SetTyping(true)

	consumed, quit := fr.HandleKey(KeyEnter)
	if consumed || quit {
		t.Error("Enter while typing should not be consumed")
	}
}

func TestHandleKeyEscExitsTyping(t *testing.T) {
	fr, _, _ := newTestRing()
	fr.SetTyping(true)

	consumed, quit := fr.HandleKey(KeyCancel)
	if !consumed || quit {
		t.Error("Esc while typing should be consumed, not quit")
	}
	if fr.Typing() {
		t.Error("expected typing cleared after Esc")
	}
}

func TestHandleKeyEscQuitsWhenNotTyping(t *testing.T) {
	fr, _, _ := newTestRing()

	consumed, quit := fr.HandleKey(KeyCancel)
	if !consumed || !quit {
		t.Error("Esc when not typing should signal quit")
	}
}

func TestHandleKeyNumberSwitchesFocus(t *testing.T) {
	fr, _, p3 := newTestRing()

	consumed, quit := fr.HandleKey("3")
	if !consumed || quit {
		t.Error("number key should be consumed, not quit")
	}
	if !fr.IsFocused(p3) {
		t.Error("expected panel 3 focused after pressing '3'")
	}
}

func TestHandleKeyNumberIgnoredWhenTyping(t *testing.T) {
	fr, _, _ := newTestRing()
	fr.SetTyping(true)

	consumed, _ := fr.HandleKey("2")
	if consumed {
		t.Error("number key should not be consumed while typing")
	}
}

func TestHandleKeyUnknownNotConsumed(t *testing.T) {
	fr, _, _ := newTestRing()

	consumed, quit := fr.HandleKey("x")
	if consumed || quit {
		t.Error("unknown key should not be consumed")
	}
}
