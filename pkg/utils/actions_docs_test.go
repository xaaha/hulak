package utils

import (
	"os"
	"strings"
	"testing"
)

func TestTemplateActionsDocumented_260(t *testing.T) {
	for _, doc := range []string{"../../docs/actions.md", "../mcp/authoring.md"} {
		content, err := os.ReadFile(doc)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range templateFuncNames {
			if !strings.Contains(string(content), "{{"+name+" ") {
				t.Errorf("%s does not show the %s action as {{%s ...}}", doc, name, name)
			}
		}
	}
}
