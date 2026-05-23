// Centralizes the "no --env → interactive picker" fallback used by the
// secrets subcommands (set, get, delete, keys, edit). Keeping the package
// variable and helper here means tests have a single stub point and the
// fallback shape stays consistent across commands.
package userflags

import "github.com/xaaha/hulak/pkg/tui/envselect"

// envPicker is the function called to interactively pick an environment when
// the user omits --env on a secrets subcommand. Indirected as a package
// variable so tests can stub it out — calling the real selector would open
// a TUI on /dev/tty and hang non-interactive test runs.
var envPicker = envselect.RunEnvSelector

// resolveEnv returns envName when non-empty, otherwise prompts the user via
// the envselect TUI. A cancelled picker returns "" with no error; callers
// should pass that through to ValidateEnvName, which rejects empty names —
// that's the single chokepoint for "no env was chosen".
func resolveEnv(envName string) (string, error) {
	if envName != "" {
		return envName, nil
	}
	return envPicker()
}
