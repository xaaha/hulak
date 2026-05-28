# Migrating from `env/` to the encrypted vault

This page walks you through moving an existing classic-mode hulak project (`env/*.env`) into the encrypted vault (`.hulak/store.age`). Read it once. Run the commands. Delete `env/`.

## Why migrate

- **Encrypted at rest.** `store.age` is an age-encrypted blob. Safe to commit. Safe to share via git.
- **Audit trail.** Every change to a secret shows up in `git log` like any other file.
- **Team sharing.** Add teammates as recipients with age or SSH keys. They decrypt with their own private key. No more sharing `.env` files in Slack DMs.
- **CI-friendly.** Set `HULAK_MASTER_KEY` once in CI. The same `hulak run` command works locally and in pipelines.

## Before you start

1. **Commit your current state.** A clean working tree is your rollback safety net.
   ```bash
   git status            # should be clean
   git log -1 --oneline  # remember this SHA
   ```
2. **Verify your hulak version.** Migration uses commands added in v0.3.
   ```bash
   hulak version
   ```
3. **Choose an identity source.** Hulak needs a private key to decrypt the vault. You have three options. Most users want the first.
   - **Generate a new age keypair** (default). Hulak creates `~/.config/hulak/identity.txt`.
   - **Use your SSH key.** Run `hulak init --ssh` first to bootstrap with `~/.ssh/id_ed25519`, then migrate.
   - **Bring your own age key.** Generate it elsewhere, then import with `hulak secrets identity import`.

See [store.md#identity](./store.md#identity) for the full identity priority chain.

## Migrate

```bash
hulak secrets migrate
```

The command reads every file in `env/`, encrypts the keys into `.hulak/store.age`, and writes the recipient list. Annotated output looks like this:

```text
✓ Generated identity at ~/.config/hulak/identity.txt
✓ Wrote recipients to .hulak/recipients.txt
✓ Migrated env/global.env  -> store.age[global]  (4 keys)
✓ Migrated env/prod.env    -> store.age[prod]    (7 keys)
✓ Migrated env/staging.env -> store.age[staging] (5 keys)
⚠ Back up ~/.config/hulak/identity.txt now. Losing it locks the vault.
```

Encoding edge cases like non-ASCII keys are handled per issue [#147](https://github.com/xaaha/hulak/issues/147). If migration warns about a key, the source line is shown. Fix it in `env/<name>.env` and re-run.

## Verify

Three quick checks. Do all three before trusting the migration.

```bash
# 1. Environments are present
hulak secrets list

# 2. Keys per environment
hulak secrets keys list --env prod

# 3. End-to-end smoke test (run a real request with the new backend)
hulak run requests/health.hk.yaml --env prod
```

If something is off, see the [rollback](#rollback) section. Your `env/` directory is still on disk.

## Commit

```bash
git add .hulak/store.age .hulak/recipients.txt
git commit -m "migrate secrets to encrypted vault"
```

Safe to commit:
- `store.age`. Encrypted. Only recipients in `recipients.txt` can decrypt it.
- `recipients.txt`. Public keys only.

**Never commit** `~/.config/hulak/identity.txt`. That is your private key. The file is created with mode `0600` for this reason.

## Back up your identity (do this NOW)

Losing `~/.config/hulak/identity.txt` without a backup means losing access to your vault. Pick one. Preferably do both.

```bash
# Option A. Export and paste into a password manager.
hulak secrets identity export
# AGE-SECRET-KEY-1QF...
# (paste into 1Password / Bitwarden / etc.)

# Option B. Add a second recipient.
# Generate a backup keypair on a different machine or USB. Then:
hulak secrets identity add-recipient <backup-pubkey> --name "backup-laptop"
```

See [store.md#identity-backup](./store.md#identity-backup) for details.

## Clean up

Once you have verified end-to-end and backed up the identity, remove the plaintext `env/`:

```bash
rm -r env/
git add -A
git commit -m "remove plaintext env/ (migrated to vault)"
```

## Rollback

Until `env/` is deleted, classic mode is one rename away.

```bash
# Move the vault out of the way
mv .hulak/store.age .hulak/store.age.bak

# Verify hulak falls back to env/
hulak run requests/health.hk.yaml --env prod
```

If you have already deleted `env/`, recover it from git:

```bash
git checkout <pre-migration-sha> -- env/
```

That is why we wrote down the SHA in "Before you start".

## Troubleshooting

| Symptom | See |
| ------- | --- |
| "no identity found" | [store.md#identity](./store.md#identity) |
| "decryption failed" | [store.md#identity-backup](./store.md#identity-backup) |
| Non-ASCII or special-character keys | issue [#147](https://github.com/xaaha/hulak/issues/147) |
| Merge conflicts on `store.age` | [store.md#merge-conflicts-on-recipientstxt-and-storeage](./store.md#merge-conflicts-on-recipientstxt-and-storeage) |

## What's next

- [docs/store.md](./store.md). Full encryption model and team sharing.
- [docs/versioning.md](./versioning.md). Git-versioned vault patterns.
- [docs/cli.md#secrets](./cli.md#secrets). Complete `secrets` command reference.
