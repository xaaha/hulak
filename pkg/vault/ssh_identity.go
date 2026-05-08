package vault

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"github.com/xaaha/hulak/pkg/utils"
	"golang.org/x/crypto/ssh"
)

// Contains SSH private key loading for vault decryption.
// SSH ed25519 (and rsa) keys are converted into age.Identity values
// via the agessh bridge so the same Decrypt() path works for both
// native age keys and SSH keys.

// LoadSSHIdentity reads an SSH private key file at path and returns an
// age.Identity suitable for vault decryption.
//
// For unencrypted keys the identity is ready immediately.
// For passphrase-protected keys the passphrase is prompted interactively
// on stderr/stdin at first use (lazy — the callback fires inside age.Decrypt).
func LoadSSHIdentity(path string) (age.Identity, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key %s: %w", path, err)
	}

	// Try parsing as an unencrypted key first.
	identity, err := agessh.ParseIdentity(pemBytes)
	if err == nil {
		return identity, nil
	}

	// Check whether the key is passphrase-protected.
	if identity, encErr := tryPassphraseProtectedKey(pemBytes, path); encErr == nil {
		return identity, nil
	}

	// Neither unencrypted parse nor encrypted detection succeeded.
	return nil, fmt.Errorf(
		"failed to parse SSH key %s: %w\n"+
			"Expected an OpenSSH-formatted private key (ssh-keygen generates this by default)",
		path, err,
	)
}

// tryPassphraseProtectedKey detects if pemBytes is a passphrase-protected SSH key
// and returns a lazy-decrypting identity that prompts on first use.
func tryPassphraseProtectedKey(pemBytes []byte, path string) (age.Identity, error) {
	var passErr *ssh.PassphraseMissingError

	// attempt to parse, could work
	_, rawErr := ssh.ParseRawPrivateKey(pemBytes)
	if !errors.As(rawErr, &passErr) || passErr.PublicKey == nil {
		return nil, fmt.Errorf("not a passphrase-protected key")
	}

	return agessh.NewEncryptedSSHIdentity(
		passErr.PublicKey,
		pemBytes,
		func() ([]byte, error) { return readPassphrase(path) },
	)
}

// readPassphrase prompts for a passphrase with no echo using the centralized prompt.
func readPassphrase(keyPath string) ([]byte, error) {
	pass, err := utils.PromptSecret(fmt.Sprintf("Enter passphrase for %s: ", keyPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read passphrase: %w", err)
	}
	return []byte(pass), nil
}

// DefaultSSHIdentityPath returns the default SSH private key path
// (~/.ssh/id_ed25519). Returns empty string if the home directory
// cannot be determined. Uses filepath.Join for cross-platform separators.
func DefaultSSHIdentityPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, utils.SSHKeyDir, utils.SSHKeyFile)
}
