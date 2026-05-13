# Hulak compared to other tools

Hulak overlaps two ecosystems: encrypted secrets management and file-based HTTP clients. This page shows where hulak fits and where each alternative wins. Last reviewed: 2026-05.

## Secrets management

| Tool          | Encrypted at rest  | Multi-recipient |     Git-friendly     | SSH-key support | No external service | GUI required |
| ------------- | :----------------: | :-------------: | :------------------: | :-------------: | :-----------------: | :----------: |
| **hulak**     |      ✅ (age)      |       ✅        |   ✅ (binary blob)   |       ✅        |         ✅          |      ❌      |
| SOPS          | ✅ (age, PGP, KMS) |       ✅        | ✅ (YAML, JSON diff) |     partial     |   optional (KMS)    |      ❌      |
| age           |         ✅         |       ✅        |       ✅ (CLI)       |       ✅        |         ✅          |      ❌      |
| git-crypt     |      ✅ (GPG)      |       ✅        |   ✅ (transparent)   |       ❌        |         ✅          |      ❌      |
| direnv        |   ❌ (plaintext)   |       n/a       |     ✅ (.envrc)      |       n/a       |         ✅          |      ❌      |
| doppler       |  ✅ (server-side)  |       ✅        |          ❌          |       ❌        |      ❌ (SaaS)      |   ✅ (web)   |
| 1Password CLI |  ✅ (server-side)  |   ✅ (vaults)   |          ❌          |       ❌        |      ❌ (SaaS)      |   ✅ (app)   |

### When _not_ to choose hulak for secrets

- **Policy-as-code on secret access** (who can read what, audit logs, just-in-time access). Use HashiCorp Vault, or SOPS with KMS.
- **OS keyring or Touch ID integration.** Use 1Password CLI.
- **No git workflow at all (server-side dashboard).** Use Doppler or AWS Secrets Manager.

## API clients

| Tool      |      File-based      |     No login     |      GraphQL      | Concurrent run | Vault-integrated |  In terminal  |
| --------- | :------------------: | :--------------: | :---------------: | :------------: | :--------------: | :-----------: |
| **hulak** |      ✅ (YAML)       |        ✅        | ✅ (TUI explorer) |       ✅       |  ✅ (built-in)   |      ✅       |
| Postman   | ❌ (cloud workspace) |        ❌        |        ✅         |    partial     |        ❌        | ❌ (Electron) |
| Bruno     |   ✅ (collections)   |        ✅        |        ✅         |       ❌       |        ❌        | ❌ (Electron) |
| Insomnia  |       partial        | ❌ (auth needed) |        ✅         |       ❌       |        ❌        | ❌ (Electron) |
| HTTPie    |   ❌ (CLI ad-hoc)    |        ✅        |        ❌         |       ❌       |        ❌        |      ✅       |
| curl      |   ❌ (CLI ad-hoc)    |        ✅        |        ❌         |       ❌       |        ❌        |      ✅       |

### When _not_ to choose hulak as an API client

- **GUI for non-engineers.** Use Bruno or Postman.
- **API mocking or contract testing.** Use Postman, Prism, or Insomnia.
- **One-off ad-hoc requests.** Use curl or HTTPie. No project scaffolding needed.

## Why hulak exists

No tool in the secrets list also runs HTTP requests. No tool in the API client list also encrypts and version-controls its own secrets. Hulak exists because keeping these in two tools means glue. You copy `.env` files into Postman. You wrap curl in `sops` invocations. You export env vars via direnv to feed Insomnia. Hulak collapses the seam. Your secrets and your requests live in the same git repo. They share the same `{{.KEY}}` template syntax. They survive `git push` without leaking plaintext.

If you only need one half, that is fine. Hulak works as a pure secrets store or a pure API client. The integration is opt-in, not mandatory.

## What's next

- [docs/store.md](./store.md). Encryption model.
- [docs/cli.md](./cli.md). Full CLI reference.
- [docs/migrating-to-vault.md](./migrating-to-vault.md). Switch from `env/`.
