package utils

import (
	"io"
	"os"
	"regexp"
	"strings"

	"golang.org/x/term"
)

// Two spaces matches the mise / kubectl convention — tight enough to scan,
// DefaultTablePadding is the gap (in spaces) between columns in PrintTable.
// loose enough not to mush columns together.
const DefaultTablePadding = 2

// DefaultTableMaxCellWidth is the soft per-cell truncation threshold for
// human-readable list views. Tight enough that long tokens, JWTs, and JSON
// blobs don't wrap or wreck the terminal; loose enough that typical URLs
// and IDs print in full.
//
// Users who need the full value should run `hulak secrets get KEY` — the table
// view is for scanning, the get command is for reading.
const DefaultTableMaxCellWidth = 60

// tableHeaderStyle wraps header cells in bold + italic + bright yellow.
// Bold ensures readability on terminals that don't render italic; yellow
// matches PrintSectionHeader so section/table headings feel like the same
// visual family.
const tableHeaderStyle = "\033[1;3;93m"

// ansiRe matches CSI SGR escape sequences (\x1b[...m). Used to strip styling
// when measuring a cell's visible width so columns align by what users see,
// not by raw byte count.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// visibleWidth returns the rune count of s with ANSI escape codes stripped.
// This is what we align on, since escape codes are zero-width visually.
func visibleWidth(s string) int {
	return len([]rune(ansiRe.ReplaceAllString(s, "")))
}

// Truncate shortens s to at most width runes. When truncation happens, the
// last three characters become "..." so the total width stays at exactly
// width. width <= 3 returns s unchanged (no useful truncation possible).
//
// Counts runes, not bytes — multi-byte characters (e.g. `••••`) are handled
// correctly.
func Truncate(s string, width int) string {
	if width <= 3 {
		return s
	}
	r := []rune(s)
	if len(r) <= width {
		return s
	}
	return string(r[:width-3]) + Ellipsis
}

// PrintTable writes rows as a borderless aligned table to out. Use it for any
// "list of records" CLI output: env names, key/value pairs, recipients, etc.
//
//   - headers, if non-nil, is printed as the first row, styled bold+italic+yellow.
//   - rows is a slice of cell slices.
//   - maxCellWidth truncates over-long cells via Truncate. 0 disables
//     truncation; DefaultTableMaxCellWidth is the standard limit.
//   - out=nil falls back to os.Stdout.
//
// Alignment is ANSI-aware: cells containing escape codes (styled headers,
// colored values) are padded based on their *visible* width, so columns line
// up cleanly regardless of styling.
func PrintTable(out io.Writer, headers []string, rows [][]string, maxCellWidth int) error {
	if out == nil {
		out = os.Stdout
	}

	truncate := func(c string) string {
		if maxCellWidth > 0 {
			return Truncate(c, maxCellWidth)
		}
		return c
	}

	// Truncate everything first so width measurements reflect what'll print.
	hdr := make([]string, len(headers))
	for i, h := range headers {
		hdr[i] = truncate(h)
	}
	body := make([][]string, len(rows))
	for i, r := range rows {
		body[i] = make([]string, len(r))
		for j, c := range r {
			body[i][j] = truncate(c)
		}
	}

	// Apply header style after truncation so escape codes don't get cut.
	styledHeaders := make([]string, len(hdr))
	for i, h := range hdr {
		styledHeaders[i] = tableHeaderStyle + h + ColorReset
	}

	// Compute per-column widths from visible content.
	var widths []int
	measure := func(r []string) {
		for i, c := range r {
			for len(widths) <= i {
				widths = append(widths, 0)
			}
			if w := visibleWidth(c); w > widths[i] {
				widths[i] = w
			}
		}
	}
	if len(styledHeaders) > 0 {
		measure(styledHeaders)
	}
	for _, r := range body {
		measure(r)
	}
	if len(widths) == 0 {
		return nil
	}

	gap := strings.Repeat(" ", DefaultTablePadding)
	writeRow := func(cells []string) error {
		var sb strings.Builder
		for i, c := range cells {
			sb.WriteString(c)
			// Pad and gap only between cells — never after the last one,
			// to keep lines free of trailing whitespace.
			if i < len(cells)-1 {
				if pad := widths[i] - visibleWidth(c); pad > 0 {
					sb.WriteString(strings.Repeat(" ", pad))
				}
				sb.WriteString(gap)
			}
		}
		sb.WriteByte('\n')
		_, err := io.WriteString(out, sb.String())
		return err
	}

	if len(styledHeaders) > 0 {
		if err := writeRow(styledHeaders); err != nil {
			return err
		}
	}
	for _, r := range body {
		if err := writeRow(r); err != nil {
			return err
		}
	}
	return nil
}

// StdoutHeaders returns headers when stdout is a TTY, nil when piped.
// Hiding headers under pipe redirection keeps scripts like
// `for env in $(hulak secrets list)` clean — the same convention as kubectl / mise.
func StdoutHeaders(headers []string) []string {
	if term.IsTerminal(int(os.Stdout.Fd())) { //nolint:gosec // G115 fd is small non-neg
		return headers
	}
	return nil
}
