package gqlexplorer

import (
	"strings"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
)

func (df *DetailForm) argRange(argName string) (int, int, bool) {
	start := -1
	end := -1
	for i := 0; i < df.argCount; i++ {
		if df.items[i].argName != argName {
			if start != -1 {
				break
			}
			continue
		}
		if start == -1 {
			start = i
		}
		end = i + 1
	}
	if start == -1 {
		return 0, 0, false
	}
	return start, end, true
}

func (df *DetailForm) argDefinition(argName string) (graphql.Argument, bool) {
	start, _, ok := df.argRange(argName)
	if !ok {
		return graphql.Argument{}, false
	}
	item := df.items[start]
	if !item.listItem {
		return graphql.Argument{Name: argName, Type: item.valueType}, true
	}
	return graphql.Argument{Name: argName, Type: item.listType}, true
}

func (df *DetailForm) argItems(argName string) []*formItem {
	var items []*formItem
	for i := 0; i < df.argCount; i++ {
		if df.items[i].argName == argName {
			items = append(items, &df.items[i])
		}
	}
	return items
}

func (df *DetailForm) setArgEnabled(argName string, enabled bool) {
	for i := 0; i < df.argCount; i++ {
		if df.items[i].argName == argName {
			df.items[i].enabled = enabled
		}
	}
}

func (df *DetailForm) syncListArgRows(argName string) {
	start, end, ok := df.argRange(argName)
	if !ok || !df.items[start].listItem {
		return
	}
	enabled := df.items[start].enabled
	arg, ok := df.argDefinition(argName)
	if !ok {
		return
	}

	df.syncGroupRange(start, end,
		func(item *formItem) int { return item.listGroup },
		func(group int) []formItem {
			return newListArgFormItems(arg, df.inputTypes, df.enumTypes, df.endpoint, group, enabled, df.nestedLists)
		},
	)
}

// boundaryGroupOf reports which element of a nested list boundary an item
// belongs to, derived from the item's own path rather than its (innermost
// only) listBoundary/boundaryGroup fields. This matches not just the
// boundary's own direct leaves but anything nested further inside one of
// its elements too (e.g. a list nested inside a nested list), which is what
// keeps a group's accounting correct even when that group has a field after
// a deeper nested list.
func boundaryGroupOf(item *formItem, fieldPath []varPathSeg) (int, bool) {
	if len(item.path) <= len(fieldPath) {
		return 0, false
	}
	for i, seg := range fieldPath {
		if item.path[i] != seg {
			return 0, false
		}
	}
	idxSeg := item.path[len(fieldPath)]
	if !idxSeg.isIndex {
		return 0, false
	}
	return idxSeg.index, true
}

// boundaryExtent finds the [start, end) span in df.items covering every item
// that belongs to a nested list boundary at any depth: its own direct
// leaves, and anything nested further inside one of its elements. Matching
// on the boundary's fieldPath (via boundaryGroupOf) rather than stopping at
// the first item whose innermost listBoundary differs means a group's own
// field that comes after a deeper nested list is still found, instead of
// being silently dropped once the scan reaches that inner list's items.
func (df *DetailForm) boundaryExtent(fieldPath []varPathSeg) (int, int, bool) {
	start := -1
	end := -1
	for i := 0; i < df.argCount; i++ {
		if _, ok := boundaryGroupOf(&df.items[i], fieldPath); !ok {
			continue
		}
		if start == -1 {
			start = i
		}
		end = i + 1
	}
	if start == -1 {
		return 0, 0, false
	}
	return start, end, true
}

// syncNestedListBoundary grows or shrinks the elements of a list-of-input-
// object field nested inside another argument, using the same grow/shrink
// rule as syncListArgRows but scoped to the field's own boundary rather than
// the whole argument.
func (df *DetailForm) syncNestedListBoundary(boundary string) {
	spec, ok := df.nestedLists[boundary]
	if !ok {
		return
	}
	start, end, ok := df.boundaryExtent(spec.fieldPath)
	if !ok {
		return
	}

	df.syncGroupRange(start, end,
		func(item *formItem) int {
			group, _ := boundaryGroupOf(item, spec.fieldPath)
			return group
		},
		func(group int) []formItem {
			return expandNestedListGroup(boundary, &spec, group, df.inputTypes, df.enumTypes, df.endpoint, df.nestedLists)
		},
	)
}

// syncGroupRange grows or shrinks a contiguous [start, end) run of argument
// items so it holds exactly one trailing empty group after the last group
// with a meaningful value (or a single empty group if none has one),
// regenerating newly added groups via generate. groupOf reports which group
// an item belongs to; shared by syncListArgRows and syncNestedListBoundary,
// which differ only in how items are grouped and regenerated.
func (df *DetailForm) syncGroupRange(
	start, end int,
	groupOf func(*formItem) int,
	generate func(group int) []formItem,
) {
	lastNonEmptyGroup := -1
	currentGroups := 0
	for i := start; i < end; {
		group := groupOf(&df.items[i])
		groupNonEmpty := false
		for i < end && groupOf(&df.items[i]) == group {
			if hasMeaningfulListValue(&df.items[i]) {
				groupNonEmpty = true
			}
			i++
		}
		if groupNonEmpty {
			lastNonEmptyGroup = group
		}
		currentGroups++
	}

	desiredGroups := 1
	if lastNonEmptyGroup >= 0 {
		desiredGroups = lastNonEmptyGroup + 2
	}

	if desiredGroups < currentGroups {
		cut := start
		for cut < end && groupOf(&df.items[cut]) < desiredGroups {
			cut++
		}
		removed := end - cut
		df.items = append(df.items[:cut], df.items[end:]...)
		df.argCount -= removed
		if df.cursor >= end {
			df.cursor -= removed
		} else if df.cursor >= cut {
			df.cursor = max(cut-1, 0)
		}
		end = cut
		currentGroups = desiredGroups
	}

	if desiredGroups > currentGroups {
		insertAt := end
		for group := currentGroups; group < desiredGroups; group++ {
			next := generate(group)
			df.items = append(df.items[:insertAt], append(next, df.items[insertAt:]...)...)
			df.argCount += len(next)
			insertAt += len(next)
		}
	}

	df.FocusCurrent()
}

func hasMeaningfulListValue(item *formItem) bool {
	if item == nil {
		return false
	}
	switch item.kind {
	case formItemTextInput:
		return strings.TrimSpace(item.Value()) != ""
	case formItemDropdown:
		return strings.TrimSpace(item.Value()) != ""
	case formItemToggle:
		return item.enabled
	default:
		return false
	}
}

func listDropdown(d tui.Dropdown) tui.Dropdown {
	options := make([]string, 0, len(d.Options)+1)
	options = append(options, "")
	options = append(options, d.Options...)
	return tui.NewDropdown(d.Label, options, 0)
}
