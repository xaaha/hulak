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
hulak secrets --help
```

For command-specific help, prefer `hulak <command> --help`.

## Command Index

| Command   | Purpose                                      | Example                               |
| --------- | -------------------------------------------- | ------------------------------------- |
| `run`     | Run one request file or a directory          | `hulak run requests/get-user.hk.yaml` |
| `version` | Print the current Hulak version              | `hulak version`                       |
| `init`    | Create project setup and env files           | `hulak init`                          |
| `migrate` | Convert Postman exports to Hulak files       | `hulak migrate collection.json`       |
| `doctor`  | Check project health                         | `hulak doctor`                        |
| `gql`     | Open the GraphQL explorer                    | `hulak gql .`                         |
| `secrets` | Inspect the environment-secrets command tree | `hulak secrets --help`                |
| `help`    | Show top-level help                          | `hulak help`                          |

## Core Behaviors

### Interactive mode

Running `hulak` with no file or directory target opens the interactive picker.

- Hulak asks you to choose a request file first.
- It only asks for an environment if the selected request uses template values like `{{.key}}`.
- In non-interactive shells, you should pass a file or directory target instead.

### Environment selection

When `--env` is omitted, behavior depends on the command:

- **`run` and `gql`**: open the interactive picker if the request files reference environment variables (`{{.key}}`). If a request needs no env, no picker.
- **`hulak secrets edit`**: always opens the picker — no default. Pass `--env <name>` (including for new envs you want to create).
- **`hulak secrets set`, `get`, `delete`, `keys`**: default to `global`. These are non-interactive operations on a known env; the default keeps scripts terse.
- **`hulak secrets list`**: takes no `--env` (it lists envs themselves).

All commands above accept `--env` / `--environment` to bypass any picker or default explicitly.

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

| Flag                     | Meaning                                                                                                                               |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| `--env`, `--environment` | Use a specific environment                                                                                                            |
| `--debug`                | Print request and response debug details                                                                                              |
| `--sequential`, `--seq`  | Run directory files one at a time                                                                                                     |
| `--timeout`              | Per-request timeout, e.g. `5m` or `90s`. Overrides `$HULAK_TIMEOUT`; YAML `timeout:` wins per file. See [body.md](./body.md#timeout). |

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

| Flag                     | Meaning                                          |
| ------------------------ | ------------------------------------------------ |
| `--env`, `--environment` | Use a specific environment and skip the selector |

Read the full guide in [graphql-explorer.md](./graphql-explorer.md).

### `secrets`

The `secrets` command tree manages secrets in the encrypted vault (`.hulak/store.age`). See [docs/store.md](./store.md) for the full encryption model and team-sharing flows.

```bash
hulak secrets --help
hulak secrets set API_KEY value --env prod
hulak secrets list
hulak secrets keys --env prod --show
hulak secrets get DB_URL --env staging        # raw to stdout — $(...) safe
hulak secrets edit                            # picks env interactively, no --env default
```

| Subcommand                | Notes                                                                                         |
| ------------------------- | --------------------------------------------------------------------------------------------- |
| `set` (`add`)             | Positional VALUE, secure prompt fallback, `--stdin` for scripts                               |
| `get`                     | Raw stdout, exit non-zero on missing key                                                      |
| `list` (`ls`)             | Environment names; styled header in TTY, plain when piped                                     |
| `keys` (`key`)            | Masked by default, `--show`, `--search` (substring or glob)                                   |
| `delete` (`rm`, `remove`) | Errors on missing key                                                                         |
| `edit`                    | TUI env picker if `--env` omitted; no global default; pass `--env <name>` to create a new env |
| `import-key`              | Import an age identity from a file or stdin                                                   |
| `export-key`              | Print or save your private key                                                                |
| `add-recipient`           | Authorize a new public key                                                                    |
| `remove-recipient`        | Revoke a public key                                                                           |
| `list-recipients`         | Show all authorized public keys                                                               |

**GUI editors** for `edit`: pass the wait flag in `$EDITOR` so hulak blocks until you save. e.g. `EDITOR="zed --wait"` or `EDITOR="code -w"`. Without it, the editor returns immediately and the file is read back unchanged.

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

| Flag                    | Meaning                                                    | Example                                           |
| ----------------------- | ---------------------------------------------------------- | ------------------------------------------------- |
| `-env`, `--environment` | Select an environment for root-flag execution              | `hulak -env prod -fp requests/get-user.hk.yaml`   |
| `-fp`, `--file-path`    | Run one exact file path                                    | `hulak -fp requests/get-user.hk.yaml`             |
| `-f`, `--file`          | Search for matching file names recursively and run matches | `hulak -f getUser`                                |
| `-dir`                  | Run a directory concurrently                               | `hulak -dir ./requests/`                          |
| `-dirseq`               | Run a directory sequentially                               | `hulak -dirseq ./requests/`                       |
| `-debug`                | Enable debug output                                        | `hulak -fp requests/get-user.hk.yaml -debug`      |
| `-timeout`              | Per-request timeout, e.g. `5m` or `90s`                    | `hulak -fp requests/get-user.hk.yaml -timeout 2m` |
| `-v`, `--version`       | Print version                                              | `hulak --version`                                 |
| `-h`, `--help`          | Print help                                                 | `hulak --help`                                    |

Use the shorthand form when it fits your workflow, but prefer `hulak run ...` in examples and onboarding material.
