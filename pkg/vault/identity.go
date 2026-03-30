package vault

import (
	"os"
	"path/filepath"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

const (
	identityFile  = "identity.txt"
	publicKeyFile = "key.pub"
)

type AgeKey struct {
	EncKey string
	DecKey string
}

func GenerateKeyPair() (AgeKey, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return AgeKey{EncKey: "", DecKey: ""}, err
	}

	publicKey := id.Recipient().String()
	privateKey := id.String()

	return AgeKey{EncKey: publicKey, DecKey: privateKey}, nil
}

func getIdentityFile() (string, error) {
	configDir, err := utils.UserConfigDir()
	if err != nil {
		return "", err
	}
	privKeyPath := filepath.Join(configDir, identityFile)
	return privKeyPath, nil
}

// StoreIdentity writes the private key to the global config identity file.
func StoreIdentity(privateKey string) error {
	identityFilePath, err := getIdentityFile()
	if err != nil {
		return err
	}

	return os.WriteFile(identityFilePath, []byte(privateKey+"\n"), utils.SecretPer)
}

// StorePublicKey writes the public key to .hulak/key.pub in the project root.
func StorePublicKey(publicKey string) error {
	markerPath, err := utils.GetProjectMarker()
	if err != nil {
		return err
	}

	pubKeyPath := filepath.Join(markerPath, publicKeyFile)
	return os.WriteFile(pubKeyPath, []byte(publicKey+"\n"), utils.FilePer)
}

func DeleteIdentity() error {
	identityFilePath, err := getIdentityFile()
	if err != nil {
		return err
	}
	err = os.Remove(identityFilePath)
	if err != nil {
		return err
	}
	return nil
}
