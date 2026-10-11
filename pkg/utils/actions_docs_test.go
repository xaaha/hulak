package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/goccy/go-yaml/parser"
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

func TestDocsYAMLExamplesParse_278(t *testing.T) {
	docs, err := filepath.Glob("../../docs/*.md")
	if err != nil {
		t.Fatal(err)
	}
	block := regexp.MustCompile("(?s)```ya?ml\n(.*?)```")
	for _, doc := range append(docs, "../../README.md", "../mcp/authoring.md") {
		content, err := os.ReadFile(doc)
		if err != nil {
			t.Fatal(err)
		}
		for i, m := range block.FindAllSubmatch(content, -1) {
			if _, err := parser.ParseBytes(m[1], 0); err != nil {
				t.Errorf("%s yaml block %d does not parse: %v", doc, i+1, err)
			}
		}
	}
}
