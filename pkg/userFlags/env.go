package userflags

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// MaxValueSizeWarnBytes is the soft per-value size threshold at `set` time.
// Above this, the user is warned and pointed at {{getFile "path"}} for blobs.
// Not a hard limit — the value is still written.
const MaxValueSizeWarnBytes = 64 << 10 // 64 KiB

// runEnvSet handles `hulak env set KEY [VALUE]`.
//
// Resolution order for the value:
//  1. --stdin flag → read all of stdin
//  2. positional VALUE → use as-is
//  3. interactive prompt with no echo (only if stdin is a TTY)
//
// The read-modify-write of store.age is wrapped in WithStoreLock so concurrent
// `hulak env set` invocations cannot lose each other's edits.
func runEnvSet(args []string, envName string, useStdin bool) error {
	if len(args) == 0 {
		return errors.New("missing required argument: KEY")
	}
	key := args[0]

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
		// load identity
		ageKey, err := vault.EnsureKeypair()
		if err != nil {
			return fmt.Errorf("failed to load keypair: %w", err)
		}

		// read
		store, err := vault.ReadStore(ageKey.Identity)
		if err != nil {
			return err
		}

		// modify in memory
		store.SetKey(envName, key, value)

		// write  (atomic .tmp+rename)
		if err := vault.WriteStore(store, ageKey.Recipient); err != nil {
			return err
		}

		utils.PrintGreen(fmt.Sprintf("%s Set %s in %s", utils.CheckMark, key, envName))
		return nil
	})
}

// resolveSetValue returns the value to store, picking from --stdin, a positional
// argument, or an interactive prompt. Trailing newlines are stripped from stdin
// reads so 'echo "x" | hulak env set FOO --stdin' stores "x" not 'x\n'.
//
// Multi-word values must be quoted (e.g. `hulak env set MOTD "hello world"`).
// More than one VALUE positional is rejected so a typo like
// `hulak env set FOO bar baz` doesn't silently store "bar baz".
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
			"too many arguments: got %d, expected KEY [VALUE]; quote multi-word values: hulak env set %s \"...\"",
			len(args),
			key,
		)

	case len(args) == 2:
		return args[1], nil

	default:
		return promptSecretValue(key)
	}
}

// promptSecretValue reads a value from the terminal with no echo, no shell
// history. The prompt and trailing newline go to stderr so a misuse like
// `FOO=$(hulak env set BAR)` never captures them into the variable.
func promptSecretValue(key string) (string, error) {
	stdinFd := int(os.Stdin.Fd()) //nolint:gosec // G115 fd is small non-neg
	// ensure intput is coming from terminal
	if !term.IsTerminal(stdinFd) {
		return "", errors.New(
			"no value provided and stdin is not a terminal — pass VALUE positionally or use --stdin",
		)
	}
	fmt.Fprintf(os.Stderr, "Enter value for %s: ", key)
	// read input without echo
	bytes, err := term.ReadPassword(stdinFd)
	// newline to stderr
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return string(bytes), nil
}

// --- GET ---

// runEnvGet handles `hulak env get KEY`. Prints the raw value to stdout
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

	if err := utils.ValidateEnvName(envName); err != nil {
		return err
	}

	identity, err := vault.LoadIdentity()
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
