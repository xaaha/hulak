// Contains command factories and handlers for hulak secrets list and keys.
package userflags

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// maskedValue is what `keys` prints in place of secret values when --show is
// off. Aliased to utils.MaskedValue so the glyph stays in sync across every
// command that masks output.
const maskedValue = utils.MaskedValue

// newEnvListCmd returns the command struct for `hulak secrets list`.
func newEnvListCmd() *command {
	listFs := flag.NewFlagSet("env list", flag.ContinueOnError)

	return &command{
		Name:    "list",
		Aliases: []string{"ls", "l"},
		Short:   "List environment names",
		Long:    "Show all environment names defined in the encrypted vault.\n\nThis lists the environments themselves (e.g. global, staging, prod).\nUse `hulak secrets keys --env <name>` to list keys within an environment.",
		Flags:   listFs,
		Examples: []*utils.CommandHelp{
			{Command: "hulak secrets list", Description: "List all environment names"},
			{Command: "hulak secrets ls", Description: "Same as list (alias)"},
		},
		Run: runEnvList,
	}
}

// runEnvList handles `hulak secrets list`. Prints environment names — one per line,
// sorted, no decoration — so output is friendly to `for env in $(hulak secrets list)`.
func runEnvList(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}
	if err := requireVaultProject(); err != nil {
		return err
	}

	store, err := vault.ReadStore()
	if err != nil {
		return err
	}

	names := store.ListEnvs()
	rows := make([][]string, len(names))
	for i, name := range names {
		rows[i] = []string{name}
	}
	return utils.PrintTable(os.Stdout, utils.StdoutHeaders([]string{"ENVIRONMENT"}), rows, 0)
}

// newEnvKeysCmd returns the command struct for `hulak secrets keys`.
func newEnvKeysCmd() *command {
	keysFs := flag.NewFlagSet("env keys", flag.ContinueOnError)
	keysEnv := registerEnvFlag(keysFs, "", "Environment to operate on")
	keysShow := registerShowFlag(keysFs, "Reveal values instead of masking them")
	keysSearch := keysFs.String(
		"search",
		"",
		"Filter keys by case-insensitive substring or glob pattern",
	)

	return &command{
		Name:    "keys",
		Aliases: []string{"key"},
		Short:   "List keys in an environment",
		Long:    "Show secret keys within an environment.\n\nValues are masked by default (••••) so the output is safe to share in screen recordings\nand meetings. Use --show to reveal them.\nUse --search to filter by case-insensitive substring or glob pattern (e.g. \"API*\", \"DB_?\").",
		Flags:   keysFs,
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets keys --env prod",
				Description: "List keys in prod with values masked",
			},
			{Command: "hulak secrets keys --env prod --show", Description: "Reveal actual values"},
			{
				Command:     "hulak secrets keys --env prod --search \"API*\"",
				Description: "Filter keys by glob pattern",
			},
			{
				Command:     "hulak secrets keys --env staging --search api",
				Description: "Filter by case-insensitive substring",
			},
		},
		Run: func(args []string) error {
			return runEnvKeys(args, *keysEnv, *keysSearch, *keysShow)
		},
	}
}

// runEnvKeys handles `hulak secrets keys`. Lists keys within an environment.
//
// Values are masked (••••) unless --show is set. --search filters keys by
// glob pattern (when the pattern contains '*', '?', or '[') or by
// case-insensitive substring otherwise.
func runEnvKeys(args []string, envName, search string, show bool) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}
	if err := requireVaultProject(); err != nil {
		return err
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

	store, err := vault.ReadStore()
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
		utils.StdoutHeaders([]string{"KEY", "VALUE"}),
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
