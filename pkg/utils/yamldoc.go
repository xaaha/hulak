package utils

import (
	"fmt"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// ValidateSingleYAMLDoc returns an error when content holds more than one YAML
// document. name identifies the content in the error, which also carries the
// 1-based line the second document starts on.
func ValidateSingleYAMLDoc(name string, content []byte) error {
	file, err := parser.ParseBytes(content, 0)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", name, err)
	}
	if len(file.Docs) < 2 {
		return nil
	}
	return fmt.Errorf(
		"%s: a request file must be a single YAML document, but a second document starts at line %d",
		name, yamlDocLine(file.Docs[1]),
	)
}

func yamlDocLine(doc *ast.DocumentNode) int {
	if doc.Start != nil {
		return doc.Start.Position.Line
	}
	if inner, ok := doc.Body.(*ast.DocumentNode); ok {
		if inner == nil {
			return 0
		}
		return yamlDocLine(inner)
	}
	if doc.Body != nil {
		return doc.Body.GetToken().Position.Line
	}
	return 0
}
