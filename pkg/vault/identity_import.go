package vault

import (
	"fmt"
	"os"
	"strings"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

// Contains import/export of age key material. Read/write of the identity.txt
// file itself lives in keys.go; this file is the user-facing CLI surface for
// moving keys in and out of hulak.

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

// ParseImportKey extracts and parses an age private key from raw import
// material (file contents or stdin). The raw input may be multi-line with
// comments — only the first non-empty, non-comment line is parsed.
//
// Used by import-key handlers that want to validate a candidate key before
// committing it (e.g. decrypt-test against a local store.age). Returns the
// parsed identity for further use; the caller decides whether to persist.
func ParseImportKey(raw string) (*age.X25519Identity, error) {
	key := extractKeyLine(raw)
	if key == "" {
		return nil, fmt.Errorf("no age private key found in input")
	}
	id, err := age.ParseX25519Identity(key)
	if err != nil {
		return nil, fmt.Errorf("invalid age private key: %w", err)
	}
	return id, nil
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

	id, err := ParseImportKey(raw)
	if err != nil {
		return err
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
		[]byte(id.String()+"\n"),
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
