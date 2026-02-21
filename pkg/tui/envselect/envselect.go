package envselect

import (
	"strings"

	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

// NoEnvFilesError returns a formatted error for missing env files.
func NoEnvFilesError() error {
	errMsg := `no '.env' files found in "env/" directory

Possible solutions:
  - Create an env file: echo "KEY=value" > env/dev.env
  - Run "hulak init" to create the env directory structure`

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
