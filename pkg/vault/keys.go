package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

// Contains on-disk identity file CRUD: path resolution, load/save/delete of
// identity.txt, plus the .old backup machinery used by rotate-key for crash
// recovery. The multi-source probe (env vars, SSH fallbacks, precedence
// rules) lives in identity_resolve.go; import/export lives in identity_import.go.

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

// DeleteIdentity removes the identity file from the global config directory.
func DeleteIdentity() error {
	identityFilePath, err := IdentityPath()
	if err != nil {
		return err
	}
	return os.Remove(identityFilePath)
}

// IdentityOldPath returns the path to the backup identity file
// (~/.config/hulak/identity.txt.old). Used by rotate-key for crash recovery.
func IdentityOldPath() (string, error) {
	path, err := IdentityPath()
	if err != nil {
		return "", err
	}
	return path + ".old", nil
}

// BackupIdentity copies the current identity.txt to identity.txt.old (mode 0600).
// Overwrites any existing .old file (one generation only).
func BackupIdentity() error {
	src, err := IdentityPath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("no identity to back up: %w", err)
	}
	dst, err := IdentityOldPath()
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, utils.SecretPer)
}

// LoadIdentityOld reads and parses the backup identity file (identity.txt.old).
// Returns error if the file doesn't exist or can't be parsed.
func LoadIdentityOld() (*age.X25519Identity, error) {
	path, err := IdentityOldPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no backup identity found: %w", err)
	}
	identity, err := age.ParseX25519Identity(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse backup identity: %w", err)
	}
	return identity, nil
}
