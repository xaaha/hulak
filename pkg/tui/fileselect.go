package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// File selection helpers live in package tui because they provide data for
// TUI flows but are not standalone Bubble Tea models.

func NoFilesError() error {
	errMsg := `no '.yaml' or '.yml' files found in current directory

Possible solutions:
  - Create an API file: echo "method: GET" > api.yaml
  - Check that files are not inside the 'env/' directory`

	return utils.ColorError(errMsg)
}

func FileItems() ([]string, error) {
	var items []string

	files, err := utils.ListFiles(".")
	if err != nil {
		if errors.Is(err, utils.ErrNoFiles) {
			return nil, nil
		}
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	envPrefix := utils.EnvironmentFolder + string(filepath.Separator)
	items = make([]string, 0, len(files))

	for _, file := range files {
		relPath, err := filepath.Rel(cwd, file)
		if err != nil {
			continue
		}

		lower := strings.ToLower(filepath.Ext(relPath))
		if !strings.HasSuffix(lower, utils.YAML) && !strings.HasSuffix(lower, utils.YML) {
			continue
		}

		if strings.HasPrefix(relPath, envPrefix) {
			continue
		}

		items = append(items, relPath)
	}
	return items, nil
}

func RunFileSelector() (string, error) {
	items, err := FileItems()
	if err != nil {
		return "", err
	}
	return RunSelector(items, "Select File: ", NoFilesError())
}
