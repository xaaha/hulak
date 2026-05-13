# Versioning your vault with git

The encrypted vault (`.hulak/store.age`) is designed to be committed. This page covers the conventions that make commit history useful when the file in question is opaque ciphertext.

## Why commit `store.age`

- **Encrypted.** Without a private key in `recipients.txt`, the blob is opaque. Safe in public repos.
- **Audit trail.** Every secret rotation, addition, or removal lands as a git commit. `git log` answers "when did we change this?" for free.
- **Disaster recovery.** A stale clone is also a backup. No external secrets service required.
- **PR review for secret changes.** Same workflow as code. Propose, review, merge.

## What to commit, what to ignore

| File | Commit? | Why |
| ---- | ------- | --- |
| `.hulak/store.age` | **Yes** | Encrypted. Safe. |
| `.hulak/recipients.txt` | **Yes** | Public keys only. |
| `~/.config/hulak/identity.txt` | **No** | Private key. Mode `0600`. |
| `env/*.env` (classic mode) | **No** | Plaintext secrets. |

Your `.gitignore` should already contain `env/` if you started with classic mode. `hulak doctor --fix` will add it if missing.

## Day-to-day workflow

```bash
# Edit a secret
hulak secrets set STRIPE_SECRET_KEY sk_live_new_value --env prod

# Stage and commit
git add .hulak/store.age
git commit -m "rotate prod STRIPE_SECRET_KEY (incident #4521)"

# Push
git push
```

The commit message is the only narrative. The diff is ciphertext. Write descriptive messages: which key, which env, and why. Future you, reading `git log` six months from now, will thank present you.

## Recovering an old secret value

```bash
# See every change to the vault
git log --oneline -- .hulak/store.age

# Restore the vault to a previous revision
git checkout <sha> -- .hulak/store.age

# Read the value as of that commit
hulak secrets get STRIPE_SECRET_KEY --env prod

# Put HEAD back
git checkout HEAD -- .hulak/store.age
```

This works because age decryption does not depend on git state. Any historical `store.age` you can check out is a valid ciphertext.

## Auditing who changed what

```bash
git log --follow --pretty=format:"%h %an %ad %s" --date=short -- .hulak/store.age
```

Author, date, message. The diff itself is opaque. So commit messages do all the work. Pick a convention. Include the env name and either the key name or "(rotation)" in every commit subject.

## Merge conflicts

`store.age` is a binary blob. Git cannot merge it. See [store.md merge conflicts](./store.md#merge-conflicts-on-recipientstxt-and-storeage) for the full flow. Short version:

```bash
# Resolve recipients.txt by keeping the union of both branches
git checkout --theirs .hulak/store.age
hulak secrets rotate                     # re-encrypt to merged recipients
git add .hulak/store.age .hulak/recipients.txt
git commit
```

## Branches and PR review

A PR that touches `store.age` should describe the change in plain English. Reviewers cannot diff ciphertext. Conventions that work:

- Title: `secrets: rotate prod STRIPE_SECRET_KEY` (or similar)
- Description: which env, which keys, why
- Optionally: `hulak secrets keys --env <env>` output before and after in the PR description, so reviewers can verify keys-added vs keys-removed without decrypting

## Backup beyond git

- `git push` covers off-site backup for `store.age` if you have a remote. Treat your git remote as the canonical store.
- `~/.config/hulak/identity.txt` is **not** in git. Back it up separately. See [migrating-to-vault.md](./migrating-to-vault.md#back-up-your-identity-do-this-now).
- For air-gapped redundancy: `cp .hulak/store.age my-backup.age`. The file is already encrypted. Copy it anywhere.

## What's next

- [docs/store.md](./store.md). Encryption model.
- [docs/migrating-to-vault.md](./migrating-to-vault.md). Moving from `env/`.
- [docs/cli.md#secrets](./cli.md#secrets). `secrets` command reference.
