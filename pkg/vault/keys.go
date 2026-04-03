package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

// Contains everythig about public and private keys (identity)

const (
	identityFile  = "identity.txt"
	publicKeyFile = "key.pub"
)

// getIdentityFilePath returns the 'identity.txt' from the global conifg location
func getIdentityFilePath() (string, error) {
	configDir, err := utils.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, identityFile), nil
}

// getPublicKeyFilePath returns the 'key.pub' path from the project marker (.hulak/)
func getPublicKeyFilePath() (string, error) {
	markerPath, err := utils.GetProjectMarker()
	if err != nil {
		return "", err
	}
	return filepath.Join(markerPath, publicKeyFile), nil
}

// GetIdentity reads and returns the raw private key string from the identity file.
func GetIdentity() (string, error) {
	path, err := getIdentityFilePath()
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
	identityFilePath, err := getIdentityFilePath()
	if err != nil {
		return err
	}

	return os.WriteFile(identityFilePath, []byte(privateKey+"\n"), utils.SecretPer)
}

// VerifyKeypair parses the raw private and public key strings, and verifies
// that the private key derives the same public key. Returns the parsed AgeKey.
func VerifyKeypair(rawPrivateKey, rawPublicKey string) (AgeKey, error) {
	identity, err := age.ParseX25519Identity(strings.TrimSpace(rawPrivateKey))
	if err != nil {
		return AgeKey{}, fmt.Errorf("failed to parse identity: %w", err)
	}

	recipient, err := age.ParseX25519Recipient(strings.TrimSpace(rawPublicKey))
	if err != nil {
		return AgeKey{}, fmt.Errorf("failed to parse public key: %w", err)
	}

	derived := identity.Recipient()
	if derived.String() != recipient.String() {
		return AgeKey{}, fmt.Errorf("keypair mismatch: identity does not match public key")
	}

	return AgeKey{
		Recipient: recipient,
		Identity:  identity,
	}, nil
}

// DeleteIdentity removes the identity file from the global config directory.
func DeleteIdentity() error {
	identityFilePath, err := getIdentityFilePath()
	if err != nil {
		return err
	}
	err = os.Remove(identityFilePath)
	if err != nil {
		return err
	}
	return nil
}
