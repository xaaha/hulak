package gqlexplorer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xaaha/hulak/pkg/features/graphql"
)

func newListArgFormItems(
	arg graphql.Argument,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	endpoint string,
	group int,
	enabled bool,
	nestedLists map[string]nestedListSpec,
) []formItem {
	itemType := ExtractListItemType(arg.Type)
	if it, ok := resolveType(inputTypes, endpoint, ExtractBaseType(itemType)); ok {
		ctx := inputExpandCtx{
			inputTypes:     inputTypes,
			enumTypes:      enumTypes,
			endpoint:       endpoint,
			argName:        arg.Name,
			parentPath:     []varPathSeg{{index: group, isIndex: true}},
			depth:          0,
			parentEnabled:  false,
			inTopLevelList: true,
			listType:       arg.Type,
			listGroup:      group,
			visited:        map[string]bool{ExtractBaseType(itemType): true},
			nestedLists:    nestedLists,
		}
		items := make([]formItem, 0, len(it.Fields))
		for _, field := range it.Fields {
			items = append(items, expandInputField(field, &ctx)...)
		}
		return items
	}

	fi := newTypedFormItem(arg.Name, itemType, enumTypes, endpoint)
	fi.argName = arg.Name
	fi.path = []varPathSeg{{index: group, isIndex: true}}
	fi.listType = arg.Type
	fi.listItem = true
	fi.listGroup = group
	fi.valueType = itemType
	fi.enabled = enabled
	fi.required = strings.HasSuffix(arg.Type, "!")
	fi.typeHint = arg.Type
	if fi.kind == formItemDropdown {
		fi.dropdown = listDropdown(fi.dropdown)
		fi.enabled = enabled
		fi.required = strings.HasSuffix(arg.Type, "!")
		fi.label = fmt.Sprintf("%s[%d]", arg.Name, group)
		return []formItem{fi}
	}
	if group > 0 {
		fi.continued = true
		fi.required = false
	}
	return []formItem{fi}
}

func newArgFormItem(
	arg graphql.Argument,
	enumTypes map[string]graphql.EnumType,
	endpoint string,
) formItem {
	return newTypedFormItem(arg.Name, arg.Type, enumTypes, endpoint)
}

// maxInputFormDepth caps how deep the form expands nested input objects. The
// visited set already terminates cyclic input types; this is a safety net for
// very deep but acyclic schemas so the form stays usable.
const maxInputFormDepth = 8

// inputExpandCtx carries the state threaded through recursive expansion of an
// input-object argument (or list element) into individual form items.
type inputExpandCtx struct {
	inputTypes map[string]graphql.InputType
	enumTypes  map[string]graphql.EnumType
	endpoint   string
	argName    string
	// parentPath is the path from the argument root to the object that owns the
	// field currently being expanded.
	parentPath []varPathSeg
	depth      int
	// parentEnabled reports whether the owning object contributes by default,
	// so an optional parent disables its children (mirroring top-level args).
	parentEnabled bool
	// inTopLevelList marks that this item belongs to a top-level list argument,
	// which drives dynamic continuation rows and whole-list enable/disable.
	inTopLevelList bool
	listType       string
	listGroup      int
	visited        map[string]bool

	// nestedLists accumulates the specs needed to regenerate additional
	// elements of list-of-input-object fields nested inside this argument,
	// keyed by pathKey. It is the same underlying map for the whole
	// expansion of one top-level argument, so a spec registered deep in the
	// recursion is visible to the DetailForm afterward.
	nestedLists map[string]nestedListSpec
	// nestedListBoundary/nestedListGroup identify which nested list element
	// the field currently being expanded belongs to, if any. They are
	// inherited unchanged through intermediate non-list objects so a deeply
	// nested leaf still resolves to the list element that contains it.
	nestedListBoundary string
	nestedListGroup    int
}

// nestedListSpec captures the context present when a list-of-input-object
// field nested inside another argument is first encountered, so additional
// elements can be regenerated on demand as the user fills the previous one
// in (mirroring newListArgFormItems for top-level list arguments).
//
// inTopLevelList/listType/listGroup carry the ambient top-level list context
// (if any) that was active at that point, so a list field nested inside a
// top-level list argument's element keeps contributing to the outer list's
// own group accounting (see syncListArgRows) in addition to this field's own
// nested boundary. Dropping them here would make every regenerated element's
// leaves report listGroup 0 regardless of which outer element they actually
// belong to, corrupting the outer list's grow/shrink scan.
type nestedListSpec struct {
	argName        string
	fieldPath      []varPathSeg
	elementBase    string
	depth          int
	visited        map[string]bool
	inTopLevelList bool
	listType       string
	listGroup      int
}

// expandNestedListGroup builds the form items for one element of a
// list-of-input-object field nested inside another argument, resolving the
// element's input type first. Used by syncNestedListBoundary, which only has
// the spec (not an already-resolved graphql.InputType) on hand.
func expandNestedListGroup(
	boundary string,
	spec *nestedListSpec,
	group int,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	endpoint string,
	nestedLists map[string]nestedListSpec,
) []formItem {
	it, ok := resolveType(inputTypes, endpoint, spec.elementBase)
	if !ok || len(it.Fields) == 0 {
		return nil
	}
	return expandNestedListElement(it, boundary, spec, group, inputTypes, enumTypes, endpoint, nestedLists)
}

// expandNestedListElement builds the form items for one element of a
// list-of-input-object field, given the element's already-resolved input
// type.
func expandNestedListElement(
	it graphql.InputType,
	boundary string,
	spec *nestedListSpec,
	group int,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	endpoint string,
	nestedLists map[string]nestedListSpec,
) []formItem {
	child := inputExpandCtx{
		inputTypes:         inputTypes,
		enumTypes:          enumTypes,
		endpoint:           endpoint,
		argName:            spec.argName,
		parentPath:         appendPathSeg(spec.fieldPath, varPathSeg{index: group, isIndex: true}),
		depth:              spec.depth,
		parentEnabled:      false,
		inTopLevelList:     spec.inTopLevelList,
		listType:           spec.listType,
		listGroup:          spec.listGroup,
		visited:            spec.visited,
		nestedLists:        nestedLists,
		nestedListBoundary: boundary,
		nestedListGroup:    group,
	}
	items := make([]formItem, 0, len(it.Fields))
	for _, sub := range it.Fields {
		items = append(items, expandInputField(sub, &child)...)
	}
	return items
}

// expandInputField turns one input field into form items. Scalar, enum and
// Boolean fields become a single leaf item. Fields whose type is an input
// object (or a list of input objects) recurse into their own fields, to
// arbitrary depth, guarded by a visited set (cycles) and a depth cap.
func expandInputField(field graphql.InputField, ctx *inputExpandCtx) []formItem {
	fieldPath := appendPathSeg(ctx.parentPath, varPathSeg{key: field.Name})
	fieldRequired := strings.HasSuffix(field.Type, "!")
	base := ExtractBaseType(field.Type)

	it, isInput := resolveType(ctx.inputTypes, ctx.endpoint, base)
	if isInput && !ctx.visited[base] && ctx.depth < maxInputFormDepth && len(it.Fields) > 0 {
		if IsListType(field.Type) {
			boundary := pathKey(ctx.argName, fieldPath)
			spec := nestedListSpec{
				argName:        ctx.argName,
				fieldPath:      fieldPath,
				elementBase:    base,
				depth:          ctx.depth + 1,
				visited:        cloneVisited(ctx.visited, base),
				inTopLevelList: ctx.inTopLevelList,
				listType:       ctx.listType,
				listGroup:      ctx.listGroup,
			}
			if ctx.nestedLists != nil {
				if _, exists := ctx.nestedLists[boundary]; !exists {
					ctx.nestedLists[boundary] = spec
				}
			}
			return expandNestedListElement(
				it, boundary, &spec, 0, ctx.inputTypes, ctx.enumTypes, ctx.endpoint, ctx.nestedLists,
			)
		}

		child := inputExpandCtx{
			inputTypes:         ctx.inputTypes,
			enumTypes:          ctx.enumTypes,
			endpoint:           ctx.endpoint,
			argName:            ctx.argName,
			parentPath:         fieldPath,
			depth:              ctx.depth + 1,
			parentEnabled:      ctx.parentEnabled && fieldRequired,
			inTopLevelList:     ctx.inTopLevelList,
			listType:           ctx.listType,
			listGroup:          ctx.listGroup,
			visited:            cloneVisited(ctx.visited, base),
			nestedLists:        ctx.nestedLists,
			nestedListBoundary: ctx.nestedListBoundary,
			nestedListGroup:    ctx.nestedListGroup,
		}
		items := make([]formItem, 0, len(it.Fields))
		for _, sub := range it.Fields {
			items = append(items, expandInputField(sub, &child)...)
		}
		return items
	}

	return []formItem{newLeafFormItem(field, fieldPath, fieldRequired, ctx)}
}

func newLeafFormItem(
	field graphql.InputField,
	fieldPath []varPathSeg,
	fieldRequired bool,
	ctx *inputExpandCtx,
) formItem {
	fi := newTypedFormItem(field.Name, field.Type, ctx.enumTypes, ctx.endpoint)
	fi.argName = ctx.argName
	fi.path = fieldPath
	fi.depth = ctx.depth
	fi.required = fieldRequired
	fi.label = labelForPath(ctx.argName, fieldPath)
	if ctx.inTopLevelList {
		fi.listType = ctx.listType
		fi.listItem = true
		fi.listGroup = ctx.listGroup
		fi.enabled = false
	} else {
		fi.enabled = ctx.parentEnabled && fieldRequired
	}
	fi.listBoundary = ctx.nestedListBoundary
	fi.boundaryGroup = ctx.nestedListGroup
	return fi
}

// labelForPath builds a display label for a nested field. Direct fields of a
// top-level input-object argument keep an empty label so only the field name is
// shown (matching the original single-level behavior); deeper or list-nested
// fields get a dotted/bracketed path like arg[0].field.subfield.
func labelForPath(argName string, path []varPathSeg) string {
	if len(path) == 1 && !path[0].isIndex {
		return ""
	}
	return pathKey(argName, path)
}

// pathKey renders an argument name and path as a flat string, used both for
// display labels (via labelForPath) and as the map key identifying a nested
// list boundary.
func pathKey(argName string, path []varPathSeg) string {
	var b strings.Builder
	b.WriteString(argName)
	for _, seg := range path {
		if seg.isIndex {
			b.WriteString("[")
			b.WriteString(strconv.Itoa(seg.index))
			b.WriteString("]")
		} else {
			b.WriteString(".")
			b.WriteString(seg.key)
		}
	}
	return b.String()
}

func appendPathSeg(path []varPathSeg, seg varPathSeg) []varPathSeg {
	out := make([]varPathSeg, len(path)+1)
	copy(out, path)
	out[len(path)] = seg
	return out
}

func cloneVisited(visited map[string]bool, add string) map[string]bool {
	out := make(map[string]bool, len(visited)+1)
	for k := range visited {
		out[k] = true
	}
	out[add] = true
	return out
}
