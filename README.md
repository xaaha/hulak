<p align="center">
  <img alt="Hulak" src="./assets/logo.svg" width="280" />
</p>

<h3 align="center">
  API calls with simple YAML. No Electron. No login. No lag.
</h3>

<p align="center">
  A file-based API client for your terminal. Define requests in YAML, run them instantly, and keep everything in git.
  <br/>
  REST, GraphQL, and OAuth 2.0 without GUI overhead.
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> &bull;
  <a href="#graphql-explorer">GraphQL Explorer</a> &bull;
  <a href="#documentation">Documentation</a>
</p>

---

### Run one request, a whole directory, or stay interactive

<img alt="Concurrent Execution" src="./assets/concurrent.gif" width="720" />

```bash
hulak run ./requests/
```

Hulak runs request files directly from your project, supports concurrent directory execution, and falls back to an interactive picker when you simply run `hulak`.

### Dedicated GraphQL Explorer

<img alt="GraphQL Explorer" src="./assets/gql.gif" width="720" />

Browse schemas from multiple endpoints, search operations, build queries interactively, execute inline, and save generated files from the terminal.

## Quick Start

### Install

```bash
brew install xaaha/tap/hulak
```

Other install options:

- `go install github.com/xaaha/hulak@latest`
- build from source with `go build -o hulak`

### Initialize a project

```bash
mkdir my_apis && cd my_apis
hulak init
```

If you want multiple environment files right away:

```bash
hulak init -env staging prod
```

Hulak creates an `env/` directory for secrets used by request files that reference template values like `{{.key}}`.

### Create a request file

```yaml
# test.hk.yaml
method: Get
url: https://jsonplaceholder.typicode.com/todos/1
```

### Run it

```bash
hulak run test.hk.yaml

# use a specific environment
hulak run test.hk.yaml --env staging

# run every request in a directory concurrently
hulak run ./requests/

# or open the interactive picker
hulak
```

If the selected request does not use environment template values, Hulak runs it without requiring `env/` setup.

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
- [Request Body](./docs/body.md)
- [Actions](./docs/actions.md)
- [Environment Secrets](./docs/environment.md)
- [Response Files](./docs/response.md)
- [GraphQL Explorer](./docs/graphql-explorer.md)
- [Auth2.0](./docs/auth20.md)

If you want the full list of commands, flags, and subcommands, use [docs/cli.md](./docs/cli.md) or run:

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
