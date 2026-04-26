package userflags

import (
	"fmt"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/tui/gqlexplorer"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

// loadGraphQLOperations handles single file mode and directory mode along
// with unifying the operations into ExplorerData for the TUI. Returns a
// zero-valued ExplorerData with a nil refresh function when the user
// cancelled either the env picker or the spinner — that's not an error.
func loadGraphQLOperations(arg string, env string) (
	gqlexplorer.ExplorerData,
	gqlexplorer.RefreshFunc,
	[]string,
	error,
) {
	prepared, err := graphql.PrepareSchemaLoad(arg, env)
	if err != nil {
		return gqlexplorer.ExplorerData{}, nil, nil, fmt.Errorf("schema preparation: %w", err)
	}
	if prepared.Cancelled {
		return gqlexplorer.ExplorerData{}, nil, nil, nil
	}

	// load spinner while waiting
	raw, err := tui.RunWithSpinnerAfter("Fetching schemas...", func() (any, error) {
		return graphql.FetchPreparedSchemas(prepared)
	})
	if err != nil {
		return gqlexplorer.ExplorerData{}, nil, nil, fmt.Errorf("schema fetch: %w", err)
	}
	loadResult, ok := raw.(graphql.LoadResult)
	if !ok && raw != nil {
		return gqlexplorer.ExplorerData{}, nil, nil, fmt.Errorf(
			"internal error: unexpected result type %T from schema fetch", raw,
		)
	}

	if loadResult.Cancelled {
		return gqlexplorer.ExplorerData{}, nil, nil, nil
	}
	refreshFn := func() (gqlexplorer.RefreshPayload, error) {
		freshPrepared, err := graphql.PrepareSchemaLoad(arg, prepared.Env)
		if err != nil {
			return gqlexplorer.RefreshPayload{}, err
		}
		if freshPrepared.Cancelled {
			return gqlexplorer.RefreshPayload{}, nil
		}
		freshLoadResult, err := graphql.FetchPreparedSchemas(freshPrepared)
		if err != nil {
			return gqlexplorer.RefreshPayload{}, err
		}
		return gqlexplorer.RefreshPayload{
			Data:     explorerDataFromLoadResult(freshLoadResult, freshPrepared.Results),
			Warnings: freshLoadResult.Warnings,
		}, nil
	}

	return explorerDataFromLoadResult(loadResult, prepared.Results), refreshFn, loadResult.Warnings, nil
}

// explorerDataFromLoadResult converts schema fetch results and process
// results into the unified ExplorerData consumed by the TUI explorer.
func explorerDataFromLoadResult(loadResult graphql.LoadResult, processResults []graphql.ProcessResult) gqlexplorer.ExplorerData {
	data := gqlexplorer.ExplorerData{
		InputTypes:      make(map[string]graphql.InputType),
		EnumTypes:       make(map[string]graphql.EnumType),
		ObjectTypes:     make(map[string]graphql.ObjectType),
		UnionTypes:      make(map[string]graphql.UnionType),
		InterfaceTypes:  make(map[string]graphql.InterfaceType),
		APIInfos:        make(map[string]yamlparser.APIInfo),
		SchemaFilePaths: make(map[string]string),
	}

	for _, result := range processResults {
		if result.Error == nil {
			data.APIInfos[result.APIInfo.URL] = result.APIInfo
			data.SchemaFilePaths[result.APIInfo.URL] = result.FilePath
		}
	}

	for i := range loadResult.Endpoints {
		endpoint := &loadResult.Endpoints[i]
		data.Operations = append(data.Operations, gqlexplorer.CollectOperations(&endpoint.Schema, endpoint.URL)...)
		for k, v := range endpoint.Schema.InputTypes {
			data.InputTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
		for k, v := range endpoint.Schema.EnumTypes {
			data.EnumTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
		for k, v := range endpoint.Schema.ObjectTypes {
			data.ObjectTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
		for k, v := range endpoint.Schema.UnionTypes {
			data.UnionTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
		for k, v := range endpoint.Schema.InterfaceTypes {
			data.InterfaceTypes[gqlexplorer.ScopedTypeKey(endpoint.URL, k)] = v
		}
	}

	return data
}
