package envselect

import (
	"fmt"
	"strings"

	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
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

// EnvItems returns available environment names without the .env suffix.
func EnvItems() []string {
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
