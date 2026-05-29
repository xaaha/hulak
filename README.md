<p align="center">
<pre>

██╗░░██╗██╗░░░██╗██╗░░░░░░█████╗░██╗░░██╗
██║░░██║██║░░░██║██║░░░░░██╔══██╗██║░██╔╝
███████║██║░░░██║██║░░░░░███████║█████═╝░
██╔══██║██║░░░██║██║░░░░░██╔══██║██╔═██╗░
██║░░██║╚██████╔╝███████╗██║░░██║██║░╚██╗
╚═╝░░╚═╝░╚═════╝░╚══════╝╚═╝░░╚═╝╚═╝░░╚═╝

</pre>
</p>

<h3 align="center">
  Git-native API client with encrypted secrets.
</h3>

<p align="center">
  <sub>REST &middot; GraphQL &middot; OAuth</sub>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> &bull;
  <a href="#graphql-explorer">GraphQL Explorer</a> &bull;
  <a href="#project-layout">Project Layout</a> &bull;
  <a href="#documentation">Documentation</a>
</p>

---

### Run one request, a whole directory, or stay interactive

<img alt="Concurrent Execution" src="./assets/concurrent.gif" width="720" />

```bash
hulak run ./requests/
```

Hulak runs request files directly from your project. It supports concurrent directory execution. It falls back to an interactive picker when you simply run `hulak`.

### Dedicated GraphQL Explorer

<img alt="GraphQL Explorer" src="./assets/gql.gif" width="720" />

Browse schemas from multiple endpoints. Search operations. Build queries interactively. Execute inline. Save generated files from the terminal.

## Quick Start

### Install

```bash
brew install xaaha/tap/hulak
```

Other install options:

- `go install github.com/xaaha/hulak@latest`
- Build from source with `go build -o hulak`

### Path A. API client with encrypted secrets (default)

```bash
mkdir my-apis && cd my-apis
hulak init # creates .hulak/store.age + identity
```

Scaffold a starter request, to quickly check how a request file looks run:

```bash
hulak example api  # writes example-api.hk.yaml you can run
```

> [!Info]
> For Other types run: `hulak example`. `example` sub-command gives you a quick way to write a request file you can modify. For more info run `hulak example -h`

To set up a secret you can run:

```bash
hulak secrets keys set placeholder  https://jsonplaceholder.typicode.com/posts -env prod

```

Now, in your `example-api.hk.yaml` file, you can reference this secret:

```yaml
method: POST
url: "{{.placeholder}}"
# rest of the body of the file remains same
```

Run the request:

```bash
hulak run example-api.hk.yaml --env prod
```

### Prefer plaintext `env/*.env` files instead of encrypted secrets?

```bash
hulak init classic
```

Plaintext mode is fully supported. See [docs/environment.md](./docs/environment.md) for more info

## Vault vs Classic

Hulak runs in two modes. Pick once during `hulak init`. You can migrate later.

- **Vault (default):** secrets live in `.hulak/store.age`, encrypted with an age or SSH keypair. Safe to commit. Teams share via a recipients file. See [docs/store.md](./docs/store.md).
- **Classic:** secrets live in plaintext `env/*.env` files. Simpler, no encryption. Add `env/` to `.gitignore`. See [docs/environment.md](./docs/environment.md).

Running classic and want to switch? See [docs/migrating-to-vault.md](./docs/migrating-to-vault.md).

## Commands

| Command   | Purpose                                | Read more                                                  |
| --------- | -------------------------------------- | ---------------------------------------------------------- |
| `run`     | Execute request file(s) or a directory | [body.md](./docs/body.md), [actions.md](./docs/actions.md) |
| `gql`     | GraphQL explorer TUI                   | [graphql-explorer.md](./docs/graphql-explorer.md)          |
| `secrets` | Encrypted vault CRUD                   | [store.md](./docs/store.md)                                |
| `init`    | Initialize a hulak project             | [store.md](./docs/store.md)                                |
| `migrate` | Postman to hulak conversion            | [migrating-to-vault.md](./docs/migrating-to-vault.md)      |
| `example` | Scaffold sample request files          | —                                                          |
| `doctor`  | Check project health                   | —                                                          |
| `version` | Print version                          | —                                                          |

Run `hulak <command> --help` for flags and per-command examples.

### Picker behavior

Omitting `--env` opens an interactive picker.

- `hulak run` and `hulak gql` only prompt when files reference `{{.key}}`.
- `hulak secrets` subcommands prompt every time (except `secrets list`).
- Non-interactive shells require `--env <name>`.

## Common Pitfalls

- **Never commit `~/.config/hulak/identity.txt`.** That is your private key. Mode 0600. Back it up first. See [docs/store.md#identity-backup](./docs/store.md#identity-backup).
- **On `hulak init`, `-env` creates env files. It is a setup flag, not a runtime selector.** `hulak init -env staging prod` scaffolds two envs.
- **`env` is an alias for `secrets`.** `hulak env list` works the same as `hulak secrets list`.
- **GUI editors need a wait flag for `secrets edit`.** Use `EDITOR="code -w"` or `EDITOR="zed --wait"`. Without it the editor returns immediately and changes are lost.
- **Merge conflicts on `store.age` need a recipe.** See [docs/versioning.md#merge-conflicts](./docs/versioning.md#merge-conflicts).

## Project layout

```text
my-project/
├── .hulak/
│   ├── store.age          # encrypted secrets (safe to commit)
│   └── recipients.txt     # public keys of recipients (safe to commit)
├── requests/
│   ├── create-user.hk.yaml
│   └── get-user.hk.yaml
└── (your project files)

~/.config/hulak/
└── identity.txt           # YOUR private key. NEVER commit. Mode 0600.
```

## GraphQL Explorer

Start the explorer with a file or a directory:

```bash
hulak gql e2etests/gql_schemas/countries.yml
hulak gql .
hulak gql -env staging ./collections/graphql
```

Read the full guide in [docs/graphql-explorer.md](./docs/graphql-explorer.md).

## Documentation

Start here for the full reference:

- [Encrypted Store](./docs/store.md). Encryption model, team sharing, CI.
- [Migrating to the Vault](./docs/migrating-to-vault.md). From `env/` to `.hulak/`.
- [Versioning Your Vault](./docs/versioning.md). Git workflow for secrets.
- [Comparison](./docs/comparison.md). Hulak vs SOPS, Bruno, and friends.
- [Request Body](./docs/body.md)
- [Actions](./docs/actions.md)
- [Environment Secrets (classic mode)](./docs/environment.md)
- [Response Files](./docs/response.md)
- [GraphQL Explorer](./docs/graphql-explorer.md)
- [Auth 2.0](./docs/auth20.md)

For the live command surface, run:

```bash
hulak help
hulak <command> --help
```

## Schema Support

The Hulak schema is available in the [Schema Store](https://www.schemastore.org/), so editors that support Schema Store can automatically enable completion for `.hk.yaml` and `.hk.yml` files.

You can also point your YAML language server directly at:

```text
https://raw.githubusercontent.com/xaaha/hulak/refs/heads/main/assets/schema.json
```

## Contributing

```bash
git clone https://github.com/xaaha/hulak.git
cd hulak
mise install
```

See [CONTRIBUTING.md](./CONTRIBUTING.md) for the full development workflow.

## Support the Project

If Hulak is useful to you, open an issue, suggest a feature, send a pull request, or sponsor the project.
