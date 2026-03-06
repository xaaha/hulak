package gqlexplorer

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/utils"
)

func TestTruncateToWidthTruncatesLongText(t *testing.T) {
	got := truncateToWidth("abcdefghijklmnopqrstuvwxyz", 12)
	if lipgloss.Width(got) != 12 {
		t.Fatalf("truncated width = %d, want 12", lipgloss.Width(got))
	}
	if got != "abcdefghi"+utils.Ellipsis {
		t.Fatalf("unexpected truncated string: %q", got)
	}
}

func TestTruncateToWidthReturnsInputWhenFits(t *testing.T) {
	const input = "hulak"
	if got := truncateToWidth(input, 12); got != input {
		t.Fatalf("width >= input should return input, got %q", got)
	}
}

func TestTruncateToWidthHandlesZeroAndNegativeWidths(t *testing.T) {
	if got := truncateToWidth("hello", 0); got != "" {
		t.Fatalf("width 0 got %q, want empty", got)
	}
	if got := truncateToWidth("hello", -1); got != "" {
		t.Fatalf("width -1 got %q, want empty", got)
	}
}

func TestRenderDetailShowsReturnTypeFields(t *testing.T) {
	op := &UnifiedOperation{
		Name:       "country",
		ReturnType: "Country",
		Endpoint:   "https://api.test/graphql",
		Arguments: []graphql.Argument{
			{Name: "code", Type: "ID!"},
		},
	}
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey("https://api.test/graphql", "Country"): {
			Name: "Country",
			Fields: []graphql.ObjectField{
				{Name: "awsRegion", Type: "String"},
				{Name: "capital", Type: "String"},
				{Name: "code", Type: "ID!"},
				{Name: "name", Type: "String"},
			},
		},
	}

	got := renderDetail(op, nil, objectTypes)

	for _, want := range []string{"country", "Country", "Fields:", "awsRegion", "capital", "code", "name"} {
		if !strings.Contains(got, want) {
			t.Errorf("renderDetail output missing %q", want)
		}
	}
	if !strings.Contains(got, "Arguments:") {
		t.Error("renderDetail should still show Arguments section")
	}
}

func TestRenderDetailNestedObjectTypes(t *testing.T) {
	op := &UnifiedOperation{
		Name:       "country",
		ReturnType: "Country",
		Endpoint:   "https://api.test/graphql",
	}
	ep := "https://api.test/graphql"
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "Country"): {
			Name: "Country",
			Fields: []graphql.ObjectField{
				{Name: "code", Type: "ID!"},
				{Name: "languages", Type: "[Language!]!"},
			},
		},
		ScopedTypeKey(ep, "Language"): {
			Name: "Language",
			Fields: []graphql.ObjectField{
				{Name: "code", Type: "ID!"},
				{Name: "name", Type: "String"},
			},
		},
	}

	got := renderDetail(op, nil, objectTypes)

	for _, want := range []string{"languages", "Language", "name"} {
		if !strings.Contains(got, want) {
			t.Errorf("renderDetail nested output missing %q", want)
		}
	}
}

func TestRenderDetailNoObjectTypeFallsBack(t *testing.T) {
	op := &UnifiedOperation{
		Name:       "hello",
		ReturnType: "String",
		Endpoint:   "https://api.test/graphql",
	}

	got := renderDetail(op, nil, nil)

	if strings.Contains(got, "Fields:") {
		t.Error("scalar return type should not show Fields section")
	}
	if !strings.Contains(got, "hello") {
		t.Error("should still show operation name")
	}
	if !strings.Contains(got, "String") {
		t.Error("should still show return type in header")
	}
}

func TestRenderDetailHeaderFormat(t *testing.T) {
	op := &UnifiedOperation{
		Name:       "country",
		ReturnType: "Country",
		Endpoint:   "https://api.test/graphql",
	}

	got := renderDetail(op, nil, nil)

	if !strings.Contains(got, "country") || !strings.Contains(got, "Country") {
		t.Error("header should contain operation name and return type")
	}
}

func TestAppendObjectTypeFieldsDepthCap(t *testing.T) {
	ep := "https://api.test/graphql"
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "A"): {
			Name:   "A",
			Fields: []graphql.ObjectField{{Name: "b", Type: "B"}},
		},
		ScopedTypeKey(ep, "B"): {
			Name:   "B",
			Fields: []graphql.ObjectField{{Name: "c", Type: "C"}},
		},
		ScopedTypeKey(ep, "C"): {
			Name:   "C",
			Fields: []graphql.ObjectField{{Name: "d", Type: "D"}},
		},
		ScopedTypeKey(ep, "D"): {
			Name:   "D",
			Fields: []graphql.ObjectField{{Name: "val", Type: "String"}},
		},
	}

	lines := appendObjectTypeFields(
		nil,
		objectTypes[ScopedTypeKey(ep, "A")],
		"  ",
		objectTypes,
		ep,
		1,
	)

	output := strings.Join(lines, "\n")
	if !strings.Contains(output, "b") {
		t.Error("depth 1 field 'b' should be present")
	}
	if !strings.Contains(output, "c") {
		t.Error("depth 2 field 'c' should be present")
	}
	if strings.Contains(output, "val") {
		t.Error("depth 4 field 'val' should be capped (maxObjectTypeDepth=3)")
	}
}
