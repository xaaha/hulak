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

> [!Tip]
> This page documents the current command surface. For the most up to date flags and subcommands, run `hulak <command> --help`. That output is generated from the same source as the binary.

## Command Index

| Command   | Purpose                                      | Example                               |
| --------- | -------------------------------------------- | ------------------------------------- |
| `run` | Run API request file(s) or directory | `hulak run path/to/file.yaml` |
| `version` | Print hulak version | `hulak version` |
| `init` | Initialize a hulak project | `hulak init` |
| `example` | Scaffold an example request file | `hulak example api` |
| `migrate` | Migrate Postman collections to hulak format | `hulak migrate collection.json` |
| `doctor` | Check project health | `hulak doctor` |
| `gql` (alias: `graphql`) | Open the GraphQL explorer | `hulak gql .` |
| `secrets` (alias: `env`) | Manage encrypted environment secrets | `hulak secrets list` |
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
- **`hulak secrets edit`**: always opens the picker. There is no default. Pass `--env <name>` (including for new envs you want to create).
- **`hulak secrets keys set/get/delete/list`**: open the interactive picker when `--env` is omitted, same as `edit`. Pass `--env <name>` to bypass the picker.
- **`hulak secrets create`**: requires `--env NAME` (no picker — you're inventing a new name).
- **`hulak secrets delete` (env-level), `rename`**: target an existing env explicitly; `delete` falls back to the picker if `--env` is omitted, `rename` takes positional `OLD NEW`.
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
hulak run path/to/file.yaml --ssh-identity ~/.ssh/work_ed25519
hulak run path/to/file.yaml --dry-run
hulak run path/to/file.yaml --dry-run --show
```

Supported flags:

| Flag | Meaning |
| ---- | ------- |
| `--debug` | Enable debug mode |
| `--dry-run` | Print the built request and exit without sending it |
| `--env`, `--environment` | Environment to use |
| `--q`, `--quiet` | Suppress the end-of-run summary table |
| `--seq`, `--sequential` | Run directory files sequentially |
| `--show` | Reveal sensitive headers (Authorization, Cookie, etc.) in --dry-run output |
| `--ssh-identity` | Path to SSH private key for vault decryption |
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

By default, creates an encrypted vault (.hulak/store.age) with an age keypair.
Use --ssh to bootstrap with your default SSH ed25519 key (~/.ssh/id_ed25519),
or --ssh-identity <path> for a custom key.

To scaffold an example request file, use 'hulak example <type>' after init.
Run 'hulak init classic' (aliases: plain, no-vault) to use the plaintext env/
layout instead. Use -env to scaffold specific environments.

```bash
hulak init
hulak init --ssh
hulak init --ssh-identity ~/.ssh/work_ed25519
hulak init -env staging prod
hulak example api
hulak init classic
```

Supported flags:

| Flag | Meaning |
| ---- | ------- |
| `--env` | Create specific environment files instead of the default setup |
| `--ssh` | Use SSH ed25519 key (~/.ssh/id_ed25519) instead of generating an age keypair |
| `--ssh-identity` | Path to SSH private key (implies --ssh; overrides the default path) |

Notes:

- `hulak init` creates the default setup. That means an encrypted vault at `.hulak/store.age`, an age keypair at `~/.config/hulak/identity.txt`, and the example API options file.
- Run `hulak init classic` for the legacy plaintext `env/` layout.
- On `init`, `-env` is a **boolean setup flag**, not an environment selector. It tells Hulak to create the named env files you pass after it.

### `example`

Scaffold a starter request file into the current directory.

Each type writes a self-contained, schema-valid file that runs against a
public test API (jsonplaceholder, httpbin, trevorblades countries). The
'options' type writes a reference card listing every available request
field — it's not runnable on its own.

Use -o/--out to write somewhere other than the current directory. Pass a
directory to keep the canonical filename, or a full path to rename. Parent
directories are created on demand.

Idempotent: re-running for a path that already exists keeps the existing
file untouched.

```bash
hulak example api
hulak example formdata
hulak example urlencoded
hulak example graphql
hulak example auth
hulak example options
hulak example api -o requests/
hulak example api -o requests/health.hk.yaml
hulak example
```

Supported flags:

| Flag | Meaning |
| ---- | ------- |
| `--o` | Output path. Directory (ends with '/' or no .yaml/.yml extension) → file lands inside with the canonical name; otherwise treated as a full file path. Parent directories are created. |
| `--out` | Output path. Directory (ends with '/' or no .yaml/.yml extension) → file lands inside with the canonical name; otherwise treated as a full file path. Parent directories are created. |

### `migrate`

Convert Postman v2.1 environment and collection JSON exports into hulak .hk.yaml and .env files.

Only Postman collections and environments are supported at this time.
To migrate plaintext env/ files to the encrypted vault, use 'hulak secrets migrate' instead.

```bash
hulak migrate collection.json
hulak migrate env.json collection.json
```

### `doctor`

Inspect your hulak project for common issues.

Vault backend: identity, store, recipients, and drift checks.
Classic backend: .gitignore, file permissions, git history.

```bash
hulak doctor
hulak doctor --fix
hulak doctor --fix --yes
hulak doctor --json
```

Supported flags:

| Flag | Meaning |
| ---- | ------- |
| `--fix` | Auto-repair safe issues (chmod, .gitignore) |
| `--json` | Output findings as JSON to stdout |
| `--yes` | Skip confirmation prompts (use with --fix) |

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

Three concern-scoped groups live here:
  - this level: environment listing, edit, backup/restore, sync, migrate.
  - `secrets keys ...`     for key-level CRUD inside an environment.
  - `secrets identity ...` for age identities and recipient management.

When --env is omitted on a command that takes one, you'll be prompted
to pick an environment from a TUI list.

'env' is kept as an alias of `secrets` for backward compatibility. See [docs/store.md](./store.md) for the full encryption model and team-sharing flows.

Aliases:

- `env`

```bash
hulak secrets list
hulak secrets keys list --env prod
hulak secrets keys set API_KEY sk-123 --env prod
hulak secrets keys get API_KEY --env staging
hulak secrets keys delete OLD_KEY --env staging
```

| Subcommand | Notes |
| ---------- | ----- |
| `create` | Create a new empty environment |
| `delete` (`rm`) | Delete an environment |
| `list` (`ls`) | List environment names |
| `keys` (`key`) | Manage keys within an environment |
| `edit` | Edit secrets interactively |
| `identity` | Manage age identities and recipients |
| `rename` (`mv`) | Rename an environment (unix-style mv) |
| `sync` (`rotate`) | Re-encrypt the store to current recipients |
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
