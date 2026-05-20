// Contains command factories and handlers for hulak secrets set, get, and delete.
package userflags

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// MaxValueSizeWarnBytes is the soft per-value size threshold at `set` time.
// Above this, the user is warned and pointed at {{getFile "path"}} for blobs.
// Not a hard limit — the value is still written.
const MaxValueSizeWarnBytes = 64 << 10 // 64 KiB

// validSetTypes lists the type names accepted by `secrets set --type`.
// Kept as a slice (not a map) so error messages can present them in a stable
// order and the default ("string") is first. Items available in json type
var validSetTypes = [5]string{"string", "int", "float", "bool", "json"}

// parseTypedValue converts the raw string value read from the CLI into the
// typed any that gets stored in the vault. Empty typeName defaults to
// "string" so callers that don't pass --type keep current behavior.
//
// int and float are returned as json.Number to match the shape the vault
// decoder emits on read (UseNumber). That keeps write-then-read a no-op and
// lets downstream JSON marshalling emit numbers as raw numbers rather than
// quoted strings.
func parseTypedValue(raw, typeName string) (any, error) {
	if typeName == "" {
		typeName = "string"
	}
	switch typeName {
	case "string":
		return raw, nil
	case "int":
		if _, err := strconv.ParseInt(raw, 10, 64); err != nil {
			return nil, fmt.Errorf("invalid int value %q: %w", raw, err)
		}
		return json.Number(raw), nil
	case "float":
		if _, err := strconv.ParseFloat(raw, 64); err != nil {
			return nil, fmt.Errorf("invalid float value %q: %w", raw, err)
		}
		return json.Number(raw), nil
	case "bool":
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid bool value %q: %w", raw, err)
		}
		return b, nil
	case "json":
		dec := json.NewDecoder(strings.NewReader(raw))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, fmt.Errorf("invalid json value: %w", err)
		}
		// Reject trailing tokens like `{"a":1}garbage` so silent truncation
		// can't happen. Trailing whitespace is fine — Decode consumes it on
		// the next call only if non-whitespace remains.
		if err := dec.Decode(new(any)); err != io.EOF {
			return nil, fmt.Errorf("invalid json value: unexpected data after value")
		}
		return v, nil
	default:
		return nil, fmt.Errorf(
			"unknown type %q: must be one of %s",
			typeName,
			strings.Join(validSetTypes[:], ", "),
		)
	}
}

// newEnvSetCmd returns the command struct for `hulak secrets set`.
func newEnvSetCmd() *command {
	setFs := flag.NewFlagSet("env set", flag.ContinueOnError)
	setEnv := registerEnvFlag(setFs, utils.DefaultEnvVal, "Environment to operate on")
	setStdin := setFs.Bool("stdin", false, "Read value from stdin")
	var setType string
	typeUsage := "Value type: " + strings.Join(validSetTypes[:], "|")
	setFs.StringVar(&setType, "type", "string", typeUsage)
	setFs.StringVar(&setType, "t", "string", typeUsage)

	return &command{
		Name:    "set",
		Aliases: []string{"add"},
		Short:   "Set a key-value pair",
		Long:    "Store a secret in the encrypted vault.\n\nIf VALUE is omitted, you'll be prompted to enter it (no echo, no shell history).\nUse --stdin to pipe the value from standard input (useful for scripts).\nUse --type to store numbers, booleans, or JSON instead of strings.",
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
			{
				Command:     "hulak secrets set userAge 3939 --type int",
				Description: "Store as an integer (preserved through to GraphQL/JSON bodies)",
			},
			{
				Command:     "hulak secrets set ENABLED true --type bool",
				Description: "Store as a boolean",
			},
			{
				Command:     "hulak secrets set config '{\"a\":1}' --type json",
				Description: "Store an arbitrary JSON value (object, array, number, etc.)",
			},
		},
		Run: func(args []string) error { return runEnvSet(args, *setEnv, *setStdin, setType) },
	}
}

// runEnvSet handles `hulak secrets set KEY [VALUE]`.
//
// Resolution order for the value:
//  1. --stdin flag → read all of stdin
//  2. positional VALUE → use as-is
//  3. interactive prompt with no echo (only if stdin is a TTY)
//
// typeName is the --type flag value; "" or "string" stores the raw string.
// int/float/bool/json are parsed by parseTypedValue and the typed result is
// what lands in the vault. Parse failure aborts before the store lock so a
// bad type/value never opens or mutates the store.
//
// The read-modify-write of store.age is wrapped in WithStoreLock so concurrent
// `hulak secrets set` invocations cannot lose each other's edits.
func runEnvSet(args []string, envName string, useStdin bool, typeName string) error {
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

	rawValue, err := resolveSetValue(args, useStdin, key)
	if err != nil {
		return err
	}

	if len(rawValue) > MaxValueSizeWarnBytes {
		utils.PrintWarningStderr(fmt.Sprintf(
			"value for %q is %.1f KB — consider {{getFile \"path\"}} for large blobs",
			key, float64(len(rawValue))/1024,
		))
	}

	typedValue, err := parseTypedValue(rawValue, typeName)
	if err != nil {
		return err
	}

	// acquire lock
	return vault.WithStoreLock(func() error {
		store, err := vault.ReadStore()
		if err != nil {
			return err
		}

		store.SetKey(envName, key, typedValue)

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

	store, err := vault.ReadStore()
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
		store, err := vault.ReadStore()
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
