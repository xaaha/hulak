package utils

import (
	"slices"
	"testing"
	"text/template/parse"
)

// parserFieldNames is the oracle templateVarNames approximates: Go's own
// parser, first identifier of every field access, in source order.
func parserFieldNames(expr string) (names []string, parsed bool) {
	funcs := map[string]any{}
	for _, name := range templateFuncNames {
		funcs[name] = func(...any) string { return "" }
	}
	for _, name := range []string{"printf", "print", "eq", "index", "len", "not", "and", "or"} {
		funcs[name] = func(...any) string { return "" }
	}

	trees, err := parse.Parse("differential", expr, "{{", "}}", funcs)
	if err != nil {
		return nil, false
	}
	for _, tree := range trees {
		collectParserFields(tree.Root, &names)
	}
	return names, true
}

func collectParserFields(node parse.Node, out *[]string) {
	switch n := node.(type) {
	case *parse.FieldNode:
		*out = append(*out, n.Ident[0])
	case *parse.ChainNode:
		collectParserFields(n.Node, out)
	case *parse.ListNode:
		if n == nil {
			return
		}
		for _, child := range n.Nodes {
			collectParserFields(child, out)
		}
	case *parse.ActionNode:
		collectParserFields(n.Pipe, out)
	case *parse.PipeNode:
		if n == nil {
			return
		}
		for _, cmd := range n.Cmds {
			collectParserFields(cmd, out)
		}
	case *parse.CommandNode:
		for _, arg := range n.Args {
			collectParserFields(arg, out)
		}
	case *parse.IfNode:
		collectBranchFields(&n.BranchNode, out)
	case *parse.RangeNode:
		collectBranchFields(&n.BranchNode, out)
	case *parse.WithNode:
		collectBranchFields(&n.BranchNode, out)
	case *parse.TemplateNode:
		collectParserFields(n.Pipe, out)
	}
}

func collectBranchFields(n *parse.BranchNode, out *[]string) {
	collectParserFields(n.Pipe, out)
	collectParserFields(n.List, out)
	collectParserFields(n.ElseList, out)
}

func TestD3_2_TemplateVarNamesMatchesGoTemplateParser(t *testing.T) {
	exprs := []string{
		"{{.token}}",
		"{{ .token }}",
		"{{- .token }}",
		"Bearer {{.token}}",
		"{{.a}}{{.b}}",
		"{{if .flag}}yes{{end}}",
		"{{if .flag}}{{.yes}}{{else}}{{.no}}{{end}}",
		"{{range .items}}x{{end}}",
		"{{range .items}}{{.name}}{{end}}",
		"{{range $i, $v := .items}}{{$v}}{{end}}",
		"{{with .user}}{{.name}}{{end}}",
		`{{printf "%s" .token}}`,
		`{{printf "Bearer %s" .token}}`,
		"{{.token | printf \"%s\"}}",
		"{{$t := .token}}{{$t}}",
		"{{$t := .token}}{{$t.field}}",
		"{{.server.host}}",
		"{{ printf \"%s\"\n  .token }}",
		`{{eq .env "prod"}}`,
		"{{if eq .env \"prod\"}}{{.prodUrl}}{{end}}",
		"{{index .list 0}}",
		"{{len .items}}",
		"{{if not .flag}}y{{end}}",
		`{{basicAuth .user .pass}}`,
		`{{printf "a/*b" .token}}`,
		`{{printf "it's %s" .token}}`,
		"{{.a.b.c}}",
		"{{.Token}}",
		"{{.}}",
		"{{/* .token */}}",
		"{{- /* .token */ -}}",
		`{{getValueOf "token" ".secrets.json"}}`,
		"{{getValueOf \"k\" `.secrets.json`}}",
		`{{printf "%c" 'x'}}`,
		`{{getFile "collection/.hidden.gql"}}`,
		`{{os "GITHUB_TOKEN"}}`,
		`{{printf "%.2f" 1.5}}`,
		"no actions at all",
		"{{.名前}}",
		"{{.ключ}}",
		"{{.grüße}}",
		"{{.日本語_1}}",
		"{{._under}}",
		"{{.日本.token}}",
		`{{$日本 := "x"}}{{$日本.token}}`,
		"{{printf \"%s\" .名前}}",
		"{{if .флаг}}y{{end}}",
		`{{printf "a\"b .fake" 1}}`,
		`{{printf "a\"b .fake" .token}}`,
		`{{printf "%s%s" '\'' .token}}`,
		"{{printf \"%s\" `.fake\\` .token}}",
		`{{printf "trailing\\" .token}}`,
		"{{getFile `queries\\get.gql`}}",
	}

	for _, expr := range exprs {
		t.Run(expr, func(t *testing.T) {
			want, parsed := parserFieldNames(expr)
			if !parsed {
				t.Fatalf("corpus entry does not parse, so it compares nothing")
			}
			if got := templateVarNames(expr); !slices.Equal(got, want) {
				t.Errorf("templateVarNames = %v, Go parser = %v", got, want)
			}
		})
	}
}
