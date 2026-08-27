package gqlexplorer

import (
	"strings"

	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

// View renders all form items in a single flat list.
// Returns the rendered string and the line number of the focused item.
func (df *DetailForm) View(op *UnifiedOperation) (string, int) {
	return df.viewMarked(op, "", func(_ string, s string) string { return s })
}

func (df *DetailForm) ViewMarked(
	op *UnifiedOperation,
	zonePrefix string,
	mark func(id, view string) string,
) (string, int) {
	return df.viewMarked(op, zonePrefix, mark)
}

func (df *DetailForm) viewMarked(
	op *UnifiedOperation,
	zonePrefix string,
	mark func(id, view string) string,
) (string, int) {
	var lines []string

	focused := df.items[df.cursor].Focused()
	headerStyle := tui.HelpStyle
	if focused {
		headerStyle = tui.SubtitleStyle
	}
	header := headerStyle.Render(utils.ChevronRight + op.Name)
	if op.ReturnType != "" {
		header += tui.HelpStyle.Render(": " + op.ReturnType)
	}
	lines = append(lines, header, "")

	const basePad = 4
	const depthIndent = 2
	cursorLine := 0
	for i := range df.items {
		depth := df.items[i].depth
		pad := basePad + depth*depthIndent
		itemPad := strings.Repeat(tui.KeySpace, pad)

		linePrefix := itemPad
		if i == df.cursor {
			if focused {
				linePrefix = strings.Repeat(tui.KeySpace, pad-2) + utils.ChevronRight
			}
			cursorLine = len(lines)
		}
		view := df.items[i].View()
		if zonePrefix != "" {
			view = mark(df.itemZoneID(zonePrefix, i), view)
		}
		for j, line := range strings.Split(view, "\n") {
			if j == 0 {
				lines = append(lines, linePrefix+line)
			} else {
				lines = append(lines, itemPad+line)
			}
		}
		if i == df.cursor && df.items[i].kind == formItemDropdown &&
			df.items[i].dropdown.Expanded() {
			cursorLine += df.items[i].dropdown.Cursor()
		}
	}

	return strings.Join(lines, "\n"), cursorLine
}
