# TUI maintainer map

This directory contains Hulak's Bubble Tea user interfaces and shared TUI widgets.

## Packages

- `tui` — shared widgets and interaction helpers used by the selector and GraphQL explorer.
- `tui/envselect` — environment picker.
- `tui/gqlexplorer` — full-screen GraphQL explorer.

Keep a component in the root `tui` package when it is reusable by more than one screen. Screen-specific state and behavior should remain in that screen's package.

## Bubble Tea flow

Each full-screen model follows the same path:

1. `Init` starts initial commands.
2. `Update` receives Bubble Tea messages and delegates keyboard, mouse, async, and resize handling.
3. Handlers mutate model state and may return a `tea.Cmd`.
4. View helpers render the updated state.

Commands must return messages to `Update`; they should not mutate model state from a goroutine.

## GraphQL explorer

`gqlexplorer.Model` remains the owner of explorer state. Its implementation is split by responsibility so a change can be located without tracing one large file.

### Model orchestration

| Change                                                                       | File                |
| ---------------------------------------------------------------------------- | ------------------- |
| Model state, messages, construction, `Init`, or `Update` dispatch            | `model.go`          |
| Terminal sizing, panel dimensions, focus synchronization, or viewport sizing | `model_layout.go`   |
| Keyboard shortcuts or panel navigation                                       | `model_keys.go`     |
| Mouse handling or mouse-zone IDs                                             | `model_mouse.go`    |
| Schema refresh                                                               | `model_refresh.go`  |
| Query execution or spinner commands                                          | `model_execute.go`  |
| Response state, caching, search, headers, or response saving                 | `model_response.go` |
| Bottom actions, notification enqueueing, or notification action-row state    | `model_actions.go`  |
| Notification message handling and state                                      | `model.go`          |
| Notification keyboard interaction and copying                               | `model_keys.go`     |
| Notification modal rendering                                                 | `model_view.go`     |
| Top-level screen composition and panel synchronization                       | `model_view.go`     |
| Program startup and exported explorer runners                                | `run.go`            |

### Detail form

| Change                                                     | File                 |
| ---------------------------------------------------------- | -------------------- |
| A single form item's widget behavior or appearance         | `form_item.go`       |
| Building a `DetailForm` from an operation                  | `form_build.go`      |
| Recursive input-object expansion and path construction     | `form_expand.go`     |
| Dynamic top-level or nested list rows                      | `form_lists.go`      |
| Form cursor, keyboard, mouse, dropdown, or search behavior | `form_navigation.go` |
| Detail-form rendering                                      | `form_view.go`       |

### Other GraphQL explorer responsibilities

| Change                                                   | File              |
| -------------------------------------------------------- | ----------------- |
| Operation and endpoint filtering                         | `filter.go`       |
| Operation list, endpoint badges, schema detail rendering | `render.go`       |
| GraphQL query construction                               | `querybuilder.go` |
| Variables construction and typed value conversion        | `variables.go`    |
| Syntax highlighting                                      | `highlight.go`    |
| Saving query/request files                               | `savefile.go`     |
| Shared operation and GraphQL type helpers                | `types.go`        |

Tests use matching responsibility names where a source area is large (for example, `model_keys.go` is covered by navigation and command tests, while form list behavior is in `form_lists_test.go` and `nested_input_test.go`). Golden tests cover complete rendered screens.
