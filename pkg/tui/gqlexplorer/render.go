package gqlexplorer

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

const (
	// treeBranch is the Unicode box-drawing character used for
	// input-type field tree rendering in the detail panel.
	treeBranch = "│"
)

var (
	itemPrefix     = strings.Repeat(tui.KeySpace, itemPadding)
	selectedPrefix = strings.Repeat(
		tui.KeySpace,
		itemPadding-len(utils.ChevronRight),
	) + utils.ChevronRight
	detailPrefix = strings.Repeat(tui.KeySpace, detailPadding)
	toggleOff    = strings.Repeat(tui.KeySpace, 2)
	toggleOn     = checkMark + tui.KeySpace
)

// truncateToWidth truncates s to at most width visual columns,
// correctly handling ANSI escape sequences and wide characters.
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	return ansi.Truncate(s, width, utils.Ellipsis)
}

func appendWrappedHelpLines(lines []string, text string, width int, prefix string) []string {
	if text == "" {
		return lines
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(text)
	for line := range strings.SplitSeq(wrapped, "\n") {
		lines = append(lines, tui.HelpStyle.Render(prefix+line))
	}
	return lines
}

func (m *Model) renderList() (string, int) {
	if len(m.filtered) == 0 {
		return tui.HelpStyle.Render(
			strings.Repeat(tui.KeySpace, itemPadding-len(utils.ChevronRight)) + noMatchesLabel,
		), 0
	}

	var lines []string
	cursorLine := 0
	var currentType OperationType
	for i := range m.filtered {
		op := &m.filtered[i]
		if op.Type != currentType {
			currentType = op.Type
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, tui.RenderBadge(string(currentType), badgeColor[currentType]))
		}
		if i == m.cursor {
			cursorLine = len(lines)
			lines = append(lines, tui.SubtitleStyle.Render(selectedPrefix+op.Name))
			wrapW := max(m.leftPanelWidth()-detailPadding, 1)
			lines = appendWrappedHelpLines(lines, op.Description, wrapW, detailPrefix)
			// Full URL shown intentionally; badges/filters use shortened form.
			lines = appendWrappedHelpLines(lines, op.Endpoint, wrapW, detailPrefix)
		} else {
			lines = append(lines, itemPrefix+op.Name)
		}
	}
	return strings.Join(lines, "\n"), cursorLine
}

func (m *Model) renderEndpointPicker() (string, int) {
	if len(m.endpoints) == 0 {
		return tui.HelpStyle.Render(itemPrefix + noMatchesLabel), 0
	}

	var lines []string
	cursorLine := 0
	for i, ep := range m.endpoints {
		prefix := itemPrefix
		if i == m.endpointCursor {
			prefix = selectedPrefix
			cursorLine = len(lines)
		}
		toggle := toggleOff
		if m.pendingEndpoints[ep] {
			toggle = toggleOn
		}
		style := lipgloss.NewStyle()
		if i == m.endpointCursor {
			style = tui.SubtitleStyle
		}
		lines = append(lines, style.Render(prefix+toggle+ep))
	}
	return strings.Join(lines, "\n"), cursorLine
}

func (m *Model) renderBadges() string {
	var shortened []string
	for ep := range m.activeEndpoints {
		shortened = append(shortened, ep)
	}
	sort.Strings(shortened)

	maxW := m.leftPanelWidth()
	ellipsis := tui.HelpStyle.Render(utils.Ellipsis)
	ellipsisW := lipgloss.Width(tui.KeySpace + ellipsis)
	var result string
	for i, s := range shortened {
		badge := tui.RenderBadge(s, tui.ColorMuted)
		candidate := result
		if candidate != "" {
			candidate += tui.KeySpace
		}
		candidate += badge
		if i > 0 && lipgloss.Width(candidate)+ellipsisW > maxW {
			result += tui.KeySpace + ellipsis
			break
		}
		if i == 0 && lipgloss.Width(candidate) > maxW {
			return ellipsis
		}
		result = candidate
	}
	return result
}

func renderDetail(
	op *UnifiedOperation,
	inputTypes map[string]graphql.InputType,
	objectTypes map[string]graphql.ObjectType,
	unionTypes map[string]graphql.UnionType,
	interfaceTypes map[string]graphql.InterfaceType,
) string {
	pad := strings.Repeat(tui.KeySpace, 2)
	argPad := strings.Repeat(tui.KeySpace, 4)

	var lines []string

	header := tui.SubtitleStyle.Render(utils.ChevronRight + op.Name)
	if op.ReturnType != "" {
		header += tui.HelpStyle.Render(": " + op.ReturnType)
	}
	lines = append(lines, header, "")

	if len(op.Arguments) > 0 {
		lines = append(lines, pad+tui.HelpStyle.Render("Arguments:"))

		nameW, typeW := 0, 0
		for _, arg := range op.Arguments {
			if len(arg.Name) > nameW {
				nameW = len(arg.Name)
			}
			if len(arg.Type) > typeW {
				typeW = len(arg.Type)
			}
		}

		for _, arg := range op.Arguments {
			required := ""
			if strings.HasSuffix(arg.Type, "!") {
				required = tui.KeySpace + tui.HelpStyle.Render("(required)")
			}
			name := fmt.Sprintf("%-*s", nameW, arg.Name)
			typStr := fmt.Sprintf("%-*s", typeW, arg.Type)
			lines = append(lines, argPad+name+tui.KeySpace+tui.KeySpace+typStr+required)

			base := ExtractBaseType(arg.Type)
			if it, ok := resolveType(inputTypes, op.Endpoint, base); ok {
				lines = appendInputTypeFields(
					lines,
					it,
					argPad+strings.Repeat(tui.KeySpace, 2),
					inputTypes,
					op.Endpoint,
					1,
				)
			}
		}
		lines = append(lines, "")
	}

	if op.ReturnType != "" {
		base := ExtractBaseType(op.ReturnType)
		if ut, ok := resolveType(unionTypes, op.Endpoint, base); ok {
			lines = append(lines, pad+tui.HelpStyle.Render("Inline Fragments:"))
			lines = appendFragmentTypes(lines, ut.PossibleTypes, argPad, objectTypes, op.Endpoint)
			lines = append(lines, "")
		} else if it, ok := resolveType(interfaceTypes, op.Endpoint, base); ok {
			if len(it.Fields) > 0 {
				lines = append(lines, pad+tui.HelpStyle.Render("Fields:"))
				ot := graphql.ObjectType{Name: it.Name, Fields: it.Fields}
				lines = appendObjectTypeFields(lines, ot, argPad, objectTypes, op.Endpoint, 1)
				lines = append(lines, "")
			}
			if len(it.PossibleTypes) > 0 {
				lines = append(lines, pad+tui.HelpStyle.Render("Inline Fragments:"))
				lines = appendFragmentTypes(lines, it.PossibleTypes, argPad, objectTypes, op.Endpoint)
				lines = append(lines, "")
			}
		} else if ot, ok := resolveType(objectTypes, op.Endpoint, base); ok {
			lines = append(lines, pad+tui.HelpStyle.Render("Fields:"))
			lines = appendObjectTypeFields(lines, ot, argPad, objectTypes, op.Endpoint, 1)
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n")
}

func appendFragmentTypes(
	lines []string,
	possibleTypes []string,
	indent string,
	objectTypes map[string]graphql.ObjectType,
	endpoint string,
) []string {
	for i, pt := range possibleTypes {
		connector := "├─"
		if i == len(possibleTypes)-1 {
			connector = "└─"
		}
		label := fragmentPrefix + pt
		line := indent + tui.HelpStyle.Render(connector) + tui.KeySpace + label
		lines = append(lines, line)

		if ot, ok := resolveType(objectTypes, endpoint, pt); ok {
			childIndent := indent + tui.KeySpace + tui.KeySpace
			if i < len(possibleTypes)-1 {
				childIndent = indent + tui.HelpStyle.Render(treeBranch) + tui.KeySpace
			}
			lines = appendObjectTypeFields(lines, ot, childIndent, objectTypes, endpoint, 1)
		}
	}
	return lines
}

func resolveType[T any](types map[string]T, endpoint, baseType string) (T, bool) {
	if t, ok := types[ScopedTypeKey(endpoint, baseType)]; ok {
		return t, true
	}
	t, ok := types[baseType]
	return t, ok
}

const maxObjectTypeDepth = 3

func appendObjectTypeFields(
	lines []string,
	ot graphql.ObjectType,
	indent string,
	objectTypes map[string]graphql.ObjectType,
	endpoint string,
	depth int,
) []string {
	for i, f := range ot.Fields {
		connector := "├─"
		if i == len(ot.Fields)-1 {
			connector = "└─"
		}
		line := indent + tui.HelpStyle.Render(connector) +
			tui.KeySpace + f.Name +
			tui.KeySpace + tui.KeySpace + tui.HelpStyle.Render(f.Type)
		lines = append(lines, line)

		if depth < maxObjectTypeDepth {
			base := ExtractBaseType(f.Type)
			if nested, ok := resolveType(objectTypes, endpoint, base); ok {
				childIndent := indent + tui.KeySpace + tui.KeySpace
				if i < len(ot.Fields)-1 {
					childIndent = indent + tui.HelpStyle.Render(treeBranch) + tui.KeySpace
				}
				lines = appendObjectTypeFields(
					lines,
					nested,
					childIndent,
					objectTypes,
					endpoint,
					depth+1,
				)
			}
		}
	}
	return lines
}

const maxInputTypeDepth = 3

func appendInputTypeFields(
	lines []string,
	it graphql.InputType,
	indent string,
	inputTypes map[string]graphql.InputType,
	endpoint string,
	depth int,
) []string {
	for i, f := range it.Fields {
		connector := "├─"
		if i == len(it.Fields)-1 {
			connector = "└─"
		}
		line := indent + tui.HelpStyle.Render(connector) +
			tui.KeySpace + f.Name +
			tui.KeySpace + tui.KeySpace + tui.HelpStyle.Render(f.Type)
		lines = append(lines, line)

		if depth < maxInputTypeDepth {
			base := ExtractBaseType(f.Type)
			if nested, ok := resolveType(inputTypes, endpoint, base); ok {
				childIndent := indent + tui.KeySpace + tui.KeySpace
				if i < len(it.Fields)-1 {
					childIndent = indent + tui.HelpStyle.Render(treeBranch) + tui.KeySpace
				}
				lines = appendInputTypeFields(
					lines,
					nested,
					childIndent,
					inputTypes,
					endpoint,
					depth+1,
				)
			}
		}
	}
	return lines
}

func (m *Model) renderLeftContent() string {
	panelW := max(m.leftPanelWidth(), 1)
	badges := m.badgeCache
	if badges != "" {
		badges = truncateToWidth(badges, panelW)
	}
	searchStyle := tui.BorderStyle
	if m.focus.LeftFocused() {
		searchStyle = tui.FocusedInputStyle
	}
	search := searchStyle.
		Padding(0, 1).
		Width(max(panelW-2, 1)).
		Render(m.search.Model.View())

	content := ""
	if m.pickingEndpoints {
		content += endpointPickerTitle
	} else {
		content += fmt.Sprintf(operationFormat, len(m.filtered), len(m.operations))
	}
	statusLine := tui.HelpStyle.Render(truncateToWidth(content, panelW))

	var list string
	if m.ready {
		list = m.viewport.View()
	} else {
		content, _ := m.renderList()
		list = content
	}

	lines := make([]string, 0, 5)
	if badges != "" {
		lines = append(lines, badges)
	}
	lines = append(lines, search)
	if m.filterHint != "" {
		lines = append(lines, lipgloss.NewStyle().Width(panelW).Render(m.filterHint))
	}
	lines = append(lines, statusLine, list)
	return strings.Join(lines, "\n")
}

func shortenEndpoint(rawURL string) string {
	if strings.Contains(rawURL, "://") {
		if parsed, err := url.Parse(rawURL); err == nil && parsed.Host != "" {
			path := parsed.Path
			path = strings.TrimSuffix(path, "/graphql")
			path = strings.TrimSuffix(path, "/gql")
			path = strings.TrimSuffix(path, "/")
			if path == "" {
				return parsed.Host
			}
			if strings.HasPrefix(path, "/") {
				return parsed.Host + path
			}
			return parsed.Host + "/" + path
		}
	}
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")
	if q := strings.IndexByte(rawURL, '?'); q >= 0 {
		rawURL = rawURL[:q]
	}
	if h := strings.IndexByte(rawURL, '#'); h >= 0 {
		rawURL = rawURL[:h]
	}
	rawURL = strings.TrimSuffix(rawURL, "/graphql")
	rawURL = strings.TrimSuffix(rawURL, "/gql")
	rawURL = strings.TrimSuffix(rawURL, "/")
	return rawURL
}
