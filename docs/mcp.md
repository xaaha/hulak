# MCP Server

`hulak mcp` starts a [Model Context Protocol](https://modelcontextprotocol.io) server over stdio, exposing your hulak request files to an AI agent (Claude Code, Cursor, Zed, and other MCP clients).

It is an adapter only: every tool wraps the same hulak packages the CLI uses. There is no separate HTTP, secret, or parsing path — a request behaves identically whether you run it from the terminal or an agent calls it.

## Why

So you can ask an agent to work with your API collection in plain language — "list the requests in the api project", "dry-run `login` against staging", "call `getUser` and show me the response", "write a new request that posts to `/orders`" — without leaving the chat or hand-copying YAML.

## Configure your client

An MCP client launches the server as a subprocess. Point it at the `hulak` binary with the `mcp` subcommand and one or more projects.

Single project:

```json
{
  "mcpServers": {
    "hulak": {
      "command": "hulak",
      "args": ["mcp", "--project", "api=~/work/api-tests"]
    }
  }
}
```

Multiple projects with a default:

```json
{
  "mcpServers": {
    "hulak": {
      "command": "hulak",
      "args": [
        "mcp",
        "--project", "api=~/work/api",
        "--project", "mob=~/work/mobile",
        "--default-project", "api"
      ]
    }
  }
}
```

### Flags

| Flag                | Meaning                                                                                     |
| ------------------- | ------------------------------------------------------------------------------------------- |
| `--project`         | A named project as `name=path` (repeatable). Path accepts `~`. Must be a hulak project dir. |
| `--default-project` | Project assumed when a request name is unambiguous but the agent passes no `project`.       |

At least one `--project` is required. Each path must be a real hulak project (contains `.hulak/` or `env/`) or the server refuses to start — a mistyped path fails loudly instead of serving empty lists.

## Projects and ambiguity

Every tool takes an optional `project`. When omitted, the server searches all configured projects. If a request name exists in exactly one, it is used; if it exists in more than one, the server returns an error listing the choices and the agent must retry with an explicit `project`. The agent never guesses beyond this resolver.

## Tools

| Tool             | Access      | What it does                                                                              |
| ---------------- | ----------- | ----------------------------------------------------------------------------------------- |
| `list_requests`  | read-only   | List request files: name, project, path, kind, and referenced files (e.g. a GraphQL `.gql`). |
| `list_envs`      | read-only   | List environment **names** per project. Use one as the `env` argument below.              |
| `dry_run`        | read-only   | Resolve a request against an env and return the exact request that would be sent, unsent. |
| `call_request`   | destructive | Send the request and return status + body. Real network call.                             |
| `write_request`  | write       | Create (or overwrite) a request file from YAML content.                                   |

The agent discovers all of this — every tool, every argument — from the MCP handshake. No prompting from you about available options is needed.

### `call_request` arguments

| Argument  | Default | Notes                                                                        |
| --------- | ------- | ---------------------------------------------------------------------------- |
| `name`    | —       | Request name, with or without extension.                                     |
| `env`     | —       | Required. Environment to resolve secrets against.                            |
| `project` | —       | Optional; see ambiguity rules above.                                         |
| `save`    | `false` | The response is always returned. Set `true` to also write `{name}_response.json`. |
| `debug`   | `false` | Include full request, response headers, and TLS details.                     |
| `timeout` | `60s`   | Go duration. A `timeout:` field in the request file wins over this.          |

Unlike `hulak run`, an agent call does **not** save a response file by default — saving is opt-in so an agent doesn't litter the repo. The CLI stays save-by-default.

## Safety

- **Secret values never cross the agent boundary.** `list_envs` returns environment names only. No tool reads or returns a decrypted secret value or private key. The agent writes requests that reference `{{.token}}`; it never sees what `token` is.
- **No secret-mutating tools.** Creating/editing/deleting environments and keys stays a human-only CLI operation. There is intentionally no `set key` tool: the value would have to pass through the agent's context to reach it, which is itself a leak.
- **`write_request` is validated.** Content is checked against the hulak request schema (`assets/schema.json`) before it touches disk, so an agent can't write malformed or hallucinated YAML. It refuses to overwrite an existing file unless `overwrite` is set, and rejects paths that escape the project.
- **Vault projects need a non-interactive identity.** stdin is the JSON-RPC channel, so a passphrase prompt would hang the session. If a served project uses the encrypted vault, configure `HULAK_MASTER_KEY`, an SSH identity (`HULAK_SSH_IDENTITY`), or a passphrase-less identity before starting. The server checks this at startup and refuses to run otherwise.

## Maintenance note

`assets/schema.json` now does double duty: editor autocomplete (via Schema Store) **and** the `write_request` validation gate. Keep it aligned with the `yamlparser` validation rules so the two surfaces don't drift.
