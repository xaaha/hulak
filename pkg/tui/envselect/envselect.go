package envselect

import (
	"fmt"
	"strings"

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
		identity, err := vault.ResolveIdentity()
		if err != nil {
			return nil, fmt.Errorf("vault: %w", err)
		}
		store, err := vault.ReadStore(identity)
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
func RunEnvSelector() (string, error) {
	items, err := envItems()
	if err != nil {
		return "", err
	}
	return tui.RunSelector(items, "Select Environment: ", noEnvFilesError())
}
