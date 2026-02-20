package envselect

import (
	"strings"

	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

func formatNoEnvFilesError() error {
	errMsg := `no '.env' files found in "env/" directory

Possible solutions:
  - Create an env file: echo "KEY=value" > env/dev.env
  - Run "hulak init" to create the env directory structure`

	return utils.ColorError(errMsg)
}

func envItems() []string {
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

// EnvItems returns available environment names without the .env suffix.
func EnvItems() []string {
	return envItems()
}

// NoEnvFilesError returns a formatted error for missing env files.
func NoEnvFilesError() error {
	return formatNoEnvFilesError()
}

// RunEnvSelector runs the environment selector and returns the selected environment.
func RunEnvSelector() (string, error) {
	return tui.RunSelector(envItems(), "Select Environment: ", formatNoEnvFilesError())
}
