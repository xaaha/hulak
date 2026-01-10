# GraphQL Schema Explorer TUI - Implementation Plan

## Overview

A minimal TUI for exploring GraphQL schemas across multiple endpoints, building queries interactively, executing them, and saving them as `{operationName}.gql` files.

**Core workflow**: `hulak gql .` → Filter → Select fields → Fill variables → Execute → Save

---

## Design Summary

### Key Features

- **Multi-endpoint aggregation**: Search across ALL GraphQL endpoints in one view
- **Schema-driven**: Introspection required, types guide the UI
- **Hybrid layout**: Dual-pane for browsing, focused views for input
- **File-based output**: Save queries as `{operationName}.gql`
- **Call-the-operation**: from the tui itself.

### Views

1. **Loading** - Schema fetch progress
2. **Schema Browser** - Unified operation list with preview
3. **Query Builder** - Field selection with checkboxes
4. **Variable Input** - Type-aware form
5. **Response** - Inline JSON display

---

## Directory Structure

```
pkg/
└── tui/
    ├── keys.go                 # Shared key bindings and helpers (NEVER use 'q' for quit!)
    ├── styles.go               # Shared styles and colors
    ├── tui.go                  # Legacy file (to be removed)
    ├── envselect/              # Phase 0: Environment selector
    │   ├── envselect.go        # Env selector model and logic
    │   └── envselect_test.go   # Tests
    └── gql/                    # Phase 1-6: GraphQL explorer
        ├── app.go              # Main Bubble Tea app, view routing
        ├── model.go            # Shared model and state
        ├── keys.go             # GQL-specific keys (imports pkg/tui)
        ├── styles.go           # GQL-specific styles (imports pkg/tui)
        ├── file.go             # .gql file generation and saving
        ├── introspection/
        │   ├── query.go        # Introspection query constant
        │   ├── fetch.go        # HTTP fetch logic
        │   ├── types.go        # Introspection response structs
        │   ├── operations.go   # Operation data structures
        │   └── parser.go       # Parse schema into operations
        ├── cache/
        │   ├── cache.go        # Read/write schema cache
        │   └── hash.go         # URL hashing
        ├── views/
        │   ├── loading.go      # Loading screen with progress
        │   ├── browser.go      # Schema browser view
        │   ├── builder.go      # Query builder view
        │   ├── variables.go    # Variable input view
        │   ├── response.go     # Response display view
        │   ├── save.go         # File save confirmation
        │   ├── auth.go         # Auth retry prompt
        │   ├── error.go        # Error display
        │   └── help.go         # Help overlay
        └── components/
            ├── fieldtree.go    # Checkbox tree for fields
            ├── splitpane.go    # Dual-pane layout
            └── typeinput.go    # Type-aware inputs (text, enum)
```

---

## Shared TUI Utilities [COMPLETED]

Located in `pkg/tui/`:

**keys.go** - Key bindings and helpers:
- `IsQuitKey(msg)` - Check for ctrl+c
- `IsCancelKey(msg)` - Check for esc
- `IsConfirmKey(msg)` - Check for enter
- `HandleQuitCancelWithFilter(msg, inFilterMode)` - Smart quit/cancel handling
- Key constants: `KeyQuit`, `KeyCancel`, `KeyEnter`, etc.

**styles.go** - Shared styles:
- Colors: `ColorPrimary`, `ColorError`, `ColorMuted`, etc.
- `BadgeColors` - For endpoint tags
- `StandardHelpText()` - Consistent help footer
- `RenderHelp()`, `RenderError()`, `RenderBadge()`

> **IMPORTANT**: Never use single letter keys (like 'q') for quit/cancel as they conflict with text input and filtering. Always use `ctrl+c` for quit and `esc` for cancel.

---

## Tickets

### Phase 0: Environment Selector TUI (Current Branch Work)

> **Note**: This phase completes the existing work on branch `72-select-environment-app`. The env selector is a prerequisite for the GraphQL explorer since templates like `{{.graphqlUrl}}` need environment resolution.

#### Ticket 0.1: Dynamic Env File Discovery [COMPLETED]

**File**: `pkg/tui/envselect/envselect.go` (new file, refactor from `pkg/tui/tui.go`)
**Size**: ~40 lines

- Replace hardcoded env list with dynamic discovery using `utils.GetEnvFiles()`..
- We ask user for env selection if and only if it is required. It's required if we are in directory mode and we need to resolve the `{{.key}}`. Otherwise, say if we are in a file mode or in dir mode without any `{{.key}}`. There is no need to ask user to select the input.
- this work should fit into the overall arch design of TUI.
- I should be able to test this manually. For now just print the env selected and exit

```go
func getEnvItems() []list.Item {
    files, err := utils.GetEnvFiles()
    // Convert to list.Item slice
    // Add "Global" as default option
}
```

**Acceptance**:

- [x] Lists all `.env` files from `env/` directory
- [x] Shows "Global" as default/first option
- [x] Handles empty env directory gracefully

**Manual Test**: Run `hulak gql .` in a directory with GraphQL files that have template URLs (e.g., `{{.graphqlUrl}}`). The env selector TUI will appear.

---

#### Ticket 0.2: Env Selector Model Export

**File**: `pkg/tui/envselect/envselect.go` (continued)
**Size**: ~50 lines

Export the model and add a public function to run the selector.

```go
type EnvSelectorModel struct {
    list     list.Model
    Selected string
    Cancelled bool
}

func NewEnvSelector() EnvSelectorModel
func RunEnvSelector() (string, error) // Returns selected env name
```

**Acceptance**:

- [ ] Model is exported
- [ ] `RunEnvSelector()` returns selected env
- [ ] Returns empty string if cancelled (Ctrl+C/q)

---

#### Ticket 0.3: Env Selector Styling

**File**: `pkg/tui/envselect/styles.go`
**Size**: ~30 lines

Define consistent styles for env selector.

```go
var (
    TitleStyle = lipgloss.NewStyle()...
    ItemStyle = lipgloss.NewStyle()...
    SelectedStyle = lipgloss.NewStyle()...
)
```

**Acceptance**:

- [ ] Matches overall hulak aesthetic
- [ ] Clear visual indication of selection

---

#### Ticket 0.4: Env Selector Tests

**File**: `pkg/tui/envselect/envselect_test.go`
**Size**: ~60 lines

**Acceptance**:

- [ ] Test env file discovery
- [ ] Test model initialization
- [ ] Test selection handling

---

#### Ticket 0.5: Integrate Env Selector with Main Flow

**File**: `init.go` (modify existing)
**Size**: ~30 lines

Trigger env selector when templates need resolution but no `-env` flag provided.

```go
func shouldPromptForEnv(secretsMap map[string]any, filePath string) bool
// Returns true if file has template vars but secretsMap is missing keys
```

**Acceptance**:

- [ ] Detects when env selection needed
- [ ] Launches TUI selector
- [ ] Uses selected env for request

---

### Phase 1: Core Infrastructure

#### Ticket 1.1: Introspection Query Constant

**File**: `pkg/tui/gql/introspection/query.go`
**Size**: ~50 lines

Create the standard GraphQL introspection query as a constant string.

```go
// Query to fetch schema with:
// - Query type fields (all queries)
// - Mutation type fields (all mutations)
// - Field arguments with types
// - Return types
// - Enum values
// - Descriptions
const IntrospectionQuery = `...`
```

**Acceptance**:

- [ ] Query string is valid GraphQL
- [ ] Includes all required fields for TUI

---

#### Ticket 1.2: Introspection Response Types

**File**: `pkg/tui/gql/introspection/types.go`
**Size**: ~80 lines

Define Go structs to unmarshal the introspection JSON response.

```go
type IntrospectionResponse struct {
    Data struct {
        Schema Schema `json:"__schema"`
    } `json:"data"`
}

type Schema struct {
    QueryType    *TypeRef `json:"queryType"`
    MutationType *TypeRef `json:"mutationType"`
    Types        []Type   `json:"types"`
}

type Type struct {
    Kind        string  `json:"kind"`
    Name        string  `json:"name"`
    Description string  `json:"description"`
    Fields      []Field `json:"fields"`
    EnumValues  []EnumValue `json:"enumValues"`
    // ...
}
// etc.
```

**Acceptance**:

- [ ] All necessary types defined
- [ ] JSON tags match introspection response

---

#### Ticket 1.3: Introspection HTTP Fetcher

**File**: `pkg/tui/gql/introspection/fetch.go`
**Size**: ~60 lines

HTTP client to send introspection query to a GraphQL endpoint.

```go
func FetchSchema(url string, headers map[string]string) (*IntrospectionResponse, error)
```

**Acceptance**:

- [ ] Sends POST with introspection query
- [ ] Supports custom headers (for auth)
- [ ] Returns parsed IntrospectionResponse
- [ ] Handles HTTP errors gracefully

---

#### Ticket 1.4: Introspection Fetcher Tests

**File**: `pkg/tui/gql/introspection/fetch_test.go`
**Size**: ~80 lines

Unit tests for the fetcher using httptest.

**Acceptance**:

- [ ] Test successful fetch
- [ ] Test auth header passthrough
- [ ] Test HTTP error handling
- [ ] Test malformed JSON handling

---

#### Ticket 1.5: Operation Data Structures

**File**: `pkg/tui/gql/introspection/operations.go`
**Size**: ~50 lines

Define simplified structs for operations (queries/mutations) used by the TUI.

```go
type Operation struct {
    Name        string
    Description string
    Type        OperationType // Query or Mutation
    Arguments   []Argument
    ReturnType  *FieldType
    SourceFile  string        // Which .hk.yaml file
    EndpointURL string
}

type Argument struct {
    Name     string
    Type     *FieldType
    Required bool
}

type FieldType struct {
    Name     string
    Kind     string // SCALAR, OBJECT, ENUM, LIST, NON_NULL
    OfType   *FieldType
    Fields   []Field // For OBJECT types
    EnumValues []string // For ENUM types
}
```

**Acceptance**:

- [ ] Structs capture all info needed for TUI
- [ ] Clear distinction between Query and Mutation

---

#### Ticket 1.6: Schema Parser - Extract Operations

**File**: `pkg/tui/gql/introspection/parser.go`
**Size**: ~100 lines

Parse IntrospectionResponse into a list of Operations.

```go
func ParseOperations(resp *IntrospectionResponse, sourceFile, endpointURL string) ([]Operation, error)
```

**Acceptance**:

- [ ] Extracts all queries from queryType
- [ ] Extracts all mutations from mutationType
- [ ] Populates arguments with types
- [ ] Resolves return types
- [ ] Handles missing mutationType gracefully

---

#### Ticket 1.7: Schema Parser Tests

**File**: `pkg/tui/gql/introspection/parser_test.go`
**Size**: ~100 lines

Unit tests with sample introspection responses.

**Acceptance**:

- [ ] Test parsing queries
- [ ] Test parsing mutations
- [ ] Test argument extraction
- [ ] Test nested return types
- [ ] Test enum types

---

#### Ticket 1.8: Cache Directory Setup

**File**: `pkg/tui/gql/cache/cache.go` (part 1)
**Size**: ~40 lines

Create `.hulak/schemas/` directory structure.

```go
func EnsureCacheDir() (string, error)
func CacheFilePath(url string) string
```

**Acceptance**:

- [ ] Creates `.hulak/schemas/` if not exists
- [ ] Returns correct path
- [ ] Works cross-platform

---

#### Ticket 1.9: URL Hashing for Cache Keys

**File**: `pkg/tui/gql/cache/hash.go`
**Size**: ~30 lines

Hash URL to generate cache filename.

```go
func HashURL(url string) string // Returns sha256 hex string
```

**Acceptance**:

- [ ] Deterministic output
- [ ] Safe filename characters

---

#### Ticket 1.10: Cache Read/Write

**File**: `pkg/tui/gql/cache/cache.go` (part 2)
**Size**: ~60 lines

Read and write cached schema with TTL.

```go
type CachedSchema struct {
    FetchedAt  time.Time
    Operations []Operation
}

func WriteCache(url string, ops []Operation) error
func ReadCache(url string, maxAge time.Duration) ([]Operation, bool)
```

**Acceptance**:

- [ ] Writes JSON to cache file
- [ ] Reads and validates TTL
- [ ] Returns false if expired or missing

---

#### Ticket 1.11: Cache Tests

**File**: `pkg/tui/gql/cache/cache_test.go`
**Size**: ~80 lines

**Acceptance**:

- [ ] Test write and read roundtrip
- [ ] Test TTL expiration
- [ ] Test missing cache file

---

#### Ticket 1.12: TUI App State and Model

**File**: `pkg/tui/gql/model.go`
**Size**: ~60 lines

Define the main model struct and view states.

```go
type ViewState int

const (
    ViewLoading ViewState = iota
    ViewBrowser
    ViewBuilder
    ViewVariables
    ViewResponse
)

type Model struct {
    state       ViewState
    operations  []Operation
    endpoints   []Endpoint
    selected    *Operation
    err         error
    // Sub-models for each view
    loading     LoadingModel
    browser     BrowserModel
    builder     BuilderModel
    variables   VariablesModel
    response    ResponseModel
}
```

**Acceptance**:

- [ ] All view states defined
- [ ] Model has necessary fields

---

#### Ticket 1.13: GQL-Specific Key Bindings [PARTIALLY DONE]

**File**: `pkg/tui/gql/keys.go`
**Size**: ~30 lines

Extend shared key bindings with GQL-specific keys. Import from `pkg/tui`.

```go
import "github.com/xaaha/hulak/pkg/tui"

// Use tui.IsQuitKey, tui.HandleQuitCancelWithFilter, etc.

// GQL-specific keys
type GQLKeyMap struct {
    tui.CommonKeyMap
    Refresh key.Binding // r - refresh schemas
    Filter  key.Binding // f - filter by endpoint
    Save    key.Binding // s - save query
}
```

**Acceptance**:

- [x] Shared keys imported from `pkg/tui/keys.go`
- [ ] GQL-specific keys added (refresh, filter, save)

---

#### Ticket 1.14: GQL-Specific Styles [PARTIALLY DONE]

**File**: `pkg/tui/gql/styles.go`
**Size**: ~30 lines

Extend shared styles with GQL-specific styles. Import from `pkg/tui`.

```go
import "github.com/xaaha/hulak/pkg/tui"

// Use tui.ColorPrimary, tui.RenderBadge, tui.StandardHelpText, etc.

// GQL-specific styles
var (
    QueryStyle    = lipgloss.NewStyle().Foreground(tui.ColorPrimary)
    MutationStyle = lipgloss.NewStyle().Foreground(tui.ColorWarning)
)
```

**Acceptance**:

- [x] Shared styles imported from `pkg/tui/styles.go`
- [ ] GQL-specific styles added (query, mutation highlighting)

---

#### Ticket 1.15: TUI App Skeleton - Init and Update

**File**: `pkg/tui/gql/app.go`
**Size**: ~80 lines

Main Bubble Tea app with Init() and Update().

```go
func NewModel(paths []string, env string) Model
func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
```

**Acceptance**:

- [ ] Init triggers schema loading
- [ ] Update routes to current view's update
- [ ] Global keys (quit, back) handled

---

#### Ticket 1.16: TUI App Skeleton - View Routing

**File**: `pkg/tui/gql/app.go` (continued)
**Size**: ~40 lines

View() method that routes to correct view.

```go
func (m Model) View() string {
    switch m.state {
    case ViewLoading: return m.loading.View()
    // ...
    }
}
```

**Acceptance**:

- [ ] Routes to correct view based on state
- [ ] Error state handled

---

#### Ticket 1.17: CLI Entry Point Integration

**File**: `pkg/features/graphql/graphql.go` (modify existing)
**Size**: ~30 lines

Update `Introspect` function to launch TUI.

```go
func Introspect(args []string) {
    // If interactive terminal, launch TUI
    // Otherwise, keep current behavior
}
```

**Acceptance**:

- [ ] TUI launches with `hulak gql .`
- [ ] Passes paths and env to TUI
- [ ] Graceful fallback for non-TTY

---

### Phase 2: Schema Browser View

#### Ticket 2.1: Loading View

**File**: `pkg/tui/gql/views/loading.go`
**Size**: ~60 lines

Loading screen with progress for schema fetching.

```go
type LoadingModel struct {
    spinner  spinner.Model
    progress []EndpointProgress
}

type EndpointProgress struct {
    Name   string
    Status LoadStatus // Pending, Loading, Done, Failed
    OpCount int
}
```

**Acceptance**:

- [ ] Shows spinner
- [ ] Lists endpoints with status
- [ ] Shows operation count when done

---

#### Ticket 2.2: Loading View - Parallel Fetch Command

**File**: `pkg/tui/gql/views/loading.go` (continued)
**Size**: ~60 lines

Tea commands for parallel schema fetching.

```go
func fetchSchemasCmd(endpoints []Endpoint) tea.Cmd
type schemaFetchedMsg struct { ... }
type schemaFailedMsg struct { ... }
```

**Acceptance**:

- [ ] Fetches schemas in parallel
- [ ] Sends progress messages
- [ ] Handles auth failures

---

#### Ticket 2.3: Browser Model Structure

**File**: `pkg/tui/gql/views/browser.go`
**Size**: ~50 lines

Browser view model with list and preview.

```go
type BrowserModel struct {
    list        list.Model
    operations  []Operation
    selected    *Operation
    filterText  string
    filterEndpoint string // Empty = all
}
```

**Acceptance**:

- [ ] Holds operation list
- [ ] Tracks selection and filters

---

#### Ticket 2.4: Browser Operation List Item

**File**: `pkg/tui/gql/views/browser.go` (continued)
**Size**: ~40 lines

Implement list.Item interface for operations.

```go
type operationItem struct {
    op Operation
}

func (i operationItem) Title() string       // Operation name
func (i operationItem) Description() string // Args summary
func (i operationItem) FilterValue() string // For fuzzy search
```

**Acceptance**:

- [ ] Shows operation name
- [ ] Shows argument hints
- [ ] Searchable by name and description

---

#### Ticket 2.5: Browser Endpoint Badge Rendering

**File**: `pkg/tui/gql/views/browser.go` (continued)
**Size**: ~40 lines

Render colored endpoint badges.

```go
func renderBadge(endpointName string, colorIndex int) string
func assignBadgeColors(endpoints []string) map[string]int
```

**Acceptance**:

- [ ] Distinct colors per endpoint
- [ ] Badge shows source file name

---

#### Ticket 2.6: Browser List Initialization

**File**: `pkg/tui/gql/views/browser.go` (continued)
**Size**: ~50 lines

Initialize list.Model with operations.

```go
func NewBrowserModel(operations []Operation) BrowserModel
```

**Acceptance**:

- [ ] Groups by Query/Mutation
- [ ] Fuzzy filtering enabled
- [ ] No pagination limit

---

#### Ticket 2.7: Browser Preview Pane

**File**: `pkg/tui/gql/views/browser.go` (continued)
**Size**: ~60 lines

Render preview for selected operation.

```go
func (m BrowserModel) renderPreview() string
```

Shows:

- Operation name and description
- Endpoint URL and source file
- Arguments with types
- Return type fields

**Acceptance**:

- [ ] Updates on selection change
- [ ] Shows all operation details
- [ ] Handles long descriptions

---

#### Ticket 2.8: Browser Dual-Pane Layout

**File**: `pkg/tui/gql/components/splitpane.go`
**Size**: ~50 lines

Dual-pane layout component.

```go
func RenderSplitPane(left, right string, leftWidth int, height int) string
```

**Acceptance**:

- [ ] Left pane for list
- [ ] Right pane for preview
- [ ] Respects terminal width

---

#### Ticket 2.9: Browser Update and View

**File**: `pkg/tui/gql/views/browser.go` (continued)
**Size**: ~80 lines

Update() and View() for browser.

```go
func (m BrowserModel) Update(msg tea.Msg) (BrowserModel, tea.Cmd)
func (m BrowserModel) View() string
```

**Acceptance**:

- [ ] Navigation updates selection
- [ ] Enter transitions to builder
- [ ] Filter input works
- [ ] Renders list + preview

---

#### Ticket 2.10: Browser Tests

**File**: `pkg/tui/gql/views/browser_test.go`
**Size**: ~80 lines

**Acceptance**:

- [ ] Test list initialization
- [ ] Test filtering
- [ ] Test selection change

---

### Phase 3: Query Builder View

#### Ticket 3.1: Field Tree Data Structure

**File**: `pkg/tui/gql/components/fieldtree.go`
**Size**: ~60 lines

Tree structure for selectable fields.

```go
type FieldNode struct {
    Name     string
    Type     string
    Selected bool
    Expanded bool
    Children []*FieldNode
    Parent   *FieldNode
}

func BuildFieldTree(returnType *FieldType) *FieldNode
```

**Acceptance**:

- [ ] Supports nested objects
- [ ] Tracks selection state
- [ ] Tracks expansion state

---

#### Ticket 3.2: Field Tree Rendering

**File**: `pkg/tui/gql/components/fieldtree.go` (continued)
**Size**: ~60 lines

Render field tree with checkboxes.

```go
func (n *FieldNode) Render(cursor int, depth int) string
```

```
[x] id
[x] name
[ ] email
[+] profile (Object)
    [ ] bio
    [ ] avatar
```

**Acceptance**:

- [ ] Shows checkboxes
- [ ] Shows expand/collapse indicator
- [ ] Indents nested fields

---

#### Ticket 3.3: Field Tree Navigation

**File**: `pkg/tui/gql/components/fieldtree.go` (continued)
**Size**: ~50 lines

Navigation and toggle logic.

```go
func (n *FieldNode) Toggle()
func (n *FieldNode) ToggleExpand()
func (n *FieldNode) SelectAll()
func (n *FieldNode) SelectNone()
func (n *FieldNode) FlattenVisible() []*FieldNode // For cursor navigation
```

**Acceptance**:

- [ ] Toggle marks/unmarks field
- [ ] Expand shows children
- [ ] SelectAll/None recursive

---

#### Ticket 3.4: Field Tree Tests

**File**: `pkg/tui/gql/components/fieldtree_test.go`
**Size**: ~80 lines

**Acceptance**:

- [ ] Test tree building
- [ ] Test toggle
- [ ] Test navigation

---

#### Ticket 3.5: Query String Generator

**File**: `pkg/tui/gql/views/builder.go`
**Size**: ~80 lines

Generate GraphQL query string from selections.

```go
func GenerateQuery(op *Operation, root *FieldNode) string
```

Output:

```graphql
query getUserById($id: ID!) {
  user(id: $id) {
    id
    name
    profile {
      bio
    }
  }
}
```

**Acceptance**:

- [ ] Correct query/mutation keyword
- [ ] Includes selected fields only
- [ ] Handles nested selections
- [ ] Includes argument variables

---

#### Ticket 3.6: Query String Generator Tests

**File**: `pkg/tui/gql/views/builder_test.go`
**Size**: ~80 lines

**Acceptance**:

- [ ] Test simple query
- [ ] Test with arguments
- [ ] Test nested fields

---

#### Ticket 3.7: Builder Model and View

**File**: `pkg/tui/gql/views/builder.go` (continued)
**Size**: ~80 lines

Builder view model.

```go
type BuilderModel struct {
    operation *Operation
    fieldTree *FieldNode
    cursor    int
    query     string // Live preview
}

func NewBuilderModel(op *Operation) BuilderModel
func (m BuilderModel) Update(msg tea.Msg) (BuilderModel, tea.Cmd)
func (m BuilderModel) View() string
```

**Acceptance**:

- [ ] Shows field tree on left
- [ ] Shows query preview on right
- [ ] Updates query on toggle

---

### Phase 4: Variable Input & Execution

#### Ticket 4.1: Variable Input Model

**File**: `pkg/tui/gql/views/variables.go`
**Size**: ~60 lines

Model for variable input form.

```go
type VariablesModel struct {
    operation *Operation
    query     string
    inputs    []VariableInput
    focused   int
}

type VariableInput struct {
    Name     string
    Type     string
    Required bool
    Value    string
    Input    textinput.Model // For text/number
    // or List for enum
}
```

**Acceptance**:

- [ ] One input per argument
- [ ] Tracks required fields
- [ ] Focus management

---

#### Ticket 4.2: Text Input Component

**File**: `pkg/tui/gql/components/typeinput.go`
**Size**: ~50 lines

Type-aware text input wrapper.

```go
func NewTypeInput(arg Argument) VariableInput
```

**Acceptance**:

- [ ] Text input for String, ID
- [ ] Numeric validation for Int, Float
- [ ] Placeholder shows type

---

#### Ticket 4.3: Enum Dropdown Component

**File**: `pkg/tui/gql/components/typeinput.go` (continued)
**Size**: ~60 lines

Dropdown for enum types.

```go
type EnumInput struct {
    Options  []string
    Selected int
}
```

**Acceptance**:

- [ ] Shows enum values
- [ ] Arrow keys to select
- [ ] Returns selected value

---

#### Ticket 4.4: Variables View Rendering

**File**: `pkg/tui/gql/views/variables.go` (continued)
**Size**: ~60 lines

Render variable input form.

```go
func (m VariablesModel) View() string
```

**Acceptance**:

- [ ] Shows all inputs with labels
- [ ] Highlights focused input
- [ ] Shows required indicator

---

#### Ticket 4.5: Variables View Update

**File**: `pkg/tui/gql/views/variables.go` (continued)
**Size**: ~60 lines

Handle input updates and validation.

```go
func (m VariablesModel) Update(msg tea.Msg) (VariablesModel, tea.Cmd)
func (m VariablesModel) Validate() []string // Returns errors
func (m VariablesModel) GetVariables() map[string]any
```

**Acceptance**:

- [ ] Tab cycles inputs
- [ ] Enter submits if valid
- [ ] Shows validation errors

---

#### Ticket 4.6: GraphQL Request Executor

**File**: `pkg/tui/gql/views/response.go`
**Size**: ~60 lines

Execute the built query.

```go
func executeQueryCmd(endpoint string, headers map[string]string, query string, variables map[string]any) tea.Cmd

type queryResultMsg struct {
    Body     []byte
    Status   int
    Duration time.Duration
    Err      error
}
```

**Acceptance**:

- [ ] Sends GraphQL POST request
- [ ] Returns response body
- [ ] Captures timing

---

#### Ticket 4.7: Response Model and View

**File**: `pkg/tui/gql/views/response.go` (continued)
**Size**: ~80 lines

Display response.

```go
type ResponseModel struct {
    status   int
    duration time.Duration
    body     string
    viewport viewport.Model
    err      error
}

func (m ResponseModel) View() string
```

**Acceptance**:

- [ ] Shows status and timing
- [ ] Scrollable JSON body
- [ ] Shows errors clearly

---

#### Ticket 4.8: Response View Update and Actions

**File**: `pkg/tui/gql/views/response.go` (continued)
**Size**: ~50 lines

Handle re-run, edit, save actions.

```go
func (m ResponseModel) Update(msg tea.Msg) (ResponseModel, tea.Cmd)
```

**Acceptance**:

- [ ] R re-runs query
- [ ] E goes back to variables
- [ ] S triggers save
- [ ] Y copies to clipboard

---

### Phase 5: File Integration

#### Ticket 5.1: Generate .gql File Content

**File**: `pkg/tui/gql/file.go`
**Size**: ~50 lines

Generate GraphQL query file content (pure GraphQL syntax).

```go
func GenerateGQLContent(op *Operation, query string, variables map[string]any) string
```

Output format:

```graphql
# Operation: getUserById
# Endpoint: https://api.example.com/graphql
# Source: users.hk.yaml

query getUserById($id: ID!) {
  user(id: $id) {
    id
    name
  }
}

# Variables (for reference):
# {
#   "id": "user-123"
# }
```

**Acceptance**:

- [ ] Valid GraphQL syntax
- [ ] Includes metadata as comments
- [ ] Variables documented in comments

---

#### Ticket 5.2: Save .gql File Logic

**File**: `pkg/tui/gql/file.go` (continued)
**Size**: ~50 lines

Save query to `.gql` file.

```go
func SaveQueryFile(op *Operation, content string) (string, error)
```

**Acceptance**:

- [ ] Filename = `{operationName}.gql`
- [ ] Same directory as source `.hk.yaml` file
- [ ] Returns saved path

---

#### Ticket 5.3: File Overwrite Prompt

**File**: `pkg/tui/gql/views/save.go`
**Size**: ~60 lines

Prompt when file exists.

```go
type SaveModel struct {
    path    string
    exists  bool
    confirm bool
}
```

**Acceptance**:

- [ ] Shows if file exists
- [ ] Confirm to overwrite
- [ ] Option to save with suffix (e.g., `getUserById_2.gql`)

---

#### Ticket 5.4: Environment Selection Integration

**File**: `pkg/tui/gql/views/loading.go` (modify)
**Size**: ~50 lines

Trigger env selection when templates need resolution.

```go
func needsEnvSelection(endpoints []Endpoint) bool
func showEnvSelector() tea.Cmd
```

**Acceptance**:

- [ ] Detects template variables in endpoint URLs
- [ ] Launches env selector TUI (from Phase 0)
- [ ] Returns selected env and reloads endpoints

---

### Phase 6: Polish

#### Ticket 6.1: Loading Spinner

**File**: `pkg/tui/gql/views/loading.go` (enhance)
**Size**: ~30 lines

Add animated spinner during fetch.

**Acceptance**:

- [ ] Spinner animates
- [ ] Stops when done

---

#### Ticket 6.2: Error States UI

**File**: `pkg/tui/gql/views/error.go`
**Size**: ~50 lines

Consistent error display.

```go
func RenderError(err error, canRetry bool) string
```

**Acceptance**:

- [ ] Clear error message
- [ ] Retry option if applicable
- [ ] Back to exit

---

#### Ticket 6.3: Auth Retry Flow

**File**: `pkg/tui/gql/views/auth.go`
**Size**: ~80 lines

Prompt for auth when introspection fails.

```go
type AuthModel struct {
    endpoint string
    input    textinput.Model
}
```

**Acceptance**:

- [ ] Shows endpoint URL
- [ ] Text input for token
- [ ] Retry with new auth

---

#### Ticket 6.4: Help Overlay

**File**: `pkg/tui/gql/views/help.go`
**Size**: ~60 lines

Show key bindings on `?`.

```go
func RenderHelp(keys KeyMap, currentView ViewState) string
```

**Acceptance**:

- [ ] Context-aware (shows keys for current view)
- [ ] Dismissible with any key

---

#### Ticket 6.5: JSON Syntax Highlighting

**File**: `pkg/tui/gql/views/response.go` (enhance)
**Size**: ~50 lines

Color JSON in response.

```go
func highlightJSON(json string) string
```

**Acceptance**:

- [ ] Keys in one color
- [ ] Strings in another
- [ ] Numbers/bools distinct

---

#### Ticket 6.6: Query Syntax Highlighting

**File**: `pkg/tui/gql/views/builder.go` (enhance)
**Size**: ~50 lines

Color GraphQL query preview.

```go
func highlightGraphQL(query string) string
```

**Acceptance**:

- [ ] Keywords colored
- [ ] Field names colored
- [ ] Variables colored

---

## Ticket Summary

| Phase                        | Tickets        | Total Lines (est.) |
| ---------------------------- | -------------- | ------------------ |
| Phase 0: Env Selector        | 5 tickets      | ~210 lines         |
| Phase 1: Core Infrastructure | 17 tickets     | ~900 lines         |
| Phase 2: Schema Browser      | 10 tickets     | ~550 lines         |
| Phase 3: Query Builder       | 7 tickets      | ~490 lines         |
| Phase 4: Variable Input      | 8 tickets      | ~480 lines         |
| Phase 5: File Integration    | 4 tickets      | ~210 lines         |
| Phase 6: Polish              | 6 tickets      | ~320 lines         |
| **Total**                    | **57 tickets** | **~3160 lines**    |

---

## Dependencies

External packages (already in project):

- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/bubbles`
- `github.com/charmbracelet/lipgloss`

No new dependencies required.

---

## Testing Strategy

- Unit tests for each package
- Integration test for full flow (mock HTTP)
- Manual testing with real GraphQL endpoints

---

## Notes

- Each ticket is ~50-100 lines including tests (max 150)
- Tickets can be done in order within each phase
- Phase 2+ depends on Phase 1 completion
- Some tickets in same file - mark as "part 2" etc.
