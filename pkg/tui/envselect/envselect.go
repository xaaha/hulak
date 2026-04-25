package envselect

import (
	"fmt"
	"strings"

	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// NoEnvFilesError returns a formatted error for missing env files.
func NoEnvFilesError() error {
	errMsg := fmt.Sprintf(
		`no '%s' files found in "%s/" directory

Possible solutions:
  - Create an env file: echo "KEY=value" > %s/dev%s
  - Run "hulak init" to create the %s directory structure`,
		utils.DefaultEnvFileSuffix,
		utils.EnvironmentFolder,
		utils.EnvironmentFolder,
		utils.DefaultEnvFileSuffix,
		utils.EnvironmentFolder,
	)

	return utils.ColorError(errMsg)
}

// EnvItems returns available environment names.
// Reads from encrypted store when available, otherwise from env/ directory.
func EnvItems() []string {
	if vault.DetectStore() == vault.StoreAge {
		identity, err := vault.LoadIdentity()
		if err != nil {
			return nil
		}
		store, err := vault.ReadStore(identity)
		if err != nil {
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
	return tui.RunSelector(EnvItems(), "Select Environment: ", NoEnvFilesError())
}
