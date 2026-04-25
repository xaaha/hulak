package utils

import (
	"bytes"
	"strings"
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{"short string unchanged", "hello", 10, "hello"},
		{"exactly at width", "hello", 5, "hello"},
		{"truncated", "hello world", 8, "hello..."},
		{"truncated keeps width", "abcdefghij", 6, "abc..."},
		{"width <= 3 unchanged", "hello", 3, "hello"},
		{"width zero unchanged", "hello", 0, "hello"},
		{"empty string", "", 5, ""},
		{"multi-byte runes", "café résumé", 6, "caf..."}, // counts runes, not bytes
		{"only multi-byte", "••••••", 5, "••..."},        // 6 bullet runes → truncate to 5
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Truncate(tc.in, tc.width)
			if got != tc.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tc.in, tc.width, got, tc.want)
			}
		})
	}
}

// stripAnsi removes ANSI escape codes so tests can compare on visible content.
func stripAnsi(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func TestPrintTable_NoHeader(t *testing.T) {
	var buf bytes.Buffer
	rows := [][]string{
		{"API_KEY", "sk-123"},
		{"DB_URL", "postgres://x"},
	}
	if err := PrintTable(&buf, nil, rows, 0); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "API_KEY") || !strings.Contains(out, "DB_URL") {
		t.Errorf("missing keys in output: %q", out)
	}
	// Second column must start at the same visible position on both rows.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), out)
	}
	if strings.Index(lines[0], "sk-123") != strings.Index(lines[1], "postgres://x") {
		t.Errorf("columns not aligned:\n  %q\n  %q", lines[0], lines[1])
	}
}

func TestPrintTable_AlignsVaryingLengths(t *testing.T) {
	var buf bytes.Buffer
	rows := [][]string{
		{"a", "1"},
		{"medium_key", "2"},
		{"a_much_longer_key_name", "3"},
	}
	if err := PrintTable(&buf, nil, rows, 0); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	pos := strings.Index(lines[0], "1")
	for i, l := range lines[1:] {
		got := strings.Index(l, []string{"2", "3"}[i])
		if got != pos {
			t.Errorf("row %d: value column at offset %d, want %d (line=%q)", i+1, got, pos, l)
		}
	}
}

func TestPrintTable_WithHeader(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"KEY", "VALUE"}
	rows := [][]string{{"foo", "bar"}}
	if err := PrintTable(&buf, headers, rows, 0); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected header + 1 row, got %d lines: %q", len(lines), buf.String())
	}
	// Header is styled; data row is not.
	if !strings.Contains(lines[0], "\x1b[") {
		t.Errorf("header line should contain ANSI escape codes; got %q", lines[0])
	}
	if strings.Contains(lines[1], "\x1b[") {
		t.Errorf("data row should NOT contain ANSI codes; got %q", lines[1])
	}
	// Visible content of header line is "KEY  VALUE".
	if visible := stripAnsi(lines[0]); !strings.HasPrefix(visible, "KEY") {
		t.Errorf("visible header should start with KEY; got %q", visible)
	}
}

func TestPrintTable_AlignsWithStyledHeader(t *testing.T) {
	// ANSI codes in the header are zero-width; columns should still align.
	var buf bytes.Buffer
	headers := []string{"K", "V"}
	rows := [][]string{
		{"abcdef", "ghi"},
		{"x", "yyyy"},
	}
	if err := PrintTable(&buf, headers, rows, 0); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	// Compute visible position of each row's second cell.
	pos := func(line string) int { return strings.Index(stripAnsi(line), strings.Fields(stripAnsi(line))[1]) }
	headerPos := pos(lines[0])
	for i, l := range lines[1:] {
		if got := pos(l); got != headerPos {
			t.Errorf("row %d second column at visible offset %d, want %d", i+1, got, headerPos)
		}
	}
}

func TestPrintTable_TruncatesLongCells(t *testing.T) {
	var buf bytes.Buffer
	long := strings.Repeat("x", 100)
	rows := [][]string{{"K", long}}
	if err := PrintTable(&buf, nil, rows, 20); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "...") {
		t.Errorf("expected ellipsis in truncated output, got %q", out)
	}
	if strings.Contains(out, long) {
		t.Errorf("expected long value to be truncated, but saw it whole in %q", out)
	}
}

func TestPrintTable_NoTruncationWhenZero(t *testing.T) {
	var buf bytes.Buffer
	long := strings.Repeat("x", 500)
	rows := [][]string{{"K", long}}
	if err := PrintTable(&buf, nil, rows, 0); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}
	if !strings.Contains(buf.String(), long) {
		t.Error("expected full value when maxCellWidth=0")
	}
}

func TestPrintTable_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintTable(&buf, nil, nil, 0); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestPrintTable_SingleColumn(t *testing.T) {
	// Used by `env list`: one cell per row, no header.
	var buf bytes.Buffer
	rows := [][]string{{"global"}, {"prod"}, {"staging"}}
	if err := PrintTable(&buf, nil, rows, 0); err != nil {
		t.Fatalf("PrintTable: %v", err)
	}
	want := "global\nprod\nstaging\n"
	if buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}
