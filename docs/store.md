# Encrypted Store

Hulak supports two backends for storing environment variables:

| Backend | Layout                             | When                                   |
| ------- | ---------------------------------- | -------------------------------------- |
| Classic | `env/global.env`, `env/<name>.env` | Default for existing projects (legacy) |
| Vault   | `.hulak/store.age` (age-encrypted) | Recommended — encrypted at rest        |

If `.hulak/store.age` exists, hulak uses the vault. Otherwise it falls back to `env/`. You can have both during a migration period; vault always wins.

## Layout

```text
my-project/
├── .hulak/
│   ├── store.age          # encrypted secrets (safe to commit)
│   └── recipients.txt     # list of public keys that can decrypt (safe to commit)
├── env/                   # legacy, optional (vault takes priority if both exist)
│   └── global.env
└── requests/
    └── create_user.hk.yaml

~/.config/hulak/
└── identity.txt           # YOUR PRIVATE KEY — never commit
```

| File                           | Contains                             | Commit? |
| ------------------------------ | ------------------------------------ | ------- |
| `.hulak/store.age`             | Encrypted secrets blob               | Yes     |
| `.hulak/recipients.txt`        | Public keys of authorized decryptors | Yes     |
| `~/.config/hulak/identity.txt` | Your age private key                 | **No**  |

## How encryption works

Hulak uses [age](https://age-encryption.org/). A modern, file-based encryption tool with simple X25519 keys.

- Each user generates an age **keypair** (one public key, one private key)
- The **public key** can be shared freely. It only encrypts
- The **private key** decrypts ciphertext encrypted to your public key. Keep it secret
- A single `store.age` file can be encrypted to **multiple public keys** at once. Any one of the corresponding private keys can decrypt it. This is what makes team sharing possible.

## `recipients.txt` format

```
# .hulak/recipients.txt
# Alice (added 2026-03-15)
age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p

# Bob (added 2026-03-20)
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBobsPublicKeyHere

# Carol (added 2026-04-01)
age1yr5tpz76yxqeg5ktrp7g2wgkxfmq9rmef0gxhvsky2pyv6lsf8msgmd00n
```

- **One public key per line**. Both `age1...` and `ssh-ed25519` formats are supported.
- **`#` comments are supported.** Use them to label keys (name, date added, role).
- **Blank lines are ignored.**
- `ssh-rsa` keys are accepted with `--allow-rsa` but ed25519 is recommended.
- `ecdsa` keys are not supported (age limitation).

> [!Tip]
> Add a comment above each key with the person's name and the date you added them. Future-you will thank present-you when someone asks "wait, who is `age1ql3z...`?".

## Quick start

### Create a new vault project

```bash
mkdir my-apis && cd my-apis
hulak init
# ✓ Initialized vault at .hulak/
#   Public key:    age1ql3z...
#   Recipients:    .hulak/recipients.txt
#   Identity file: ~/.config/hulak/identity.txt
# ⚠ Back up the identity file — losing it means losing access to the vault.

hulak secrets set API_KEY --env prod
# Enter value for API_KEY: ▊
# ✓ Set API_KEY in prod

hulak run requests/create_user.yaml --env prod
```

> [!Important]
> Your private key in `~/.config/hulak/identity.txt` is the **only** way to decrypt `store.age`. If you lose it without a backup or another recipient, the encrypted data is unrecoverable.

### Migrate an existing classic project

```bash
hulak secrets migrate
# ✓ Migrated global.env → store.age[global] (3 keys)
# ✓ Migrated prod.env → store.age[prod] (5 keys)
# ⚠ Save this recovery key somewhere safe (you won't see it again):
#   AGE-SECRET-KEY-1QF...
```

The legacy `env/` directory is left in place. Vault takes priority while both exist. Delete `env/` once you're confident.

For the canonical step-by-step walkthrough, see [migrating-to-vault.md](./migrating-to-vault.md).

## Identity

Hulak needs a private key to decrypt `store.age`. It checks these sources:

| Source | When to use |
|--------|-------------|
| `HULAK_MASTER_KEY` env var | CI/CD pipelines — **explicit override**, never falls back |
| `~/.config/hulak/identity.txt` | Default — dedicated age keypair |
| `HULAK_SSH_IDENTITY` env var | Point at a specific SSH key |
| `~/.ssh/id_ed25519` | Auto-detected SSH key |

For decryption, hulak tries **every** identity it finds (except when `HULAK_MASTER_KEY` is set — that one is strict and fails hard if wrong). This means a stale `identity.txt` from another project won't block decryption if your SSH key is also a recipient.

If you use SSH for git, hulak can reuse your existing SSH key. No separate keypair needed. Just make sure your SSH public key is added as a recipient (via `--github` or directly).

```bash
# Use a specific SSH key for this run
hulak run requests/get-user.yaml --ssh-identity ~/.ssh/work_ed25519

# Or set it for the session
export HULAK_SSH_IDENTITY=~/.ssh/work_ed25519
```

## Identity backup

The single biggest pitfall is losing your private key. Two ways to protect against it:

### 1. Export to a password manager

```bash
hulak secrets export-key
# AGE-SECRET-KEY-1QF...
# (paste into 1Password / Bitwarden / etc.)
```

### 2. Add a second recipient

A second recipient (a backup keypair you keep on a USB stick, or a teammate) means losing one identity isn't fatal. See [team sharing](#team-sharing) below.

To restore on a new machine:

```bash
hulak secrets import-key ~/Downloads/identity-backup.txt
# ✓ Identity imported to ~/.config/hulak/identity.txt
```

## Team sharing

A single `store.age` can be encrypted to many recipients. Each teammate has their own private key; any of them can decrypt the shared file.

### Joining a team (with GitHub. Recommended)

If the new member uses SSH for git, they already have keys published on GitHub. No key exchange needed:

```bash
# === Existing team member ===
hulak secrets add-recipient --github alice --name Alice
# ✓ Fetched 2 ssh-ed25519 keys from https://github.com/alice.keys
# ✓ Added 2 recipients

git add .hulak/recipients.txt .hulak/store.age
git commit -m "add Alice as recipient"
git push

# === New member (Alice) ===
git pull
hulak secrets get API_KEY --env prod
# Just works — her ~/.ssh/id_ed25519 matches one of the recipients
```

This also works with self-hosted GitLab, Forgejo, or any server that publishes keys at `/<username>.keys`:

```bash
hulak secrets add-recipient --github alice --keyserver https://gitlab.company.com --name Alice
```

### Joining a team (with age keys)

If the new member prefers a dedicated age keypair (or is on a new machine and doesn't want `hulak init`'s vault-scaffolding side effects), use `gen-identity`:

```bash
# === New member's machine ===
hulak secrets gen-identity
# ✓ Identity written to ~/.config/hulak/identity.txt
# Send your public key to a vault member and have them run:
#   hulak secrets add-recipient age1bob...
# age1bob...   ← printed to stdout, pipe-friendly
```

Unlike `hulak init`, `gen-identity` only creates the global identity file — no `.hulak/` files in the current directory, so cloning the repo later works without conflicts.

The new member sends their **public key** (`age1bob...`) to an existing team member via Slack, email, or a PR. **Public keys are not secret.** Never share the private key from `~/.config/hulak/identity.txt`.

```bash
# === Existing team member ===
hulak secrets add-recipient age1bob... --name Bob
# ✓ Added 1 recipient

git add .hulak/recipients.txt .hulak/store.age
git commit -m "add Bob as recipient"
git push

# === New member ===
git pull
hulak secrets list   # ✓ works — Bob's identity can decrypt
```

### Leaving a team

```bash
hulak secrets remove-recipient Alice
# ✓ Removed recipient
# ⚠ Note: Alice can still decrypt copies of store.age from before this point.
#   Rotate upstream secrets if compromise is suspected.

git add .hulak/recipients.txt .hulak/store.age
git commit -m "remove Alice as recipient"
git push
```

### Why rotation matters

Removing a recipient prevents them from decrypting **future** versions of `store.age`. They can still decrypt any copy they already have. Local clones, old git commits, backups they made.

To truly revoke access:

1. `hulak secrets remove-recipient <pubkey>`
2. Rotate every secret the leaver could have read (API keys, DB passwords, OAuth client secrets, etc.) on the upstream service
3. `hulak secrets set <KEY> <new-value>` for each rotated secret
4. Commit and push

This is a fundamental property of asymmetric encryption. True of age, GPG, SOPS, git-crypt, and every similar tool. There is no scheme that can un-show plaintext.

### Self-removal guard

You cannot remove yourself if you are the only recipient. That would brick the store.

```bash
hulak secrets remove-recipient age1ql3z...
# error: refusing to remove the last recipient — the store would become
# unrecoverable. Add another recipient first, or delete .hulak/store.age manually
```

## Commands

For the full command reference. Flags, aliases, and examples. See the [`secrets` section in cli.md](./cli.md#secrets).

> [!Tip]
> Need a snapshot of `store.age`? Just `cp .hulak/store.age my-backup.age`. The file is already encrypted. Copy it anywhere. To restore, copy it back. No dedicated subcommand needed.

> [!Note]
> `hulak secrets edit` opens an interactive picker when `--env` is omitted. The same flow as `hulak run`. There is no silent "global" default for edit. Pass `--env <name>` (including for new envs you want to create).

## Merge conflicts on `recipients.txt` and `store.age`

Two teammates may add or remove recipients on parallel branches. The merge behavior:

- **`recipients.txt` is plain text.** Git merges it like any source file. If both branches added different recipients, you'll get a clean three-way merge. If both added a recipient at the same line, you'll get a textual conflict. Resolve by keeping both lines.
- **`store.age` is a binary blob.** Git can't merge it. Whichever branch's `store.age` you accept will be encrypted to **only that branch's recipient set**. The other branch's added recipient is silently dropped.

### How to resolve

After merging the text side cleanly, regenerate `store.age` to match the merged recipient list:

```bash
# Resolve recipients.txt manually so it's the union you want
git checkout --theirs .hulak/store.age   # pick one ciphertext to start from
hulak secrets rotate                     # re-encrypt to the current (merged) recipients.txt
git add .hulak/store.age .hulak/recipients.txt
git commit
```

`hulak secrets rotate` re-encrypts the store to the current `recipients.txt` without changing keys. Exactly what you need after a merge.

> [!Tip]
> For frequent recipient churn (large teams), consider squashing multiple `add-recipient` / `remove-recipient` commits into one to reduce merge surface area.

## CI / scripts

For non-interactive use (CI, scripts), set the master key as an environment variable:

```bash
export HULAK_MASTER_KEY="AGE-SECRET-KEY-1QF..."
hulak run requests/create_user.yaml --env prod
```

`HULAK_MASTER_KEY` takes precedence over `~/.config/hulak/identity.txt`. Store the value in your CI provider's secret manager (GitHub Actions secrets, GitLab CI variables, etc.).

> [!Caution]
> Environment variables are visible to other processes running as the same user. On Linux, `ps -e auxe` shows the env table for any process owned by you. In CI this is fine. The runner is single-tenant. On a shared workstation (e.g., a build server with multiple users sharing one account), prefer the file-based identity at `~/.config/hulak/identity.txt` instead of `HULAK_MASTER_KEY`.

### CI templates

**GitHub Actions:**

```yaml
jobs:
  test:
    runs-on: ubuntu-latest
    env:
      HULAK_MASTER_KEY: ${{ secrets.HULAK_MASTER_KEY }}
    steps:
      - uses: actions/checkout@v4
      - run: brew install xaaha/tap/hulak # or download from releases
      - run: hulak run requests/smoke.yaml --env staging
```

**GitLab CI:**

```yaml
test:
  variables:
    HULAK_MASTER_KEY: $HULAK_MASTER_KEY # set in project CI/CD variables, masked
  script:
    - hulak run requests/smoke.yaml --env staging
```

Mark `HULAK_MASTER_KEY` as a **masked** / **protected** variable in both systems so the value is redacted from build logs.

## If your identity is compromised

Treat a leaked private key (laptop stolen, accidental commit, etc.) like a leaked database password. Assume the worst and rotate. Steps:

1. **Rotate the keypair atomically.** From any machine that still has access:
   ```bash
   hulak secrets rotate-key
   ```
   This generates a fresh keypair, swaps it in `recipients.txt`, re-encrypts the store, and backs up the old key to `identity.txt.old` (in case you need to read past backups).
2. **Rotate every secret in the store upstream.** API keys, DB passwords, OAuth client secrets. Anything the leaker may have read. The leaker still has copies of `store.age` and the old identity from before the rotation; the only way to invalidate that data is to make it useless by changing the upstream values.
3. **Force teammates to pull.** They need the re-encrypted `store.age` so their next decrypt uses the new recipient list.
4. **Audit history.** `git log -- .hulak/store.age` shows when ciphertexts changed; cross-reference with whatever you know about when the leak happened.

This is the same playbook as SOPS, GPG, and git-crypt. Encryption can't un-show plaintext, but a fast rotation plus upstream secret change is the standard mitigation.

## Threat model: what hulak does and doesn't protect against

| Threat | Defended? |
|--------|-----------|
| Secrets accidentally committed to a public repo | ✅ Encrypted at rest; ciphertext is useless without a recipient's private key |
| Casual snooping (PR diffs, commit log) | ✅ Encrypted blob |
| Compromised CI without explicit credentials | ✅ CI must have its own identity (via `HULAK_MASTER_KEY` or a recipient key) |
| Stolen laptop with plaintext `identity.txt` | ❌ Mitigate with OS disk encryption; future: passphrase-protected identities |
| Malicious local process reading env vars | ❌ OS-level isolation problem (same caveat as `aws-vault exec`, `direnv`) |
| Insider with current access leaking secrets | ❌ Trust model assumes recipients are trustworthy |
| Removed member with old git clone | ⚠️ They retain history; rotate secret **values** at source after removal |
| Attacker with leaked key adds themselves as a recipient before you rotate | ⚠️ Mitigate with branch protection + CODEOWNERS on `.hulak/recipients.txt` so changes need review |

**Key invariant**: editing `recipients.txt` does nothing on its own. To produce a `store.age` that a new key can decrypt, you have to re-encrypt — which requires an existing recipient's private key. A public repo with the ciphertext is useless to an attacker who never had a recipient key.

**The git history caveat**: removing a recipient does not unleak past ciphertexts. Anyone who had a valid key at any point in history can still decrypt the snapshots they pulled. After removing a member, **rotate every secret value at the source** (DB password, API key, etc.) — that's the only way to invalidate already-read data.

## See also

- [Migrating from `env/` to the vault](./migrating-to-vault.md). Step-by-step walkthrough.
- [Versioning your vault with git](./versioning.md). Commit workflow.
- [Hulak vs SOPS, Bruno, and friends](./comparison.md). Positioning.

## Privacy

hulak makes no network calls except for the HTTP requests you author in your `.hk.yaml` files. That's the whole product. There is no telemetry, no analytics, no version-check ping, no error reporting. The CLI runs entirely against local files and the requests you author. If you observe a network call from hulak outside of running your own request files, that's a bug. Please file it.

## FAQ

**Is it safe to commit `store.age` and `recipients.txt`?**
Yes. `store.age` is an encrypted blob; without a private key in the recipient list, it's opaque. `recipients.txt` contains public keys only.

**Is it safe to commit `~/.config/hulak/identity.txt`?**
**No.** This is your private key. Anyone with it can decrypt every `store.age` you have access to. The file is created with mode `0600` for this reason. Don't include `~/.config/` in dotfile repos that are pushed publicly.

**What if I lose my private key?**
If you have no backup and no second recipient, the data is gone. Mitigations:

- `hulak secrets export-key` → save in a password manager
- `hulak secrets add-recipient <backup-key>` → second recipient as redundancy
- `cp .hulak/store.age my-backup.age` → snapshot the encrypted store (still needs a key to decrypt)

**Can I have both `env/` and `store.age`?**
Yes, during migration. Vault takes priority. After verifying everything works, delete `env/`.

**How big can a single value be?**
Functionally, anything. Practically, hulak warns at 64 KB and recommends `{{getFile "path"}}` for large blobs (certs, JSON fixtures). Those are read from disk on demand instead of decrypted on every invocation.

**Does this work with SSH keys?**
Yes. Both `ssh-ed25519` public keys (as recipients) and `~/.ssh/id_ed25519` private keys (as identities) are supported. See [Identity](#identity) and [Team sharing](#team-sharing) above. Use `--github <username>` to add teammates directly from their GitHub SSH keys.
