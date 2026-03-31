package vault

import (
	"os"
	"path/filepath"

	"github.com/xaaha/hulak/pkg/utils"
)

// SetPublicKey writes the public encryption key to .hulak/key.pub in the project root.
func SetPublicKey(publicEncKey string) error {
	markerPath, err := utils.GetProjectMarker()
	if err != nil {
		return err
	}

	pubKeyPath := filepath.Join(markerPath, publicKeyFile)
	return os.WriteFile(pubKeyPath, []byte(publicEncKey), utils.FilePer)
}
