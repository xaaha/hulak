package vault

import (
	"os"
	"path/filepath"

	"github.com/xaaha/hulak/pkg/utils"
)

// SetPublicKey writes the public key to .hulak/key.pub in the project root.
func SetPublicKey(publicKey string) error {
	markerPath, err := utils.GetProjectMarker()
	if err != nil {
		return err
	}

	pubKeyPath := filepath.Join(markerPath, publicKeyFile)
	return os.WriteFile(pubKeyPath, []byte(publicKey+"\n"), utils.FilePer)
}
