// Shared Run handlers for key-level CRUD operations (set, get, delete).
//
// Both the top-level `secrets set/get/delete` factories (env_crud.go) and the
// nested `secrets keys set/get/delete` factories (env_keys_crud.go) dispatch
// here. Keeping the logic in one file means a bug fix lands once and benefits
// both command paths — the factories themselves are pure wiring.
package userflags

import (
	"encoding/json"
	"errors"
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
// Kept as a fixed-size array (not a map) so error messages can present them
// in a stable order, the default ("string") stays first, and adding a value
// fails the compiler if the size constant is not updated to match.
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

// runEnvSet handles set semantics for both `secrets set` and `secrets keys set`.
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

	envName, cancelled, err := resolveAndValidateEnv(envName)
	if err != nil {
		return err
	}
	if cancelled {
		return nil
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

// runEnvGet handles get semantics for both `secrets get` and `secrets keys get`.
// Prints the raw value to stdout (suitable for $(...) capture) and returns a
// non-zero error if the key or environment is missing.
func runEnvGet(args []string, envName string) error {
	if len(args) == 0 {
		return errors.New("missing required argument: KEY")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments: got %d, expected KEY", len(args))
	}
	key := args[0]

	envName, cancelled, err := resolveAndValidateEnv(envName)
	if err != nil {
		return err
	}
	if cancelled {
		return nil
	}

	store, err := vault.ReadStore()
	if err != nil {
		return err
	}

	env, err := requireEnvExists(store, envName)
	if err != nil {
		return err
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

// runEnvDelete handles delete semantics for both `secrets delete` and
// `secrets keys delete`. Removes the key from the given environment under the
// file lock. Exits non-zero if the key (or env) is missing — a missing key is
// reported, not silently treated as success.
func runEnvDelete(args []string, envName string) error {
	if len(args) == 0 {
		return errors.New("missing required argument: KEY")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments: got %d, expected KEY", len(args))
	}
	key := args[0]

	envName, cancelled, err := resolveAndValidateEnv(envName)
	if err != nil {
		return err
	}
	if cancelled {
		return nil
	}

	return vault.WithStoreLock(func() error {
		store, err := vault.ReadStore()
		if err != nil {
			return err
		}

		env, err := requireEnvExists(store, envName)
		if err != nil {
			return err
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
