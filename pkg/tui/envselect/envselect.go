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
func envItems() []string {
	if vault.DetectStore() == vault.StoreAge {
		identity, err := vault.LoadIdentity()
		if err != nil {
			utils.PrintRed(fmt.Sprintf("vault: %v", err))
			return nil
		}
		store, err := vault.ReadStore(identity)
		if err != nil {
			utils.PrintRed(fmt.Sprintf("vault: failed to read store: %v", err))
			return nil
		}
		return store.ListEnvs()
	}

	var items []string
	if files, err := utils.GetEnvFiles(); err == nil {
		for _, file := range files {
			if name, ok := strings.CutSuffix(file, utils.DefaultEnvFileSuffix); ok {
				items = append(items, name)
			}
		}
	}
	return items
}

// RunEnvSelector runs the environment selector and returns the selected environment.
func RunEnvSelector() (string, error) {
	return tui.RunSelector(envItems(), "Select Environment: ", noEnvFilesError())
}
