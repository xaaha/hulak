package fileselect

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/tui"
	"github.com/xaaha/hulak/pkg/utils"
)

func formatNoFilesError() error {
	errMsg := `no '.yaml' or '.yml' files found in current directory

Possible solutions:
  - Create an API file: echo "method: GET" > api.yaml
  - Check that files are not inside the 'env/' directory`

	return utils.ColorError(errMsg)
}

func fileItems() []string {
	var items []string

	files, err := utils.ListFiles(".")
	if err != nil {
		return nil
	}

	cwd, _ := os.Getwd()
	for _, file := range files {
		relPath, err := filepath.Rel(cwd, file)
		if err != nil {
			continue
		}

		lower := strings.ToLower(relPath)
		if !strings.HasSuffix(lower, utils.YAML) && !strings.HasSuffix(lower, utils.YML) {
			continue
		}

		if strings.Contains(relPath, utils.ResponseBase) {
			continue
		}

		if strings.HasPrefix(relPath, utils.EnvironmentFolder+string(filepath.Separator)) {
			continue
		}

		items = append(items, relPath)
	}
	return items
}

// RunFileSelector runs the file selector and returns the selected file path.
func RunFileSelector() (string, error) {
	return tui.RunSelector(fileItems(), "Select File: ", formatNoFilesError())
}
