# Contributing to Hulak

Thanks for your interest in contributing. This guide covers everything you need to get productive.

## Prerequisites

- [mise](https://mise.jdx.dev/getting-started.html) (manages Go version + dev tools)

## Setup

```bash
git clone https://github.com/xaaha/hulak.git
cd hulak
mise install
```

`mise install` reads `mise.toml`, installs tools, and automatically sets up git hooks:

| Tool            | Purpose                                                     |
| --------------- | ----------------------------------------------------------- |
| `go`            | Pinned Go version                                           |
| `watchexec`     | File watcher for hot reload during development              |
| `golangci-lint` | Linter aggregator (see `.golangci.yml` for enabled linters) |
| `vhs`           | Terminal GIF recorder for demos                             |

## Development Workflow

### Hot Reload

Use mise tasks to auto-rebuild on file changes:

```bash
mise run watch:gql      # hot reload: runs GraphQL explorer
mise run watch:unit     # auto-runs pkg/ tests on save
mise run watch:tui      # auto-runs TUI golden tests on save
```

These use `watchexec` under the hood. Every `.go` file change triggers a rebuild.

### Tasks

All project commands are mise tasks. Run `mise tasks` to see the full list.

| Command                    | Description                                           |
| -------------------------- | ----------------------------------------------------- |
| `mise run build`           | Build the binary                                      |
| `mise run test:unit`       | Run unit tests with 30s timeout                       |
| `mise run test:tui`        | Run TUI golden file tests                             |
| `mise run test:tui:update` | Regenerate TUI golden files after intentional changes |
| `mise run test:api`        | Run E2E API calls                                     |
| `mise run test:auth2`      | Test OAuth2 flow                                      |
| `mise run lint`            | Format + lint (`golangci-lint`)                       |
| `mise run check`           | Lint + unit tests (pre-push sanity check)             |
| `mise run bench`           | Run benchmarks                                        |
| `mise run coverage:gen`    | Generate coverage report                              |
| `mise run coverage:view`   | Open coverage in browser                              |
| `mise run hooks`           | Re-install git pre-commit hooks                       |
| `mise run record`          | Record terminal demo GIF with VHS                     |
| `mise run watch:gql`       | Hot reload: GraphQL explorer                          |
| `mise run watch:unit`      | Hot reload: unit tests                                |
| `mise run watch:tui`       | Hot reload: TUI golden tests                          |

### Pre-commit Hooks

Git hooks are installed automatically during `mise install`. To re-install manually, run `mise run hooks`. The pre-commit hook runs before every commit:

1. `go fmt ./...`
2. `go vet ./...`
3. `go test ./pkg/... -short -timeout 30s`

This prevents committing code that doesn't compile or breaks tests.

## Testing

### Unit Tests

```bash
mise test:unit # run all unit tests OR
go test ./pkg/utils/                # specific package
```

Tests follow table-driven patterns with `t.Run()` subtests. See any `*_test.go` file for examples.

### TUI Golden File Tests (teatest)

TUI components have snapshot tests that capture visual output and compare against golden files.

```bash
mise run test:tui            # runs golden file comparisons
mise run test:tui:update     # regenerate golden files after intentional TUI changes
mise run watch:tui           # auto-run TUI tests on file changes
```

Golden files live in `pkg/tui/testdata/*.golden`. When you change TUI rendering:

1. Run tests, they fail showing the diff between old and new output
2. Review the diff to confirm the change is intentional
3. Run with `-update` to accept the new output
4. Commit the updated `.golden` files

To add a new golden file test, see `pkg/tui/selector_golden_test.go` for the pattern.

## Linting

```bash
mise run lint           # format + golangci-lint
golangci-lint run ./... # lint only
```

See `.golangci.yml` for the full list of enabled linters and their configuration.

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

#### Homebrew Tap Token Setup

- The release workflow uses a [GitHub App](https://docs.github.com/en/apps) to push the Homebrew formula to [xaaha/homebrew-tap](https://github.com/xaaha/homebrew-tap). The App generates a short-lived token on every release — no manual rotation needed.
- The version is injected at build time via ldflags, so no need to edit `version.go`.

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

# Best Practices

- Review existing issues, pull requests, and documentation before starting work.
- If an issue description is unclear or incomplete, please ask questions before beginning implementation.
- Each pull request should address **one issue only**.
- Keep pull requests focused and reasonably sized to make reviews easier.
- Reference the related issue in the pull request description when applicable.
- Follow the existing coding style and conventions used in the project.
- Write clear, readable, and maintainable code.
- Add or update tests for new features and bug fixes whenever possible.
- The use of AI tools is acceptable, but contributors must fully understand and review any AI-assisted code.
- Pull requests that are entirely auto-generated with AI will not be accepted.
- Be respectful and constructive in discussions, reviews, and feedback.
- All contributors are expected to follow the project’s Code of Conduct.
