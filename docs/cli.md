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
| `run` | Run API request file(s) or directory | `hulak run path/to/file.yaml` |
| `version` | Print hulak version | `hulak version` |
| `init` | Initialize a hulak project | `hulak init` |
| `migrate` | Migrate Postman collections to hulak format | `hulak migrate collection.json` |
| `doctor` | Check project health | `hulak doctor` |
| `gql` | Open the GraphQL explorer | `hulak gql .` |
| `secrets` | Manage encrypted environment secrets | `hulak secrets list` |
| `help` | Show help for hulak | `hulak help` |

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

Execute one or more API request files.

Pass a file path to run a single request, or a directory to run all files in it.
Directories run concurrently by default; use --sequential for ordered execution.

```bash
hulak run path/to/file.yaml
hulak run path/to/file.yaml --env staging
hulak run path/to/dir/
hulak run path/to/dir/ --sequential
```

Supported flags:

| Flag | Meaning |
| ---- | ------- |
| `--debug` | Enable debug mode |
| `--env`, `--environment` | Environment to use |
| `--q`, `--quiet` | Suppress the end-of-run summary table |
| `--seq`, `--sequential` | Run directory files sequentially |
| `--timeout` | Per-request timeout, e.g. 5m or 90s (default 60s) |

Notes:

- `hulak run` accepts either a file path or a directory path.
- Directories run concurrently by default.
- `hulak run path/to/file.yaml --debug --env staging` is supported; trailing flags after the path are parsed correctly.

### `version`

Print the current hulak version.

Useful for bug reports and verifying installs.

```bash
hulak version
```

### `init`

Set up a new hulak project in the current directory.

By default, creates an encrypted vault (.hulak/store.age) plus an example 'apiOptions.hk.yaml'.
Run 'hulak init classic' (aliases: plain, no-vault) to use the plaintext env/
layout instead. Use -env to scaffold specific environments.

```bash
hulak init
hulak init -env staging prod
hulak init classic
hulak init plain
hulak init classic -env staging prod
hulak init classic --help
```

Supported flags:

| Flag | Meaning |
| ---- | ------- |
| `--env` | Create specific environment files instead of the default setup |

Notes:

- `hulak init` creates the default setup, including `env/global.env` and the example API options file.
- On `init`, `-env` is a **boolean setup flag**, not an environment selector. It tells Hulak to create the named env files you pass after it.

### `migrate`

Convert Postman v2.1 environment and collection JSON exports into hulak .hk.yaml and .env files.

```bash
hulak migrate collection.json
hulak migrate env.json collection.json
```

### `doctor`

Inspect your hulak project for common issues: missing .gitignore entries,
loose file permissions on env files, and secrets leaked into git history.

```bash
hulak doctor
```

### `gql`

Launch an interactive TUI to browse and run GraphQL operations
defined in your .yml/.yaml files.

Aliases:

- `graphql`

```bash
hulak gql .
hulak gql path/to/schema.yml
hulak gql -env staging .
```

Supported flags:

| Flag | Meaning |
| ---- | ------- |
| `--env`, `--environment` | Environment to use (skips interactive selector) |

Read the full guide in [graphql-explorer.md](./graphql-explorer.md).

### `secrets`

Manage environment secrets stored in the encrypted vault (.hulak/store.age).

Secrets are organized by environment (e.g. global, staging, prod).
The default environment is "global" unless --env is specified. See [docs/store.md](./store.md) for the full encryption model and team-sharing flows.

```bash
hulak secrets list
hulak secrets set API_KEY sk-123 --env prod
hulak secrets get API_KEY --env staging
hulak secrets keys --env prod
hulak secrets delete OLD_KEY
```

| Subcommand | Notes |
| ---------- | ----- |
| `set` (`add`) | Set a key-value pair |
| `get` (`g`, `show`, `view`) | Get a value by key |
| `list` (`ls`, `l`) | List environment names |
| `keys` (`key`) | List keys in an environment |
| `delete` (`rm`, `remove`, `del`) | Delete a key |
| `edit` | Edit secrets interactively |
| `import-key` (`import-identity`) | Import an age identity (private key) |
| `export-key` (`export-identity`) | Export the age identity (private key) |
| `add-recipient` | Add a recipient for shared vault access |
| `remove-recipient` | Remove a recipient |
| `list-recipients` | List all recipients |
| `rotate` (`sync`, `reencrypt`) | Re-encrypt the store to current recipients |
| `rotate-key` (`rotate-identity`) | Rotate your age identity (keypair) |
| `migrate` | Migrate env/*.env files to the encrypted vault |
| `backup` | Create a backup of the encrypted store |
| `restore` | Restore the encrypted store from a backup |

**GUI editors** for `edit`: pass the wait flag in `$EDITOR` so hulak blocks until you save. e.g. `EDITOR="zed --wait"` or `EDITOR="code -w"`. Without it, the editor returns immediately and the file is read back unchanged.

### `help`

Print the top-level hulak help.

For help on a specific command, use `hulak <command> --help` instead.

```bash
hulak help
hulak secrets --help
hulak secrets keys --help
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
