// Contains command factories and handlers for hulak secrets set, get, and delete.
package userflags

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// MaxValueSizeWarnBytes is the soft per-value size threshold at `set` time.
// Above this, the user is warned and pointed at {{getFile "path"}} for blobs.
// Not a hard limit — the value is still written.
const MaxValueSizeWarnBytes = 64 << 10 // 64 KiB

// newEnvSetCmd returns the command struct for `hulak secrets set`.
func newEnvSetCmd() *command {
	setFs := flag.NewFlagSet("env set", flag.ContinueOnError)
	setEnv := registerEnvFlag(setFs, utils.DefaultEnvVal, "Environment to operate on")
	setStdin := setFs.Bool("stdin", false, "Read value from stdin")

	return &command{
		Name:    "set",
		Aliases: []string{"add"},
		Short:   "Set a key-value pair",
		Long:    "Store a secret in the encrypted vault.\n\nIf VALUE is omitted, you'll be prompted to enter it (no echo, no shell history).\nUse --stdin to pipe the value from standard input (useful for scripts).",
		Flags:   setFs,
		Args: []argDef{
			{Name: "key", Required: true, Desc: "Secret key name"},
			{Name: "value", Desc: "Secret value (omit to be prompted, or use --stdin)"},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets set API_KEY sk-123",
				Description: "Set a value in the default (global) environment",
			},
			{
				Command:     "hulak secrets set DB_URL --env prod",
				Description: "Prompt for the value (no shell history)",
			},
			{
				Command:     "echo -n \"$TOKEN\" | hulak secrets set TOKEN --stdin",
				Description: "Read value from stdin (scripts/CI)",
			},
			{
				Command:     "hulak secrets set FEATURE_FLAG true --env staging",
				Description: "Set a value in a specific environment",
			},
		},
		Run: func(args []string) error { return runEnvSet(args, *setEnv, *setStdin) },
	}
}

// runEnvSet handles `hulak secrets set KEY [VALUE]`.
//
// Resolution order for the value:
//  1. --stdin flag → read all of stdin
//  2. positional VALUE → use as-is
//  3. interactive prompt with no echo (only if stdin is a TTY)
//
// The read-modify-write of store.age is wrapped in WithStoreLock so concurrent
// `hulak secrets set` invocations cannot lose each other's edits.
func runEnvSet(args []string, envName string, useStdin bool) error {
	if len(args) == 0 {
		return errors.New("missing required argument: KEY")
	}
	key := args[0]

	if err := requireVaultProject(); err != nil {
		return err
	}
	if err := utils.ValidateEnvName(envName); err != nil {
		return err
	}

	value, err := resolveSetValue(args, useStdin, key)
	if err != nil {
		return err
	}

	if len(value) > MaxValueSizeWarnBytes {
		utils.PrintWarningStderr(fmt.Sprintf(
			"value for %q is %.1f KB — consider {{getFile \"path\"}} for large blobs",
			key, float64(len(value))/1024,
		))
	}

	// acquire lock
	return vault.WithStoreLock(func() error {
		identity, err := vault.ResolveIdentity()
		if err != nil {
			return fmt.Errorf("failed to load identity: %w", err)
		}

		store, err := vault.ReadStore(identity)
		if err != nil {
			return err
		}

		store.SetKey(envName, key, value)

		if err := vault.WriteStoreToRecipients(store); err != nil {
			return err
		}

		utils.PrintSuccessStderr(fmt.Sprintf("Set %s in %s", key, envName))
		return nil
	})
}

// resolveSetValue returns the value to store, picking from --stdin, a positional
// argument, or an interactive prompt. Trailing newlines are stripped from stdin
// reads so 'echo "x" | hulak secrets set FOO --stdin' stores "x" not 'x\n'.
//
// Multi-word values must be quoted (e.g. `hulak secrets set MOTD "hello world"`).
// More than one VALUE positional is rejected so a typo like
// `hulak secrets set FOO bar baz` doesn't silently store "bar baz".
func resolveSetValue(args []string, useStdin bool, key string) (string, error) {
	switch {
	case useStdin:
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read stdin: %w", err)
		}
		return strings.TrimRight(string(data), "\r\n"), nil

	case len(args) > 2:
		return "", fmt.Errorf(
			"too many arguments: got %d, expected KEY [VALUE]; quote multi-word values: hulak secrets set %s \"...\"",
			len(args),
			key,
		)

	case len(args) == 2:
		return args[1], nil

	default:
		return utils.PromptSecret(fmt.Sprintf("Enter value for %s: ", key))
	}
}

// newEnvGetCmd returns the command struct for `hulak secrets get`.
func newEnvGetCmd() *command {
	getFs := flag.NewFlagSet("env get", flag.ContinueOnError)
	getEnv := registerEnvFlag(getFs, utils.DefaultEnvVal, "Environment to operate on")

	return &command{
		Name:    "get",
		Aliases: []string{"g", "show", "view"},
		Short:   "Get a value by key",
		Long:    "Retrieve a secret from the encrypted vault and print it to stdout.\n\nOutput is raw — no formatting — suitable for $(...) substitution in scripts.\nExits non-zero if the key is missing.",
		Flags:   getFs,
		Args: []argDef{
			{Name: "key", Required: true, Desc: "Secret key to retrieve"},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets get API_KEY",
				Description: "Print API_KEY from the default environment",
			},
			{
				Command:     "hulak secrets get DB_URL --env prod",
				Description: "Print DB_URL from the prod environment",
			},
			{
				Command:     "API_KEY=$(hulak secrets get API_KEY --env staging)",
				Description: "Capture a value into a shell variable",
			},
		},
		Run: func(args []string) error { return runEnvGet(args, *getEnv) },
	}
}

// runEnvGet handles `hulak secrets get KEY`. Prints the raw value to stdout
// (suitable for $(...) capture) and returns a non-zero error if the key
// or environment is missing.
func runEnvGet(args []string, envName string) error {
	if len(args) == 0 {
		return errors.New("missing required argument: KEY")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments: got %d, expected KEY", len(args))
	}
	key := args[0]

	if err := requireVaultProject(); err != nil {
		return err
	}
	if err := utils.ValidateEnvName(envName); err != nil {
		return err
	}

	identity, err := vault.ResolveIdentity()
	if err != nil {
		return fmt.Errorf("failed to load identity: %w", err)
	}

	store, err := vault.ReadStore(identity)
	if err != nil {
		return err
	}

	env := store.GetEnv(envName)
	if env == nil {
		return fmt.Errorf("environment %q not found in vault store", envName)
	}

	value, ok := env[key]
	if !ok {
		return fmt.Errorf("key %q not found in environment %q", key, envName)
	}

	return printValue(value)
}

// printValue writes a stored value to stdout in a script-friendly form.
// Strings print raw; other types are JSON-encoded so numbers/bools/objects
// round-trip predictably (e.g. json.Number("8000") prints as 8000, not "8000").
func printValue(value any) error {
	if s, ok := value.(string); ok {
		fmt.Println(s)
		return nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to format value: %w", err)
	}
	fmt.Println(string(b))
	return nil
}

// newEnvDeleteCmd returns the command struct for `hulak secrets delete`.
func newEnvDeleteCmd() *command {
	deleteFs := flag.NewFlagSet("env delete", flag.ContinueOnError)
	deleteEnv := registerEnvFlag(deleteFs, utils.DefaultEnvVal, "Environment to operate on")

	return &command{
		Name:    "delete",
		Aliases: []string{"rm", "remove", "del"},
		Short:   "Delete a key",
		Long:    "Remove a secret from the encrypted vault.\n\nExits non-zero if the key doesn't exist.",
		Flags:   deleteFs,
		Args: []argDef{
			{Name: "key", Required: true, Desc: "Secret key to delete"},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets delete OLD_KEY",
				Description: "Delete OLD_KEY from the default environment",
			},
			{
				Command:     "hulak secrets rm STALE_TOKEN --env staging",
				Description: "Delete from a specific environment (alias)",
			},
		},
		Run: func(args []string) error { return runEnvDelete(args, *deleteEnv) },
	}
}

// runEnvDelete handles 'hulak secrets delete KEY'.
// Removes the key from the given environment under the file lock.
// Exits non-zero if the key (or env) is missing
// — a missing key is reported, not silently treated as success.
//
// Unlike set, this uses LoadIdentity (not EnsureKeypair) so running delete in a
// fresh project errors with "no identity found" instead of generating spurious
// keys. There's nothing to delete if no store exists.
func runEnvDelete(args []string, envName string) error {
	if len(args) == 0 {
		return errors.New("missing required argument: KEY")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments: got %d, expected KEY", len(args))
	}
	key := args[0]

	if err := requireVaultProject(); err != nil {
		return err
	}
	if err := utils.ValidateEnvName(envName); err != nil {
		return err
	}

	return vault.WithStoreLock(func() error {
		identity, err := vault.ResolveIdentity()
		if err != nil {
			return fmt.Errorf("failed to load identity: %w", err)
		}

		store, err := vault.ReadStore(identity)
		if err != nil {
			return err
		}

		env := store.GetEnv(envName)
		if env == nil {
			return fmt.Errorf("environment %q not found in vault store", envName)
		}
		if _, ok := env[key]; !ok {
			return fmt.Errorf("key %q not found in environment %q", key, envName)
		}

		store.DeleteKey(envName, key)

		if err := vault.WriteStoreToRecipients(store); err != nil {
			return err
		}

		utils.PrintSuccessStderr(fmt.Sprintf("Deleted %s from %s", key, envName))
		return nil
	})
}
