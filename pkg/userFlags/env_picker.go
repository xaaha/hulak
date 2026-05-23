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
// the envselect TUI.
//
// The bool return distinguishes "user hit Esc to cancel" (cancelled=true,
// no error) from "the picker errored" (cancelled=false, err set) and from
// the normal path (cancelled=false, value set). Without it, callers couldn't
// tell a deliberate cancel from an accidental empty value, and would surface
// a misleading "environment name cannot be empty" from ValidateEnvName.
// On cancel, callers should return nil — Esc is a user action, not a failure.
func resolveEnv(envName string) (resolved string, cancelled bool, err error) {
	if envName != "" {
		return envName, false, nil
	}
	picked, err := envPicker()
	if err != nil {
		return "", false, err
	}
	if picked == "" {
		return "", true, nil
	}
	return picked, false, nil
}
