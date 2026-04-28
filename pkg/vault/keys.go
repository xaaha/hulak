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

// IdentityPath returns the absolute path to the user's age identity file
// under the platform config dir (~/.config/hulak/identity.txt on Linux,
// the macOS equivalent, etc.).
func IdentityPath() (string, error) {
	configDir, err := utils.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, utils.IdentityFile), nil
}

// IdentityExists reports whether the identity file is already present on disk.
// Cheaper than LoadIdentity when the caller only needs the boolean.
func IdentityExists() bool {
	path, err := IdentityPath()
	if err != nil {
		return false
	}
	return utils.FileExists(path)
}

// GetIdentity reads and returns the raw private key string from the identity file.
func GetIdentity() (string, error) {
	path, err := IdentityPath()
	if err != nil {
		return "", err
	}
	byt, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(byt), nil
}

// LoadIdentity reads and parses the existing identity file.
// Unlike EnsureKeypair, this never creates keys — it errors if the identity is missing.
func LoadIdentity() (*age.X25519Identity, error) {
	raw, err := GetIdentity()
	if err != nil {
		return nil, fmt.Errorf("no identity found: %w", err)
	}
	identity, err := age.ParseX25519Identity(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("failed to parse identity: %w", err)
	}
	return identity, nil
}

// SetIdentity writes the private key to the global config identity file.
// Creates the parent directory if it doesn't exist so first-use bootstrap
// works without a separate "init the config dir" step.
func SetIdentity(privateKey string) error {
	identityFilePath, err := IdentityPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(identityFilePath), utils.SecretDirPer); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
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
	identityFilePath, err := IdentityPath()
	if err != nil {
		return err
	}
	err = os.Remove(identityFilePath)
	if err != nil {
		return err
	}
	return nil
}

// ResolveIdentity returns the age identity to use for decryption.
// Precedence: HULAK_MASTER_KEY env var → ~/.config/hulak/identity.txt.
//
// Wraps parse errors with user-friendly messages:
//   - public key in env var → "this looks like a public key"
//   - garbage in env var → hint at AGE-SECRET-KEY- format
func ResolveIdentity() (*age.X25519Identity, error) {
	if raw := strings.TrimSpace(os.Getenv(utils.MasterKey)); raw != "" {
		return parseMasterKey(raw)
	}
	return LoadIdentity()
}

// parseMasterKey parses the HULAK_MASTER_KEY value with friendly errors.
func parseMasterKey(raw string) (*age.X25519Identity, error) {
	identity, err := age.ParseX25519Identity(raw)
	if err != nil {
		if strings.HasPrefix(raw, "age1") {
			return nil, fmt.Errorf(
				"%s contains what looks like a public key (age1...), not a private key. "+
					"Private keys start with AGE-SECRET-KEY-",
				utils.MasterKey,
			)
		}
		return nil, fmt.Errorf(
			"%s is set but could not be parsed as an age private key: %w\n"+
				"Expected format: AGE-SECRET-KEY-1... ",
			utils.MasterKey, err,
		)
	}
	return identity, nil
}
