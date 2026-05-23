// Centralizes the "no --env → interactive picker" fallback used by the
// secrets subcommands (set, get, delete, keys, edit). Keeping the package
// variable and helper here means tests have a single stub point and the
// fallback shape stays consistent across commands.
package userflags

import "github.com/xaaha/hulak/pkg/tui/envselect"

// envPicker is the function called to interactively pick an environment when
// the user omits --env on a secrets subcommand. Indirected as a package
// variable so tests can stub it out — calling the real selector would open
// a TUI on /dev/tty and hang non-interactive test runs. The (string, bool,
// error) shape comes from envselect.RunEnvSelector; see its doc for the
// cancelled-bool contract.
var envPicker = envselect.RunEnvSelector

// resolveEnv returns envName when non-empty, otherwise prompts the user via
// the envselect TUI. The cancelled bool propagates from the picker so callers
// can distinguish a deliberate Esc/Ctrl+C from an error and exit cleanly.
func resolveEnv(envName string) (resolved string, cancelled bool, err error) {
	if envName != "" {
		return envName, false, nil
	}
	return envPicker()
}
