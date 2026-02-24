# Contributing to Hulak

Thanks for your interest in contributing. This guide covers everything you need to get productive.

## Prerequisites

- [Go](https://go.dev/dl/) (version pinned in `go.mod`)
- [mise](https://mise.jdx.dev/getting-started.html) (manages Go version + dev tools)

## Setup

```bash
git clone https://github.com/xaaha/hulak.git
cd hulak
mise install    # installs Go, watchexec, golangci-lint, vhs
go mod tidy
make install-hooks
```

`mise install` reads `mise.toml` and installs:

| Tool | Purpose |
|------|---------|
| `go` | Pinned Go version |
| `watchexec` | File watcher for hot reload during development |
| `golangci-lint` | Linter (wraps revive, gosec, staticcheck, errcheck, bodyclose, gocritic) |
| `vhs` | Terminal GIF recorder for demos |

## Development Workflow

### Hot Reload

Use mise tasks to auto-rebuild on file changes:

```bash
mise run dev          # hot reload: runs TUI with form_data example
mise run dev:gql      # hot reload: runs GraphQL explorer
mise run dev:test     # auto-runs pkg/ tests on save
mise run dev:tui      # auto-runs TUI tests on save
```

These use `watchexec` under the hood. Every `.go` file change triggers a rebuild.

### Make Targets

For one-off commands without file watching:

| Command | Description |
|---------|-------------|
| `make build` | Build the binary |
| `make unit` | Run all tests |
| `make test-unit` | Run pkg/ tests with 30s timeout |
| `make lint` | Format + lint (`golangci-lint`) |
| `make check` | Lint + unit tests (pre-push sanity check) |
| `make check-e2e` | Full tests including real API calls |
| `make bench` | Run benchmarks |
| `make gql` | Run GraphQL explorer |
| `make gen-coverage` | Generate coverage report |
| `make view-coverage` | Open coverage in browser |
| `make install-hooks` | Install git pre-commit hooks |

### Pre-commit Hooks

`make install-hooks` sets up a git hook that runs before every commit:

1. `go fmt ./...`
2. `go vet ./...`
3. `go test ./pkg/... -short -timeout 30s`

This prevents committing code that doesn't compile or breaks tests.

## Testing

### Unit Tests

```bash
go test ./...                       # all tests
go test ./pkg/utils/                # specific package
go test -run=TestCopyEnvMap ./pkg/utils/ -v  # single test
```

Tests follow table-driven patterns with `t.Run()` subtests. See any `*_test.go` file for examples.

### TUI Golden File Tests (teatest)

TUI components have snapshot tests that capture visual output and compare against golden files.

```bash
go test ./pkg/tui/...               # runs golden file comparisons
go test ./pkg/tui/... -update       # regenerate golden files after intentional TUI changes
```

Golden files live in `pkg/tui/testdata/*.golden`. When you change TUI rendering:

1. Run tests — they fail showing the diff between old and new output
2. Review the diff to confirm the change is intentional
3. Run with `-update` to accept the new output
4. Commit the updated `.golden` files

To add a new golden file test, see `pkg/tui/selector_golden_test.go` for the pattern.

## Linting

```bash
make lint               # format + golangci-lint
golangci-lint run ./... # lint only
```

The linter config (`.golangci.yml`) runs these linters:

| Linter | What it catches |
|--------|----------------|
| revive | Go style rules (naming, error handling, imports) |
| gosec | Security issues (hardcoded creds, weak crypto) |
| staticcheck | Bugs, deprecated APIs, dead code |
| errcheck | Unchecked errors |
| bodyclose | Unclosed HTTP response bodies |
| gocritic | Performance and correctness patterns |
| govet | Suspicious constructs |

## Code Style

### Imports

Group in this order, separated by blank lines:

1. Standard library
2. External dependencies
3. Internal packages (`github.com/xaaha/hulak/...`)

### Naming

- Exported: `PascalCase` (functions, types, constants)
- Unexported: `camelCase`
- Packages: lowercase single word

### Error Handling

- Always check and handle errors
- Wrap with context: `fmt.Errorf("parsing config: %w", err)`
- User-facing: `utils.ColorError()`
- Fatal: `utils.PanicRedAndExit()`

### Testing

- Table-driven with `t.Run()`
- Use `t.TempDir()` for temp directories
- Test both success and error paths

## Recording Terminal Demos

```bash
mise run record    # generates demo.gif from demo.tape
```

Edit `demo.tape` to change the recording script. See [VHS docs](https://github.com/charmbracelet/vhs) for the tape format.

## CI/CD

### Pull Requests

Every push and PR to `main` triggers `.github/workflows/ci.yml`:

- Runs `go test ./...`
- Runs `golangci-lint`
- Checks `go fmt` produced no diff

### Releases

Releases are automated with [GoReleaser](https://goreleaser.com/). When a version tag is pushed:

```bash
git tag v0.2.0
git push origin v0.2.0
```

`.github/workflows/release.yml` triggers and:

1. Cross-compiles for Linux, macOS, Windows (amd64 + arm64)
2. Creates a GitHub Release with changelog
3. Uploads binaries and checksums
4. Pushes the updated Homebrew formula to [xaaha/homebrew-tap](https://github.com/xaaha/homebrew-tap)

The version is injected at build time via ldflags — no need to edit `version.go`.

For local builds with a specific version:

```bash
go build -ldflags "-X github.com/xaaha/hulak/pkg/userFlags.version=v0.2.0" -o hulak
```

## Project Structure

```
pkg/
  actions/       # Template actions (getValueOf, getFile) with caching
  apiCalls/      # HTTP request execution and response handling
  envparser/     # .env file parsing, type inference, variable substitution
  features/      # OAuth 2.0 flow, browser launching
  features/graphql/  # GraphQL-specific handling
  migration/     # Postman collection/environment migration
  tui/           # Bubble Tea TUI components
  tui/apicaller/ # Interactive file+env picker
  tui/envselect/ # Environment selection
  tui/gqlexplorer/ # GraphQL schema explorer
  userFlags/     # CLI flag parsing and subcommands
  utils/         # Shared utilities (file ops, printing, paths)
  yamlparser/    # YAML parsing, validation, template resolution
```