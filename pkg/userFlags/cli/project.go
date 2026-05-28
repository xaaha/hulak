package cli

import (
	"fmt"

	"github.com/xaaha/hulak/pkg/utils"
)

// RequireVaultProject returns an error if the current directory is not inside
// a hulak vault project. Checks that .hulak/ directory actually exists on disk
// (not just that FindProjectRoot found an env/ marker). The store.age file
// itself may not exist yet (fresh init, before first `set`).
func RequireVaultProject() error {
	markerPath, err := utils.GetProjectMarker()
	if err != nil {
		return fmt.Errorf(
			"no vault project found\n\n" +
				"Run 'hulak init' to create one, or change to a hulak project directory",
		)
	}
	if !utils.DirExists(markerPath) {
		return fmt.Errorf(
			"this is a classic (env/) project, not a vault project\n\n" +
				"Run 'hulak secrets migrate' to upgrade to the encrypted vault",
		)
	}
	return nil
}
