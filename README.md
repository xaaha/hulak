<p align="center">
  <img alt="Hulak" src="./assets/logo.svg" width="280" />
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

#### Shell completion (go install / source builds)

Homebrew installs completion automatically. If you installed via `go install`
or built from source, opt in once:

```bash
# zsh
hulak completion zsh > "${fpath[1]}/_hulak"        # then restart your shell

# bash (macOS, Homebrew bash-completion)
hulak completion bash > $(brew --prefix)/etc/bash_completion.d/hulak

# bash (Linux)
hulak completion bash | sudo tee /etc/bash_completion.d/hulak >/dev/null
```

Zsh requires `autoload -Uz compinit && compinit` in your `.zshrc`.

### Path A. API client with encrypted secrets (default)

```bash
mkdir my-apis && cd my-apis
hulak init                                            # creates .hulak/store.age + identity
hulak secrets set Url https://api.example.com/v1 --env prod
```

Scaffold a starter request (runs as-is against a public test API):

```bash
hulak example api                                     # writes example-api.hk.yaml
```

Other types: `hulak example formdata`, `hulak example graphql` (alias `gql`), `hulak example auth`, `hulak example options` (reference card).

Or write your own:

```yaml
# test.hk.yaml
method: Get
url: "{{.Url}}/health"
```

Run it:

```bash
hulak run test.hk.yaml --env prod
hulak run ./requests/                                 # whole directory, concurrent
hulak                                                 # interactive picker
```

See [docs/store.md](./docs/store.md) for the full encryption model.

### Path B. Just the secrets store

If you only want the vault piece, the same `init` command works. Skip writing `.hk.yaml` files:

```bash
mkdir my-secrets && cd my-secrets
hulak init
hulak secrets set DATABASE_URL postgres://... --env prod
hulak secrets add-recipient --github alice --name Alice   # share with a teammate
git add .hulak/ && git commit -m "add prod secrets"
```

### Migrating from an older `env/` setup?

See [docs/migrating-to-vault.md](./docs/migrating-to-vault.md).

### Prefer plaintext `env/` files?

```bash
hulak init classic
```

Plaintext mode is fully supported for throwaway projects, or when secrets live entirely outside hulak. See [docs/environment.md](./docs/environment.md).

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

- [CLI Reference](./docs/cli.md)
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
