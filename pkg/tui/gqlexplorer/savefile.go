package gqlexplorer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func (m *Model) saveQueryAndVariables() tea.Cmd {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	op := &m.filtered[m.cursor]

	dir := m.responseSaveDir(op.Endpoint)
	if dir == "" {
		return m.enqueueNotification(tui.NotificationError, "Cannot determine save directory")
	}

	query := BuildQueryString(op, m.detailForm)
	if query == "" {
		return m.enqueueNotification(tui.NotificationError, "Empty query")
	}

	var sb strings.Builder
	sb.WriteString(query)

	vars := BuildVariablesString(op, m.detailForm)
	if vars != "" {
		sb.WriteString("\n\n# Variables:\n")
		for _, line := range strings.Split(vars, "\n") {
			sb.WriteString("# ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	fileName := op.Name + ".gql"
	fullPath := filepath.Join(dir, fileName)

	if err := os.WriteFile(fullPath, []byte(sb.String()), utils.FilePer); err != nil {
		return m.enqueueNotification(tui.NotificationError, "Save failed: "+err.Error())
	}
	return m.enqueueNotification(tui.NotificationInfo, "Saved "+relativePath(fullPath))
}

func (m *Model) createHulakRequestFile() tea.Cmd {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	op := &m.filtered[m.cursor]

	dir := m.responseSaveDir(op.Endpoint)
	if dir == "" {
		return m.enqueueNotification(tui.NotificationError, "Cannot determine save directory")
	}

	query := BuildQueryString(op, m.detailForm)
	if query == "" {
		return m.enqueueNotification(tui.NotificationError, "Empty query")
	}

	gqlFileName := op.Name + ".gql"
	gqlPath := filepath.Join(dir, gqlFileName)
	if err := os.WriteFile(gqlPath, []byte(query), utils.FilePer); err != nil {
		return m.enqueueNotification(tui.NotificationError, "Save .gql failed: "+err.Error())
	}

	gqlRelPath := relativePath(gqlPath)
	yamlContent := buildHkYaml(op, m.detailForm, m.apiInfos, gqlRelPath)

	yamlFileName := op.Name + ".hk.yaml"
	yamlPath := filepath.Join(dir, yamlFileName)
	if fileExists(yamlPath) {
		stamp := time.Now().Format("20060102-150405")
		yamlFileName = op.Name + "-" + stamp + ".hk.yaml"
		yamlPath = filepath.Join(dir, yamlFileName)
	}

	if err := os.WriteFile(yamlPath, []byte(yamlContent), utils.FilePer); err != nil {
		return m.enqueueNotification(tui.NotificationError, "Save failed: "+err.Error())
	}

	return m.enqueueNotification(
		tui.NotificationInfo,
		fmt.Sprintf("Created %s and %s", relativePath(yamlPath), gqlRelPath),
	)
}

func buildHkYaml(
	op *UnifiedOperation,
	df *DetailForm,
	apiInfos map[string]yamlparser.APIInfo,
	gqlRelPath string,
) string {
	var sb strings.Builder
	sb.WriteString("---\nmethod: POST\n")
	fmt.Fprintf(&sb, "url: %q\n", op.Endpoint)

	info, hasInfo := apiInfos[op.Endpoint]
	if hasInfo && len(info.Headers) > 0 {
		sb.WriteString("headers:\n")
		for _, k := range sortedKeys(info.Headers) {
			fmt.Fprintf(&sb, "  %s: %s\n", k, info.Headers[k])
		}
	} else {
		sb.WriteString("headers:\n  Content-Type: application/json\n")
	}

	sb.WriteString("body:\n  graphql:\n")
	fmt.Fprintf(&sb, "    query: '{{getFile %q}}'\n", gqlRelPath)

	varsMap := BuildVariablesMap(op, df)
	if len(varsMap) > 0 {
		sb.WriteString("    variables:\n")
		for _, arg := range op.Arguments {
			v, ok := varsMap[arg.Name]
			if !ok {
				continue
			}
			fmt.Fprintf(&sb, "      %s: %s\n", arg.Name, yamlScalar(v))
		}
	} else if len(op.Arguments) > 0 {
		sb.WriteString("    variables:\n")
		for _, arg := range op.Arguments {
			fmt.Fprintf(&sb, "      %s: \"\"\n", arg.Name)
		}
	}

	return sb.String()
}

func yamlScalar(v any) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", val)
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%d", int(val))
		}
		return fmt.Sprintf("%g", val)
	case nil:
		return "null"
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}

func relativePath(absPath string) string {
	if wd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(wd, absPath); err == nil {
			return rel
		}
	}
	return absPath
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
