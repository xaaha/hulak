package vault

import (
	"os"
	"path/filepath"

	"github.com/xaaha/hulak/pkg/utils"
)

const (
	identityFile  = "identity.txt"
	publicKeyFile = "key.pub"
)

// getIdentityFile returns the 'identity.txt' from the global conifg location
func getIdentityFile() (string, error) {
	configDir, err := utils.UserConfigDir()
	if err != nil {
		return "", err
	}
	privKeyPath := filepath.Join(configDir, identityFile)
	return privKeyPath, nil
}

func GetIdentity() (string, error) {
	path, err := getIdentityFile()
	if err != nil {
		return "", err
	}
	byt, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(byt), nil
}

// SetIdentity writes the private key to the global config identity file.
func SetIdentity(privateKey string) error {
	identityFilePath, err := getIdentityFile()
	if err != nil {
		return err
	}

	return os.WriteFile(identityFilePath, []byte(privateKey+"\n"), utils.SecretPer)
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
