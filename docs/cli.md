# Hulak CLI Reference

Hulak supports two ways of running requests:

- **Recommended:** command-first usage such as `hulak run path/to/file.yaml`
- **Supported shorthand:** root flags such as `hulak -fp path/to/file.yaml` or `hulak -dir path/to/dir/`

If you are documenting or teaching Hulak, prefer the command-first form.

## Quick Start

```bash
# run one request file
hulak run path/to/file.yaml

# run one request file with a specific environment
hulak run path/to/file.yaml --env staging

# run a directory concurrently
hulak run path/to/dir/

# run a directory sequentially
hulak run path/to/dir/ --sequential

# open the interactive picker
hulak
```

## Discovering Commands

Use these help entry points when you want the current CLI surface from the binary itself:

```bash
hulak help
hulak run --help
hulak gql --help
hulak env --help
```

For command-specific help, prefer `hulak <command> --help`.

## Command Index

| Command | Purpose | Example |
| --- | --- | --- |
| `run` | Run one request file or a directory | `hulak run requests/get-user.hk.yaml` |
| `version` | Print the current Hulak version | `hulak version` |
| `init` | Create project setup and env files | `hulak init` |
| `migrate` | Convert Postman exports to Hulak files | `hulak migrate collection.json` |
| `doctor` | Check project health | `hulak doctor` |
| `gql` | Open the GraphQL explorer | `hulak gql .` |
| `env` | Inspect the environment-secrets command tree | `hulak env --help` |
| `help` | Show top-level help | `hulak help` |

## Core Behaviors

### Interactive mode

Running `hulak` with no file or directory target opens the interactive picker.

- Hulak asks you to choose a request file first.
- It only asks for an environment if the selected request uses template values like `{{.key}}`.
- In non-interactive shells, you should pass a file or directory target instead.

### Environment selection

- The default environment is `global`.
- Environment selection only matters when the request needs template resolution.
- `run` and `gql` support `--env` / `--environment` to skip the environment picker.

## Commands

### `run`

Run a single request file or every request file in a directory.

```bash
hulak run path/to/file.yaml
hulak run path/to/file.yaml --env staging
hulak run path/to/file.yaml --debug
hulak run path/to/dir/
hulak run path/to/dir/ --sequential
```

Supported flags:

| Flag | Meaning |
| --- | --- |
| `--env`, `--environment` | Use a specific environment |
| `--debug` | Print request and response debug details |
| `--sequential`, `--seq` | Run directory files one at a time |

Notes:

- `hulak run` accepts either a file path or a directory path.
- Directories run concurrently by default.
- `hulak run path/to/file.yaml --debug --env staging` is supported; trailing flags after the path are parsed correctly.

### `version`

Print the installed Hulak version.

```bash
hulak version
```

### `init`

Create the default Hulak project layout in the current directory.

```bash
hulak init
hulak init -env staging prod
```

Notes:

- `hulak init` creates the default setup, including `env/global.env` and the example API options file.
- On `init`, `-env` is a **boolean setup flag**, not an environment selector. It tells Hulak to create the named env files you pass after it.

### `migrate`

Convert Postman v2.1 environment and collection exports into Hulak files.

```bash
hulak migrate collection.json
hulak migrate env.json collection.json
```

### `doctor`

Run project health checks.

```bash
hulak doctor
```

This checks for common issues such as missing `.gitignore` entries, weak env file permissions, and secrets in git history.

### `gql`

Open the GraphQL explorer for one file or a directory.

```bash
hulak gql .
hulak gql path/to/schema.yml
hulak gql -env staging ./collections/graphql
```

Aliases:

- `graphql`
- `GraphQL`

Supported flags:

| Flag | Meaning |
| --- | --- |
| `--env`, `--environment` | Use a specific environment and skip the selector |

Read the full guide in [graphql-explorer.md](./graphql-explorer.md).

### `env`

The `env` command tree is exposed in the CLI for encrypted environment-secret management.

```bash
hulak env --help
hulak env set API_KEY value --env prod
hulak env list --env staging
hulak env keys --env prod
```

Subcommands currently shown by the CLI:

- `set` (`add`)
- `get`
- `list` (`ls`)
- `keys` (`key`)
- `delete` (`rm`, `remove`)
- `edit`
- `import-key`
- `export-key`
- `add-recipient`
- `remove-recipient`
- `list-recipients`

Important: the command surface is present, but the current handlers still print `not yet implemented` for the scaffolded `env` subcommands.

### `help`

Print the top-level command list.

```bash
hulak help
```

For command-specific help, use:

```bash
hulak <command> --help
```

## Supported Root Flags (Shorthand)

These are still supported. They are useful when you want the older root-flag style or need file-name search behavior.

| Flag | Meaning | Example |
| --- | --- | --- |
| `-env`, `--environment` | Select an environment for root-flag execution | `hulak -env prod -fp requests/get-user.hk.yaml` |
| `-fp`, `--file-path` | Run one exact file path | `hulak -fp requests/get-user.hk.yaml` |
| `-f`, `--file` | Search for matching file names recursively and run matches | `hulak -f getUser` |
| `-dir` | Run a directory concurrently | `hulak -dir ./requests/` |
| `-dirseq` | Run a directory sequentially | `hulak -dirseq ./requests/` |
| `-debug` | Enable debug output | `hulak -fp requests/get-user.hk.yaml -debug` |
| `-v`, `--version` | Print version | `hulak --version` |
| `-h`, `--help` | Print help | `hulak --help` |

Use the shorthand form when it fits your workflow, but prefer `hulak run ...` in examples and onboarding material.
