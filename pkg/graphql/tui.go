// Package graphql provides GraphQL schema exploration TUI
package graphql

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View represents different screens in the TUI
type View int

const (
	ViewEnvSelect View = iota
	ViewEndpointInput
	ViewSchemaLoading
	ViewSchemaExplorer
	ViewQueryBuilder
	ViewSavedQueries
)

// model represents the application state
type model struct {
	config      ExplorerConfig
	currentView View

	// Environment selection
	envList     list.Model
	selectedEnv string

	// Endpoint input
	endpointInput textinput.Model
	endpoint      string

	// Schema data
	schema        *GraphQLSchema
	schemaLoading bool
	schemaError   error

	// Schema explorer
	explorerList list.Model
	selectedType string
	typeDetails  string

	// Query builder
	queryInput  textinput.Model
	queryText   string
	queryResult string

	// Saved queries
	savedQueries []SavedQuery
	queriesList  list.Model

	// UI state
	width  int
	height int
	err    error
}

// SavedQuery represents a saved GraphQL query
type SavedQuery struct {
	Name        string
	Description string
	Query       string
	Variables   map[string]any
	FilePath    string
}

// GraphQLSchema represents the introspected schema
type GraphQLSchema struct {
	Queries   []SchemaType
	Mutations []SchemaType
	Types     []SchemaType
}

// SchemaType represents a type in the schema
type SchemaType struct {
	Name        string
	Description string
	Fields      []SchemaField
	Kind        string
}

// SchemaField represents a field in a type
type SchemaField struct {
	Name        string
	Description string
	Type        string
	Args        []SchemaArg
	IsRequired  bool
}

// SchemaArg represents an argument to a field
type SchemaArg struct {
	Name        string
	Type        string
	Description string
	IsRequired  bool
}

// initialModel creates the initial model state
func initialModel(config ExplorerConfig) model {
	// Create environment list
	items := make([]list.Item, len(config.EnvFiles))
	for i, env := range config.EnvFiles {
		items[i] = listItem{
			title: env,
			desc:  fmt.Sprintf("Environment: %s", env),
		}
	}

	envList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	envList.Title = "Select Environment"

	// Create endpoint input
	endpointInput := textinput.New()
	endpointInput.Placeholder = "https://api.example.com/graphql"
	endpointInput.Focus()
	endpointInput.CharLimit = 256
	endpointInput.Width = 50

	return model{
		config:        config,
		currentView:   ViewEnvSelect,
		envList:       envList,
		endpointInput: endpointInput,
	}
}

// listItem implements list.Item interface
type listItem struct {
	title string
	desc  string
}

func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.desc }
func (i listItem) FilterValue() string { return i.title }

// Init initializes the model
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.envList.SetSize(msg.Width-4, msg.Height-4)
		return m, nil

	case schemaLoadedMsg:
		m.schemaLoading = false
		if msg.err != nil {
			m.schemaError = msg.err
			return m, nil
		}
		m.schema = msg.schema
		m.currentView = ViewSchemaExplorer
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			// Go back to previous view
			if m.currentView > ViewEnvSelect {
				m.currentView--
			}
			return m, nil
		}

		// Handle view-specific keys
		return m.handleViewUpdate(msg)
	}

	return m, nil
}

// handleViewUpdate handles updates for specific views
func (m model) handleViewUpdate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.currentView {
	case ViewEnvSelect:
		m.envList, cmd = m.envList.Update(msg)
		if msg.String() == "enter" {
			if item, ok := m.envList.SelectedItem().(listItem); ok {
				m.selectedEnv = strings.TrimSuffix(item.title, ".env")
				m.currentView = ViewEndpointInput
			}
		}

	case ViewEndpointInput:
		m.endpointInput, cmd = m.endpointInput.Update(msg)
		if msg.String() == "enter" {
			m.endpoint = m.endpointInput.Value()
			if m.endpoint != "" {
				m.currentView = ViewSchemaLoading
				return m, loadSchemaCmd(m.endpoint, m.selectedEnv)
			}
		}

	case ViewSchemaExplorer:
		m.explorerList, cmd = m.explorerList.Update(msg)
		if msg.String() == "enter" {
			// Handle selection
		}

	case ViewQueryBuilder:
		m.queryInput, cmd = m.queryInput.Update(msg)

	case ViewSavedQueries:
		m.queriesList, cmd = m.queriesList.Update(msg)
	}

	return m, cmd
}

// View renders the current view
func (m model) View() string {
	switch m.currentView {
	case ViewEnvSelect:
		return m.renderEnvSelect()
	case ViewEndpointInput:
		return m.renderEndpointInput()
	case ViewSchemaLoading:
		return m.renderSchemaLoading()
	case ViewSchemaExplorer:
		return m.renderSchemaExplorer()
	case ViewQueryBuilder:
		return m.renderQueryBuilder()
	case ViewSavedQueries:
		return m.renderSavedQueries()
	default:
		return "Unknown view"
	}
}

// Render methods for each view
func (m model) renderEnvSelect() string {
	style := lipgloss.NewStyle().
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		titleStyle.Render("Hulak GraphQL Explorer"),
		m.envList.View(),
		helpStyle.Render("↑/↓: navigate • enter: select • q: quit"),
	)

	return style.Render(content)
}

func (m model) renderEndpointInput() string {
	content := fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s\n\n%s",
		titleStyle.Render("Enter GraphQL Endpoint"),
		fmt.Sprintf("Selected environment: %s", successStyle.Render(m.selectedEnv)),
		"",
		m.endpointInput.View(),
		helpStyle.Render("enter: continue • esc: back • q: quit"),
	)

	return lipgloss.NewStyle().Padding(2).Render(content)
}

func (m model) renderSchemaLoading() string {
	if m.schemaError != nil {
		return errorStyle.Render(fmt.Sprintf("Error loading schema: %v", m.schemaError))
	}

	return lipgloss.NewStyle().
		Padding(4).
		Render(fmt.Sprintf("%s\n\nLoading schema from %s...",
			titleStyle.Render("Loading"),
			m.endpoint))
}

func (m model) renderSchemaExplorer() string {
	if m.schema == nil {
		return "No schema loaded"
	}

	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s queries • %s mutations • %s types\n\n%s",
		titleStyle.Render("Schema Explorer"),
		fmt.Sprintf("Endpoint: %s", m.endpoint),
		successStyle.Render(fmt.Sprintf("%d", len(m.schema.Queries))),
		successStyle.Render(fmt.Sprintf("%d", len(m.schema.Mutations))),
		successStyle.Render(fmt.Sprintf("%d", len(m.schema.Types))),
		helpStyle.Render("↑/↓: navigate • enter: view details • b: query builder • s: saved queries • q: quit"),
	)

	return lipgloss.NewStyle().Padding(2).Render(content)
}

func (m model) renderQueryBuilder() string {
	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		titleStyle.Render("Query Builder"),
		m.queryInput.View(),
		helpStyle.Render("ctrl+s: save • ctrl+r: run • esc: back • q: quit"),
	)

	return lipgloss.NewStyle().Padding(2).Render(content)
}

func (m model) renderSavedQueries() string {
	content := fmt.Sprintf(
		"%s\n\n%s queries saved\n\n%s",
		titleStyle.Render("Saved Queries"),
		successStyle.Render(fmt.Sprintf("%d", len(m.savedQueries))),
		helpStyle.Render("↑/↓: navigate • enter: load • d: delete • esc: back • q: quit"),
	)

	return lipgloss.NewStyle().Padding(2).Render(content)
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			MarginBottom(1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// loadSchemaCmd is a command to load the GraphQL schema
func loadSchemaCmd(endpoint, env string) tea.Cmd {
	return func() tea.Msg {
		// Load environment variables for headers if needed
		headers := make(map[string]string)

		// Perform introspection
		schema, err := IntrospectSchema(endpoint, headers)
		return schemaLoadedMsg{
			endpoint: endpoint,
			schema:   schema,
			err:      err,
		}
	}
}

// schemaLoadedMsg indicates the schema has been loaded
type schemaLoadedMsg struct {
	endpoint string
	schema   *GraphQLSchema
	err      error
}
