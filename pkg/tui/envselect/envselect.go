package envselect

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

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

// envItems returns available environment names.
// Reads from encrypted store when available, otherwise from env/ directory.
//
// Returns a non-nil error only when the vault is broken (missing identity,
// decrypt failure, recipient drift). An empty-but-healthy vault and a missing
// env/ directory both return (nil, nil) so the caller falls through to the
// "no envs configured" prompt instead of an alarming error.
func envItems() ([]string, error) {
	if vault.DetectStore() == vault.StoreAge {
		store, err := vault.ReadStore()
		if err != nil {
			return nil, fmt.Errorf("vault: reading store: %w", err)
		}
		return store.ListEnvs(), nil
	}

	var items []string
	if files, err := utils.GetEnvFiles(); err == nil {
		for _, file := range files {
			if name, ok := strings.CutSuffix(file, utils.DefaultEnvFileSuffix); ok {
				items = append(items, name)
			}
		}
	}
	return items, nil
}

// RunEnvSelector runs the environment selector and returns the selected environment.
// Surfaces vault-layer errors verbatim so the user sees the actual problem
// (e.g. "identity file is corrupt") instead of an empty selector.
//
// When stdin is not a terminal (CI, cron, piped input), the selector refuses
// to launch and returns an actionable error. Without this guard, bubbletea
// would silently fall back to /dev/tty — which in CI with a PTY hangs forever
// waiting for keypress, and in detached contexts surfaces a cryptic
// "could not open a new TTY" error. The check is gated on having items to
// show so that the more helpful "no envs configured" error still wins when
// the store is empty.
func RunEnvSelector() (string, error) {
	items, err := envItems()
	if err != nil {
		return "", err
	}
	if len(items) > 0 && !term.IsTerminal(int(os.Stdin.Fd())) { //nolint:gosec // G115 fd is small non-neg
		return "", errors.New(
			"no --env provided and stdin is not a terminal — pass --env <name> to skip the picker",
		)
	}
	return tui.RunSelector(items, "Select Environment: ", noEnvFilesError())
}
