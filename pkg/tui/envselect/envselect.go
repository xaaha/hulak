package envselect

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// noEnvFilesError returns a formatted error for missing environments.
// The message adapts to the active backend (vault or classic env/).
func noEnvFilesError() error {
	if vault.DetectStore() == vault.StoreAge {
		return utils.HelpfulError(
			"no environments found in encrypted store",
			"Possible causes",
			[]string{
				`The store has no environments yet — add one with "hulak secret set"`,
				"Identity file is missing or unreadable (check ~/.config/hulak/identity.txt)",
				"Store decryption failed (wrong identity for this store)",
			},
		)
	}

	return utils.HelpfulError(
		fmt.Sprintf(
			`no '%s' files found in "%s/" directory`,
			utils.DefaultEnvFileSuffix,
			utils.EnvironmentFolder,
		),
		"Possible solutions",
		[]string{
			fmt.Sprintf(
				`Create an env file: echo "KEY=value" > %s/dev%s`,
				utils.EnvironmentFolder,
				utils.DefaultEnvFileSuffix,
			),
			fmt.Sprintf(
				`Run "hulak init" to create the %s directory structure`,
				utils.EnvironmentFolder,
			),
		},
	)
}

// RunEnvSelector runs the environment selector and reports whether the user
// cancelled (Esc/Ctrl+C). Surfaces vault-layer errors verbatim so the user
// sees the actual problem (e.g. "identity file is corrupt") instead of an
// empty selector.
//
// The cancelled bool exists so callers don't have to infer cancellation from
// a magic empty string — Esc and "the picker errored" become unambiguous.
// On cancel, returns ("", true, nil); on success, (env, false, nil); on
// error, ("", false, err).
//
// Terminal handling:
//
//   - stdin is a TTY → run the picker against stdin/stdout as usual.
//   - stdin is piped (e.g. `pbpaste | hulak secrets set TOKEN --stdin`) but
//     stdout is a TTY and /dev/tty is openable → run the picker against
//     /dev/tty so the pipe stays free to feed the secret value. Without this
//     fallback, legitimate piped-value workflows would be locked out of the
//     picker.
//   - stdin is piped and stdout is not a TTY (CI, cron, captured output) →
//     refuse with an actionable error. Falling through to bubbletea here
//     would hang in PTY-allocated CI jobs or emit a cryptic "could not open
//     a new TTY" in detached contexts.
//
// The empty-items path skips the TTY logic entirely so the more helpful
// "no envs configured" error still wins when the store is empty.
func RunEnvSelector() (env string, cancelled bool, err error) {
	items, err := envparser.ListEnvironments()
	if err != nil {
		return "", false, err
	}

	picked, err := pickEnv(items)
	if err != nil {
		return "", false, err
	}
	if picked == "" {
		return "", true, nil
	}
	return picked, false, nil
}

// pickEnv routes the picker to the correct input/output channel based on
// what is currently a terminal. See RunEnvSelector for the decision matrix.
func pickEnv(items []string) (string, error) {
	if len(items) == 0 {
		return tui.RunSelector(items, "Select Environment: ", noEnvFilesError())
	}

	if term.IsTerminal(int(os.Stdin.Fd())) { //nolint:gosec // G115 fd is small non-neg
		return tui.RunSelector(items, "Select Environment: ", noEnvFilesError())
	}

	if !term.IsTerminal(int(os.Stdout.Fd())) { //nolint:gosec // G115 fd is small non-neg
		return "", noInteractiveTerminalError()
	}

	tty, err := tui.OpenControllingTerminal()
	if err != nil {
		return "", noInteractiveTerminalError()
	}
	defer tty.Close()

	return tui.RunSelectorOnTTY(items, "Select Environment: ", noEnvFilesError(), tty)
}

func noInteractiveTerminalError() error {
	return errors.New(
		"no --env provided and stdin is not a terminal — pass --env <name> to skip the picker",
	)
}
