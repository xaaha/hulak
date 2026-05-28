// Snapshot tests freeze the public command surface of `hulak secrets` so the
// upcoming restructure (#230) can move code without changing observable
// behavior. Each snapshot is a frozen literal in this file. When a refactor
// intentionally changes the surface, update the literal in the same PR — the
// diff is the review artifact.
//
// Covered:
//   - secrets-level subcommand inventory: name | aliases | flags
//   - secrets-level help text (Long, COMMANDS block, EXAMPLES)
//
// Behavioral tests for each command already live next to it
// (env_crud_test.go, env_keys_test.go, env_list_test.go,
// env_edit_test.go, env_backup_test.go, etc.).
package userflags

import (
	"flag"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// ansiSeq matches every ANSI CSI escape sequence used by utils.PrintSectionHeader
// and friends. Stripping these before snapshotting keeps the literal stable
// regardless of terminal capability detection.
var ansiSeq = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiSeq.ReplaceAllString(s, "")
}

// formatCommandSurface returns one line per subcommand of cmd in the form
// `name | aliases | flags`, alpha-sorted by name. Aliases and flags use
// comma-joined values; an empty list renders as `-` so blank columns are
// visually obvious in a diff.
//
// Flags are filtered: the auto-injected `h`/`help` pair from command.Execute
// is omitted because it isn't part of any FlagSet at construction time — it
// gets registered lazily on first parse. Excluding them keeps the snapshot
// stable regardless of whether something has executed the command yet.
func formatCommandSurface(cmd *command) string {
	subs := append([]*command(nil), cmd.SubCommands...)
	slices.SortFunc(subs, func(a, b *command) int {
		return strings.Compare(a.Name, b.Name)
	})

	var sb strings.Builder
	for _, s := range subs {
		aliases := strings.Join(s.Aliases, ",")
		if aliases == "" {
			aliases = "-"
		}

		var flags []string
		if s.Flags != nil {
			s.Flags.VisitAll(func(f *flag.Flag) {
				if f.Name == "h" || f.Name == "help" {
					return
				}
				flags = append(flags, f.Name)
			})
		}
		sort.Strings(flags)
		flagStr := strings.Join(flags, ",")
		if flagStr == "" {
			flagStr = "-"
		}

		fmt.Fprintf(&sb, "%s | %s | %s\n", s.Name, aliases, flagStr)
	}
	return sb.String()
}

func getSecretsCmd(t *testing.T) *command {
	t.Helper()
	root := subCommands()
	cmd := root.findSub("secrets")
	if cmd == nil {
		t.Fatal("secrets subcommand missing")
	}
	return cmd
}

// TestSecretsSurfaceSnapshot freezes the secrets-level subcommand surface.
// Any rename, alias change, flag add/remove, or new/removed subcommand
// surfaces here as a diff that must be acknowledged in the PR.
func TestSecretsSurfaceSnapshot(t *testing.T) {
	got := formatCommandSurface(getSecretsCmd(t))

	const want = `backup | - | f,force,o,out
edit | - | env,environment
identity | - | -
keys | key | env,environment,search,show
list | ls | -
migrate | - | -
restore | - | f,force
sync | rotate | -
`

	if got != want {
		t.Errorf("secrets surface drift\n--- want\n%s\n--- got\n%s", want, got)
	}
}

// TestSecretsIdentitySurfaceSnapshot freezes the `secrets identity` subgroup
// shape. Catches name or alias drift on the 8 leaves moved here from the
// secrets top level.
func TestSecretsIdentitySurfaceSnapshot(t *testing.T) {
	root := subCommands()
	secrets := root.findSub("secrets")
	if secrets == nil {
		t.Fatal("secrets subcommand missing")
	}
	identity := secrets.findSub("identity")
	if identity == nil {
		t.Fatal("secrets identity subcommand missing")
	}

	got := formatCommandSurface(identity)

	const want = `add-recipient | - | allow-rsa,github,keyserver,name,stdin
export | - | o,out
generate | gen | name
import | - | force,name,stdin
list | ls | -
list-recipients | - | -
remove-recipient | - | -
rotate | - | -
`

	if got != want {
		t.Errorf("secrets identity surface drift\n--- want\n%s\n--- got\n%s", want, got)
	}
}

// TestSecretsKeysSurfaceSnapshot freezes the `secrets keys` subgroup. Added
// in Chunk 7 of the #230 restructure when `keys` gained nested commands
// (list/set/get/delete). The bare `secrets keys --env prod` legacy shorthand
// is preserved by the parent's Run handler — not captured here because the
// snapshot only describes children.
func TestSecretsKeysSurfaceSnapshot(t *testing.T) {
	root := subCommands()
	secrets := root.findSub("secrets")
	if secrets == nil {
		t.Fatal("secrets subcommand missing")
	}
	keys := secrets.findSub("keys")
	if keys == nil {
		t.Fatal("secrets keys subcommand missing")
	}

	got := formatCommandSurface(keys)

	const want = `delete | rm | env,environment
get | - | env,environment
list | ls | env,environment,search,show
set | add | env,environment,stdin,t,type
`

	if got != want {
		t.Errorf("secrets keys surface drift\n--- want\n%s\n--- got\n%s", want, got)
	}
}

// TestSecretsHelpSnapshot freezes the parent help block for `hulak secrets`.
// Catches changes to Long, Examples, and the COMMANDS section ordering.
func TestSecretsHelpSnapshot(t *testing.T) {
	cmd := getSecretsCmd(t)
	got := stripANSI(captureStdout(t, cmd.printHelp))

	const want = `Manage environment secrets stored in the encrypted vault (.hulak/store.age).

Secrets are organized by environment (e.g. global, staging, prod).
Environment lifecycle and vault-file ops live at this level. Key-level
CRUD lives under ` + "`secrets keys`" + `. When --env is omitted on a command
that takes one, you'll be prompted to pick an environment.

'env' is retained as an alias for backward compatibility with pre-0.3 docs.

COMMANDS
  backup           - Create a backup of the encrypted store
  edit             - Edit secrets interactively
  identity         - Manage age identities and recipients
  keys (key)       - Manage keys within an environment
  list (ls)        - List environment names
  migrate          - Migrate env/*.env files to the encrypted vault
  restore          - Restore the encrypted store from a backup
  sync (rotate)    - Re-encrypt the store to current recipients

EXAMPLES
  $ hulak secrets list
    List environment names defined in the vault
  $ hulak secrets keys list --env prod
    List keys in the prod environment (values masked)
  $ hulak secrets keys set API_KEY sk-123 --env prod
    Set a key in the prod environment
  $ hulak secrets keys get API_KEY --env staging
    Get a value from the staging environment
  $ hulak secrets keys delete OLD_KEY --env staging
    Delete a key from the staging environment

LEARN MORE
  Use ` + "`hulak <command> --help`" + ` for more information about a command.
`

	if got != want {
		t.Errorf("secrets help drift\n--- want\n%s\n--- got\n%s", want, got)
	}
}
