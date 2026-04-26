package userflags

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/term"

	"github.com/xaaha/hulak/pkg/tui/envselect"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// maskedValue is what `keys` prints in place of secret values when --show is off.
const maskedValue = "••••"

// envPicker is the function called to interactively pick an environment when
// the user omits --env on `hulak env edit`. Indirected as a package variable
// so tests can stub it out — calling the real selector would open a TUI on
// /dev/tty and wait for keypress, hanging non-interactive test runs.
var envPicker = envselect.RunEnvSelector

// stdoutHeaders returns headers when stdout is a TTY, nil when piped.
// Hiding headers under pipe redirection keeps scripts like
// `for env in $(hulak env list)` clean — the same convention as kubectl / mise.
func stdoutHeaders(headers []string) []string {
	if term.IsTerminal(int(os.Stdout.Fd())) { //nolint:gosec // G115 fd is small non-neg
		return headers
	}
	return nil
}

// ---- SET ----

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
	//nolint:gosec // G705 TTY prompt, no taint sink
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

// --- DELETE ---

// runEnvDelete handles 'hulak env delete KEY'.
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

	if err := utils.ValidateEnvName(envName); err != nil {
		return err
	}

	return vault.WithStoreLock(func() error {
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
		if _, ok := env[key]; !ok {
			return fmt.Errorf("key %q not found in environment %q", key, envName)
		}

		store.DeleteKey(envName, key)

		if err := vault.WriteStore(store, identity.Recipient()); err != nil {
			return err
		}

		utils.PrintGreen(fmt.Sprintf("%s Deleted %s from %s", utils.CheckMark, key, envName))
		return nil
	})
}

// --- LIST ---

// runEnvList handles `hulak env list`. Prints environment names — one per line,
// sorted, no decoration — so output is friendly to `for env in $(hulak env list)`.
func runEnvList(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}

	identity, err := vault.LoadIdentity()
	if err != nil {
		return fmt.Errorf("failed to load identity: %w", err)
	}

	store, err := vault.ReadStore(identity)
	if err != nil {
		return err
	}

	names := store.ListEnvs()
	rows := make([][]string, len(names))
	for i, name := range names {
		rows[i] = []string{name}
	}
	return utils.PrintTable(os.Stdout, stdoutHeaders([]string{"ENVIRONMENT"}), rows, 0)
}

// --- KEYS ---

// runEnvKeys handles `hulak env keys`. Lists keys within an environment.
//
// Values are masked (••••) unless --show is set. --search filters keys by
// glob pattern (when the pattern contains '*', '?', or '[') or by
// case-insensitive substring otherwise.
func runEnvKeys(args []string, envName, search string, show bool) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}
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

	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if search != "" {
		keys, err = filterKeys(keys, search)
		if err != nil {
			return err
		}
	}

	rows := make([][]string, 0, len(keys))
	for _, k := range keys {
		val := maskedValue
		if show {
			val = formatTableValue(env[k])
		}
		rows = append(rows, []string{k, val})
	}
	return utils.PrintTable(
		os.Stdout,
		stdoutHeaders([]string{"KEY", "VALUE"}),
		rows,
		utils.DefaultTableMaxCellWidth,
	)
}

// filterKeys returns the subset of keys matching pattern. Glob mode (filepath.Match)
// is used when pattern contains '*', '?', or '['; otherwise case-insensitive
// substring match.
func filterKeys(keys []string, pattern string) ([]string, error) {
	isGlob := strings.ContainsAny(pattern, "*?[")
	lowered := strings.ToLower(pattern)

	out := make([]string, 0, len(keys))
	for _, k := range keys {
		var match bool
		if isGlob {
			ok, err := filepath.Match(pattern, k)
			if err != nil {
				return nil, fmt.Errorf("invalid glob pattern %q: %w", pattern, err)
			}
			match = ok
		} else {
			match = strings.Contains(strings.ToLower(k), lowered)
		}
		if match {
			out = append(out, k)
		}
	}
	return out, nil
}

// formatTableValue renders a stored value for inline (one-line) display.
// Strings show raw with control characters escaped (\n, \r, \t, etc.) so a
// value containing a newline or carriage return can't shift later rows or
// blank out the line via \r. Other types JSON-encode.
func formatTableValue(v any) string {
	if s, ok := v.(string); ok {
		return escapeControlChars(s)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// escapeControlChars rewrites ASCII control characters to a visible escape
// form so a stored value can't manipulate the terminal when printed in a
// table row. Concretely:
//
//   - \n would push the rest of the row onto a new line, breaking column
//     alignment for everything that follows.
//   - \r would rewind the cursor and overwrite the start of the line — a
//     subsequent value could blank out the key column entirely.
//   - \t expands to the next tab stop, which varies by terminal.
//   - other control bytes (BEL 0x07, ESC 0x1B, etc.) can ring the bell or
//     start an escape sequence we never intended.
//
// All ASCII control chars are 0x00–0x1F plus 0x7F (DEL). Any byte >= 0x80
// belongs to a multi-byte UTF-8 sequence (continuation bytes are 0x80–0xBF,
// leading bytes are 0xC0+); none of those are control chars, so we can scan
// byte-by-byte without ever splitting a rune.
//
// \n, \r, \t get the familiar two-char escapes; the rest fall back to \xNN
// hex so the output always stays printable ASCII.
func escapeControlChars(s string) string {
	// Fast path: most values have no control bytes (URLs, tokens, IDs).
	// Scan once and bail out without allocating if there's nothing to escape.
	hasControl := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c == 0x7F {
			hasControl = true
			break
		}
	}
	if !hasControl {
		return s
	}

	// Slow path: at least one control byte. Pre-size the builder to the
	// input length — a lower bound, since each escape grows the output.
	var sb strings.Builder
	sb.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\n':
			sb.WriteString(`\n`)
		case c == '\r':
			sb.WriteString(`\r`)
		case c == '\t':
			sb.WriteString(`\t`)
		case c < 0x20 || c == 0x7F:
			// Catch-all for BEL, ESC, NUL, DEL, etc. Hex form keeps the
			// output unambiguous and printable on every terminal.
			fmt.Fprintf(&sb, `\x%02x`, c)
		default:
			// Printable ASCII (0x20–0x7E) and UTF-8 bytes (>= 0x80) pass
			// through untouched — emojis, accents, CJK all render normally.
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

// --- EDIT ---

// runEnvEdit handles `hulak env edit`. Decrypts the named environment to a
// temporary 0600 JSON file, opens it in $EDITOR (or vi), then validates and
// writes the result back. The saved JSON REPLACES the environment wholesale —
// keys removed in the editor are removed from the store. Other environments in
// the store are untouched. Editor non-zero exit or unchanged content → no write.
//
// When envName is empty, the user is prompted via the env picker TUI — the
// same flow as `hulak run`. Edit deliberately does NOT default to "global":
// editing is destructive enough that we want explicit selection. To create or
// edit a brand-new env, pass it explicitly: `hulak env edit --env staging`.
//
// The whole read/edit/validate/write cycle is wrapped in WithStoreLock so an
// edit cannot race with a parallel set/delete.
func runEnvEdit(args []string, envName string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}

	if envName == "" {
		picked, err := envPicker()
		if err != nil {
			return err
		}
		envName = picked
	}

	if err := utils.ValidateEnvName(envName); err != nil {
		return err
	}

	return vault.WithStoreLock(func() error {
		ageKey, err := vault.EnsureKeypair()
		if err != nil {
			return fmt.Errorf("failed to load keypair: %w", err)
		}
		store, err := vault.ReadStore(ageKey.Identity)
		if err != nil {
			return err
		}

		// Marshal the env (or {} if the env doesn't exist yet — edit creates it).
		env := store.GetEnv(envName)
		if env == nil {
			env = make(vault.Env)
		}
		original, err := json.MarshalIndent(env, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal env: %w", err)
		}

		// Temp file inside .hulak/ keeps plaintext on the same filesystem
		// (same security boundary as store.age). The name encodes the env so
		// users see "edit-prod.json" in their editor's title bar — much nicer
		// than a random suffix. Safe to use a deterministic name because:
		//   - we're inside WithStoreLock (no concurrent edit)
		//   - ValidateEnvName already restricts to [a-zA-Z0-9_-] (no path tricks)
		//   - O_TRUNC overwrites any leftover from a previous crashed run
		markerPath, err := utils.GetProjectMarker()
		if err != nil {
			return err
		}
		tmpPath := filepath.Join(markerPath, "edit-"+envName+".json")
		tmpFile, err := os.OpenFile(
			tmpPath,
			os.O_RDWR|os.O_CREATE|os.O_TRUNC,
			utils.SecretPer,
		)
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		// Always remove the plaintext temp — even on editor crash, invalid
		// JSON, or panic up the stack.
		defer os.Remove(tmpPath)

		if _, err := tmpFile.Write(original); err != nil {
			_ = tmpFile.Close()
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			return fmt.Errorf("failed to close temp file: %w", err)
		}

		if err := launchEditor(tmpPath); err != nil {
			return err
		}

		edited, err := os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to read edited file: %w", err)
		}

		if bytes.Equal(original, edited) {
			utils.PrintGreen(fmt.Sprintf("%s No changes to %s", utils.CheckMark, envName))
			return nil
		}

		var newEnv vault.Env
		dec := json.NewDecoder(bytes.NewReader(edited))
		dec.UseNumber()
		if err := dec.Decode(&newEnv); err != nil {
			return fmt.Errorf("invalid JSON in edited file (store unchanged): %w", err)
		}

		store.Envs[envName] = newEnv

		if err := vault.WriteStore(store, ageKey.Recipient); err != nil {
			return err
		}

		utils.PrintGreen(fmt.Sprintf("%s Updated %s", utils.CheckMark, envName))
		return nil
	})
}

// launchEditor runs $EDITOR (or vi if unset) with path appended as its last
// argument. Stdin/Stdout/Stderr are wired to the parent terminal so the user
// interacts directly with the editor.
//
// $EDITOR is whitespace-split into argv (handles "code -w", "nvim --clean")
// but NOT shell-parsed — quotes and shell metachars in $EDITOR are not
// interpreted. Users with exotic editor invocations should write a wrapper
// script and point $EDITOR at it.
func launchEditor(path string) error {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		editor = utils.Editor
	}
	parts := strings.Fields(editor)
	parts = append(parts, path)

	//nolint:gosec // G204 $EDITOR is user-controlled by design — that's the contract.
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}
	return nil
}
