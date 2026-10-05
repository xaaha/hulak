package utils

import (
	"fmt"

	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
)

// ValidateSingleYAMLDoc returns an error when content holds more than one
// non-empty YAML document, so a leading or trailing `---` stays legal. name
// identifies the content in the error, which also carries the 1-based line the
// second document starts on.
func ValidateSingleYAMLDoc(name string, content []byte) error {
	file, err := parser.ParseBytes(content, 0)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", name, err)
	}
	seen := false
	for _, doc := range file.Docs {
		if emptyYAMLDoc(doc.Body) {
			continue
		}
		if seen {
			return fmt.Errorf(
				"%s: a request file must be a single YAML document, but a second document starts at line %d",
				name, yamlDocLine(doc),
			)
		}
		seen = true
	}
	return nil
}

func emptyYAMLDoc(body ast.Node) bool {
	switch node := body.(type) {
	case nil, *ast.DirectiveNode:
		return true
	case *ast.DocumentNode:
		return emptyYAMLDoc(node.Body)
	default:
		return false
	}
}

func yamlDocLine(doc *ast.DocumentNode) int {
	if doc.Start != nil {
		return doc.Start.Position.Line
	}
	return doc.Body.GetToken().Position.Line
}
