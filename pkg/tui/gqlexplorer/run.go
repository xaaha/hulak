package gqlexplorer

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func RunExplorer(
	operations []UnifiedOperation,
	inputTypes map[string]graphql.InputType,
	enumTypes map[string]graphql.EnumType,
	objectTypes map[string]graphql.ObjectType,
	unionTypes map[string]graphql.UnionType,
	interfaceTypes map[string]graphql.InterfaceType,
) error {
	model := NewModel(
		operations,
		inputTypes,
		enumTypes,
		objectTypes,
		unionTypes,
		interfaceTypes,
		make(map[string]yamlparser.APIInfo),
	)
	return runExplorerModel(&model)
}

func RunExplorerWithRefresh(
	data *ExplorerData,
	refreshFn RefreshFunc,
	initialWarnings []string,
) error {
	model := NewModel(
		data.Operations,
		data.InputTypes,
		data.EnumTypes,
		data.ObjectTypes,
		data.UnionTypes,
		data.InterfaceTypes,
		data.APIInfos,
	)
	model.schemaFilePaths = data.SchemaFilePaths
	model.SetRefresh(refreshFn)
	model.SetInitialWarnings(initialWarnings)
	return runExplorerModel(&model)
}

func runExplorerModel(model *Model) error {
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
