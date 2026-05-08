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

// ResolveIdentity returns the identity to use for decryption.
// Precedence: HULAK_MASTER_KEY → identity.txt → HULAK_SSH_IDENTITY → ~/.ssh/id_ed25519.
// Returns age.Identity (interface) — callers should not assume X25519.
func ResolveIdentity() (age.Identity, error) {
	// 1. HULAK_MASTER_KEY (highest precedence)
	if raw := strings.TrimSpace(os.Getenv(utils.MasterKey)); raw != "" {
		return parseMasterKey(raw)
	}

	// 2. identity.txt
	if IdentityExists() {
		return LoadIdentity()
	}

	// 3. HULAK_SSH_IDENTITY env var
	if sshPath := strings.TrimSpace(os.Getenv(utils.SSHIdentityEnvVar)); sshPath != "" {
		identity, err := LoadSSHIdentity(sshPath)
		if err != nil {
			return nil, fmt.Errorf("%s is set but key could not be loaded: %w", utils.SSHIdentityEnvVar, err)
		}
		utils.PrintInfoStderr(fmt.Sprintf("Using SSH identity %s", sshPath))
		return identity, nil
	}

	// 4. Default ~/.ssh/id_ed25519
	defaultPath := DefaultSSHIdentityPath()
	if defaultPath != "" && utils.FileExists(defaultPath) {
		identity, err := LoadSSHIdentity(defaultPath)
		if err == nil {
			utils.PrintInfoStderr(fmt.Sprintf("Using SSH identity %s (no identity.txt found)", defaultPath))
			return identity, nil
		}
		// Don't error on auto-fallback failure — fall through to helpful error
	}

	// 5. Nothing found
	return nil, noIdentityError()
}

// noIdentityError returns a helpful error when no identity could be resolved.
func noIdentityError() error {
	return utils.HelpfulError(
		"no identity found — cannot decrypt the vault",
		"Set up an identity using one of these methods",
		[]string{
			"Run 'hulak init' to generate an age keypair",
			"Import an existing key with 'hulak secrets import-key'",
			fmt.Sprintf("Set %s to an AGE-SECRET-KEY- value (for CI)", utils.MasterKey),
			fmt.Sprintf("Set %s to point at your SSH private key", utils.SSHIdentityEnvVar),
			"Place an ed25519 SSH key at ~/.ssh/id_ed25519 (auto-detected)",
		},
	)
}

// parseMasterKey parses the HULAK_MASTER_KEY value with friendly errors.
func parseMasterKey(raw string) (*age.X25519Identity, error) {
	identity, err := age.ParseX25519Identity(raw)
	if err != nil {
		if strings.HasPrefix(raw, AgePrefix) {
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

// WrapDecryptError checks if HULAK_MASTER_KEY is set and the error looks like
// an age "no identity matched" failure. If so, wraps it with actionable hints.
// Otherwise returns the original error unchanged.
func WrapDecryptError(err error) error {
	if os.Getenv(utils.MasterKey) == "" {
		return err
	}
	if !strings.Contains(err.Error(), "no identity matched") {
		return err
	}
	return fmt.Errorf(
		"failed to decrypt store: %s is set but does not match any recipient in this project.\n"+
			"Common causes:\n"+
			"  - This key is for a different project\n"+
			"  - You were removed as a recipient\n"+
			"  - Stale whitespace or quotes from copy-paste",
		utils.MasterKey,
	)
}

// ExportKey reads the identity file and returns the raw private key string.
// Returns a friendly error pointing to `hulak init` if no identity exists.
func ExportKey() (string, error) {
	raw, err := GetIdentity()
	if err != nil {
		return "", fmt.Errorf(
			"no identity file found — run 'hulak init' to create one: %w", err,
		)
	}
	return strings.TrimSpace(raw), nil
}

// ImportKey validates the raw key material, normalizes whitespace, and writes
// it to the identity file atomically (tmp + rename). Refuses to overwrite an
// existing identity unless force is true.
//
// The raw input may be multi-line (e.g. age-keygen output with comments).
// Only the first non-empty, non-comment line is used as the key.
//
// Refuses to run if HULAK_MASTER_KEY is set — importing while the env var
// shadows the on-disk identity makes the import dead-on-arrival.
func ImportKey(raw string, force bool) error {
	if os.Getenv(utils.MasterKey) != "" {
		return fmt.Errorf(
			"%s is set — unset it before importing an identity, "+
				"otherwise the env var shadows the on-disk file",
			utils.MasterKey,
		)
	}

	key := extractKeyLine(raw)
	if key == "" {
		return fmt.Errorf("no age private key found in input")
	}

	if _, err := age.ParseX25519Identity(key); err != nil {
		return fmt.Errorf("invalid age private key: %w", err)
	}

	identityPath, err := IdentityPath()
	if err != nil {
		return err
	}

	if !force && utils.FileExists(identityPath) {
		return fmt.Errorf(
			"identity already exists at %s — use --force to overwrite", identityPath,
		)
	}

	return utils.AtomicWriteFile(
		identityPath,
		[]byte(key+"\n"),
		utils.SecretPer,
		utils.SecretDirPer,
	)
}

// extractKeyLine returns the first non-empty, non-comment line from raw input.
// Handles age-keygen output where comments (# lines) precede the key.
func extractKeyLine(raw string) string {
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		return line
	}
	return ""
}
