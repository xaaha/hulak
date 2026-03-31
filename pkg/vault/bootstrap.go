package vault

import (
	"os"

	"github.com/xaaha/hulak/pkg/utils"
)

// EnsureKeypair generates and stores an age keypair if one does not already exist.
// It's idempotent. If both key.pub and identity.txt exist, reads them back,
// verifies they are a matching pair, and returns the parsed AgeKey.
// Both .hulak/ and the global config directory must already exist (created by hulak init).
func EnsureKeypair() (AgeKey, error) {
	identityPath, err := getIdentityFilePath()
	if err != nil {
		return AgeKey{}, err
	}

	pubKeyPath, err := getPublicKeyFilePath()
	if err != nil {
		return AgeKey{}, err
	}

	// If both files exist, read, verify pair, and return
	if utils.FileExists(pubKeyPath) && utils.FileExists(identityPath) {
		privKeyStr, err := GetIdentity()
		if err != nil {
			return AgeKey{}, err
		}

		pubKeyBytes, err := os.ReadFile(pubKeyPath)
		if err != nil {
			return AgeKey{}, err
		}

		return VerifyKeypair(privKeyStr, string(pubKeyBytes))
	}

	// Generate new keypair
	ageKey, err := GenerateKeyPair()
	if err != nil {
		return AgeKey{}, err
	}

	if err := SetPublicKey(ageKey.Recipient.String()); err != nil {
		return AgeKey{}, err
	}

	if err := SetIdentity(ageKey.Identity.String()); err != nil {
		os.Remove(pubKeyPath)
		return AgeKey{}, err
	}

	return ageKey, nil
}
