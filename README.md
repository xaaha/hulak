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

### Prefer plaintext `env/` files instead of encrypted secrets?

```bash
hulak init classic
```

Plaintext mode is fully supported. See [docs/environment.md](./docs/environment.md) for more info

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
