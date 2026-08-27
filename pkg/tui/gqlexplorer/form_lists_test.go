package gqlexplorer

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
)

func TestBuildDetailFormListArgStartsWithSingleInput(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "ids", Type: "[ID!]!"}},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil detail form")
		return
	}
	if df.argCount != 1 {
		t.Fatalf("expected 1 list input initially, got %d", df.argCount)
	}
	if !df.items[0].listItem {
		t.Fatal("expected first argument item to be marked as a list item")
	}
}

func TestBuildDetailFormListEnumStartsBlank(t *testing.T) {
	ep := "ep"
	enums := map[string]graphql.EnumType{
		"Status": {Name: "Status", Values: []graphql.EnumValue{{Name: "OPEN"}, {Name: "CLOSED"}}},
	}
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "statuses", Type: "[Status!]!"}},
	}
	df := buildDetailForm(op, nil, enums, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil detail form")
		return
	}
	if df.items[0].kind != formItemDropdown {
		t.Fatalf("expected dropdown list item, got %d", df.items[0].kind)
	}
	if got := df.items[0].Value(); got != "" {
		t.Fatalf("expected blank initial dropdown value, got %q", got)
	}
}

func TestBuildDetailFormListInputObjectStartsWithSingleGroup(t *testing.T) {
	ep := "ep"
	inputTypes := map[string]graphql.InputType{
		"UserFilter": {
			Name: "UserFilter",
			Fields: []graphql.InputField{
				{Name: "name", Type: "String"},
				{Name: "active", Type: "Boolean"},
			},
		},
	}
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "filters", Type: "[UserFilter!]!"}},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil detail form")
		return
	}
	if df.argCount != 2 {
		t.Fatalf("expected one input-object group with 2 rows, got %d", df.argCount)
	}
	if got := df.items[0].label; got != "filters[0].name" {
		t.Fatalf("unexpected first label %q", got)
	}
}

func TestListArgAddsFollowUpInputAfterValueEntered(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "ids", Type: "[ID!]!"}},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil detail form")
		return
	}

	df.items[0].input.Model.SetValue("a")
	df.syncListArgRows("ids")

	if df.argCount != 2 {
		t.Fatalf("expected second list input to be added, got %d arg items", df.argCount)
	}
	if !df.items[1].continued {
		t.Fatal("expected second list input to render as a continuation row")
	}
}

func TestListArgRemovesExtraTrailingBlankInputs(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "ids", Type: "[ID!]!"}},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil detail form")
		return
	}

	df.items[0].input.Model.SetValue("a")
	df.syncListArgRows("ids")
	df.items[1].input.Model.SetValue("b")
	df.syncListArgRows("ids")
	if df.argCount != 3 {
		t.Fatalf("expected 3 arg items after typing two values, got %d", df.argCount)
	}

	df.items[1].input.Model.SetValue("")
	df.items[2].input.Model.SetValue("")
	df.syncListArgRows("ids")

	if df.argCount != 2 {
		t.Fatalf("expected trailing blank list inputs to collapse, got %d", df.argCount)
	}
}

func TestListArgSpaceTogglesAllRows(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "ids", Type: "[ID]"}},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil detail form")
		return
	}

	df.items[0].input.Model.SetValue("a")
	df.syncListArgRows("ids")
	df.FocusCurrent()
	df.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})

	for i := 0; i < df.argCount; i++ {
		if !df.items[i].enabled {
			t.Fatalf("expected list row %d to be enabled", i)
		}
	}
}

func TestListInputObjectAddsFollowUpGroupAfterValueEntered(t *testing.T) {
	ep := "ep"
	inputTypes := map[string]graphql.InputType{
		"UserFilter": {
			Name: "UserFilter",
			Fields: []graphql.InputField{
				{Name: "name", Type: "String"},
				{Name: "active", Type: "Boolean"},
			},
		},
	}
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "filters", Type: "[UserFilter!]!"}},
	}
	df := buildDetailForm(op, inputTypes, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil detail form")
		return
	}

	df.items[0].input.Model.SetValue("alice")
	df.syncListArgRows("filters")

	if df.argCount != 4 {
		t.Fatalf("expected second object group to be added, got %d arg items", df.argCount)
	}
	if got := df.items[2].label; got != "filters[1].name" {
		t.Fatalf("unexpected second-group label %q", got)
	}
}

func TestEscExitsTextInputEditing(t *testing.T) {
	ep := "ep"
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "q", Type: "String!"}},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	df.FocusCurrent()

	df.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !df.items[0].input.Model.Focused() {
		t.Fatal("should be editing after Enter")
	}
	df.HandleKey(tea.KeyMsg{Type: tea.KeyEscape})
	if df.items[0].input.Model.Focused() {
		t.Fatal("Esc should exit editing")
	}
}

func TestDetailFormMouseClickFocusesTextInputAndEnablesArg(t *testing.T) {
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: "ep",
		Arguments: []graphql.Argument{{Name: "q", Type: "String"}},
	}
	df := buildDetailForm(op, nil, nil, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil detail form")
		return
	}

	z := tui.NewMouseZone()
	prefix := z.ID("detail")
	view, _ := df.ViewMarked(op, prefix, z.Mark)
	_ = tui.ScanMouseZones(view)

	x, y := waitForMouseZone(t, df.itemZoneID(prefix, 0))
	ok := df.HandleMouse(prefix, tea.MouseMsg{
		X:      x,
		Y:      y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	if !ok {
		t.Fatal("expected text input click to be handled")
	}
	if df.cursor != 0 {
		t.Fatalf("expected cursor on clicked item, got %d", df.cursor)
	}
	if !df.items[0].input.Model.Focused() {
		t.Fatal("expected text input to enter editing on click")
	}
	if !df.items[0].enabled {
		t.Fatal("expected arg to be enabled on click")
	}
}

func TestDetailFormMouseClickTogglesExpandableField(t *testing.T) {
	ep := "ep"
	objectTypes := map[string]graphql.ObjectType{
		ScopedTypeKey(ep, "Country"): {
			Name: "Country",
			Fields: []graphql.ObjectField{
				{Name: "code", Type: "ID!"},
				{Name: "language", Type: "Language"},
			},
		},
		ScopedTypeKey(ep, "Language"): {
			Name:   "Language",
			Fields: []graphql.ObjectField{{Name: "name", Type: "String"}},
		},
	}
	op := &UnifiedOperation{Name: "country", ReturnType: "Country", Endpoint: ep}
	df := buildDetailForm(op, nil, nil, objectTypes, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil detail form")
		return
	}

	z := tui.NewMouseZone()
	prefix := z.ID("detail")
	view, _ := df.ViewMarked(op, prefix, z.Mark)
	_ = tui.ScanMouseZones(view)

	x, y := waitForMouseZone(t, df.itemZoneID(prefix, 1))
	ok := df.HandleMouse(prefix, tea.MouseMsg{
		X:      x,
		Y:      y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	if !ok {
		t.Fatal("expected expandable field click to be handled")
	}
	if df.cursor != 1 {
		t.Fatalf("expected cursor on clicked expandable field, got %d", df.cursor)
	}
	if !df.items[1].toggle.Value {
		t.Fatal("expected expandable field to toggle on")
	}
	if df.Len() != 3 {
		t.Fatalf("expected child field to be inserted after click, got %d items", df.Len())
	}
}

func TestDetailFormMouseClickExpandsAndSelectsDropdown(t *testing.T) {
	ep := "ep"
	enums := map[string]graphql.EnumType{
		ScopedTypeKey(ep, "Status"): {
			Name:   "Status",
			Values: []graphql.EnumValue{{Name: "OPEN"}, {Name: "CLOSED"}, {Name: "PAUSED"}},
		},
	}
	op := &UnifiedOperation{
		Name: "Test", Type: TypeQuery, Endpoint: ep,
		Arguments: []graphql.Argument{{Name: "status", Type: "Status"}},
	}
	df := buildDetailForm(op, nil, enums, nil, nil, nil)
	if df == nil {
		t.Fatal("expected non-nil detail form")
		return
	}

	z := tui.NewMouseZone()
	prefix := z.ID("detail")
	view, _ := df.ViewMarked(op, prefix, z.Mark)
	_ = tui.ScanMouseZones(view)

	x, y := waitForMouseZone(t, df.itemZoneID(prefix, 0))
	ok := df.HandleMouse(prefix, tea.MouseMsg{
		X:      x,
		Y:      y,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	if !ok {
		t.Fatal("expected dropdown click to be handled")
	}
	if !df.items[0].dropdown.Expanded() {
		t.Fatal("expected first click to expand dropdown")
	}

	view, _ = df.ViewMarked(op, prefix, z.Mark)
	_ = tui.ScanMouseZones(view)
	x, y = waitForMouseZoneMinHeight(t, df.itemZoneID(prefix, 0), 2)
	ok = df.HandleMouse(prefix, tea.MouseMsg{
		X:      x,
		Y:      y + 2,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	})
	if !ok {
		t.Fatal("expected expanded dropdown option click to be handled")
	}
	if df.items[0].dropdown.Expanded() {
		t.Fatal("expected option click to collapse dropdown")
	}
	if got := df.items[0].dropdown.Value(); got != "PAUSED" {
		t.Fatalf("expected clicked option to be selected, got %q", got)
	}
	if !df.items[0].enabled {
		t.Fatal("expected dropdown arg to be enabled on click")
	}
}
