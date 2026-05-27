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

	const want = `add-recipient | - | allow-rsa,github,keyserver,name,stdin
backup | - | f,force,o,out
delete | rm | env,environment
edit | - | env,environment
export-key | export-identity | o,out
gen-identity | generate-identity | name
get | - | env,environment
import-key | import-identity | force,name,stdin
keys | key | env,environment,search,show
list | ls | -
list-identity | - | -
list-recipients | - | -
migrate | - | -
remove-recipient | - | -
restore | - | f,force
rotate-key | rotate-identity | -
set | add | env,environment,stdin,t,type
sync | rotate | -
`

	if got != want {
		t.Errorf("secrets surface drift\n--- want\n%s\n--- got\n%s", want, got)
	}
}

// TestSecretsHelpSnapshot freezes the parent help block for `hulak secrets`.
// Catches changes to Long, Examples, and the COMMANDS section ordering.
func TestSecretsHelpSnapshot(t *testing.T) {
	cmd := getSecretsCmd(t)
	got := stripANSI(captureStdout(t, cmd.printHelp))

	const want = `Manage environment secrets stored in the encrypted vault (.hulak/store.age).

Secrets are organized by environment (e.g. global, staging, prod).
When --env is omitted, you'll be prompted to pick an environment interactively.

'env' is retained as an alias for backward compatibility with pre-0.3 docs.

COMMANDS
  add-recipient                       - Add a recipient for shared vault access
  backup                              - Create a backup of the encrypted store
  delete (rm)                         - Delete a key
  edit                                - Edit secrets interactively
  export-key (export-identity)        - Export the age identity (private key)
  gen-identity (generate-identity)    - Generate a new age keypair without creating a vault
  get                                 - Get a value by key
  import-key (import-identity)        - Import an age identity (private key)
  keys (key)                          - List keys in an environment
  list (ls)                           - List environment names
  list-identity                       - List identities that can decrypt the vault
  list-recipients                     - List all recipients
  migrate                             - Migrate env/*.env files to the encrypted vault
  remove-recipient                    - Remove a recipient
  restore                             - Restore the encrypted store from a backup
  rotate-key (rotate-identity)        - Rotate your age identity (keypair)
  set (add)                           - Set a key-value pair
  sync (rotate)                       - Re-encrypt the store to current recipients

EXAMPLES
  $ hulak secrets list
    List environment names defined in the vault
  $ hulak secrets set API_KEY sk-123 --env prod
    Set a secret in the prod environment
  $ hulak secrets get API_KEY --env staging
    Get a secret from the staging environment
  $ hulak secrets keys --env prod
    List keys in the prod environment (values masked)
  $ hulak secrets delete OLD_KEY
    Delete a key from the default environment

LEARN MORE
  Use ` + "`hulak <command> --help`" + ` for more information about a command.
`

	if got != want {
		t.Errorf("secrets help drift\n--- want\n%s\n--- got\n%s", want, got)
	}
}
