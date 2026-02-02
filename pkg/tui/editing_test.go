package tui

import "testing"

func TestMoveCursorUp(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		expected int
	}{
		{"from middle", 5, 4},
		{"from 1", 1, 0},
		{"from 0 stays", 0, 0},
		{"negative stays", -1, -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MoveCursorUp(tc.cursor)
			if result != tc.expected {
				t.Errorf("MoveCursorUp(%d) = %d, want %d", tc.cursor, result, tc.expected)
			}
		})
	}
}

func TestMoveCursorDown(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		maxIndex int
		expected int
	}{
		{"from middle", 2, 5, 3},
		{"from 0", 0, 5, 1},
		{"at max stays", 5, 5, 5},
		{"beyond max stays", 6, 5, 6},
		{"single item", 0, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MoveCursorDown(tc.cursor, tc.maxIndex)
			if result != tc.expected {
				t.Errorf("MoveCursorDown(%d, %d) = %d, want %d", tc.cursor, tc.maxIndex, result, tc.expected)
			}
		})
	}
}

func TestClampCursor(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		maxIndex int
		expected int
	}{
		{"within bounds", 2, 5, 2},
		{"at lower bound", 0, 5, 0},
		{"at upper bound", 5, 5, 5},
		{"below lower", -1, 5, 0},
		{"above upper", 10, 5, 5},
		{"empty list", 5, -1, 0},
		{"single item valid", 0, 0, 0},
		{"single item above", 1, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ClampCursor(tc.cursor, tc.maxIndex)
			if result != tc.expected {
				t.Errorf("ClampCursor(%d, %d) = %d, want %d", tc.cursor, tc.maxIndex, result, tc.expected)
			}
		})
	}
}
