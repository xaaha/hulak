package vault

import (
	"fmt"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"github.com/xaaha/hulak/pkg/utils"
	"golang.org/x/crypto/ssh"
)

// Contains SSH-aware recipient key parsing: detects key type from a string
// and returns an age.Recipient suitable for encryption.

// KeyType classifies the format of a public key string.
// For SSH keys, the value is the protocol's key type field (e.g. "ssh-ed25519").
// ClassifyKeyType derives it from the key itself.
type KeyType string

// ClassifyKeyType extracts the key type from a public key string.
// For SSH keys (space-delimited), returns the first field (e.g. "ssh-ed25519").
// For age keys (age1... prefix), returns "age".
// Returns empty string for unrecognized formats.
func ClassifyKeyType(key string) KeyType {
	k := strings.TrimSpace(key)

	if strings.HasPrefix(k, AgePrefix) {
		return Age
	}

	// SSH authorized_keys format: "<type> <base64data> [comment]"
	if typ, _, ok := strings.Cut(k, " "); ok {
		return KeyType(typ)
	}

	return ""
}

// ParseRecipientKey parses a public key string and returns an age.Recipient.
//
// Supported formats:
//   - age X25519 keys (age1...)
//   - SSH ed25519 keys (ssh-ed25519 ...)
//   - SSH RSA keys (ssh-rsa ...) — only when allowRSA is true
//
// Unsupported SSH key types (ecdsa, dsa, etc.) are rejected.
// Key type detection is handled by the ssh library — no hardcoded type strings.
func ParseRecipientKey(key string, allowRSA bool) (age.Recipient, KeyType, error) {
	k := strings.TrimSpace(key)

	// Sanity check: public keys are short. A line longer than this is not a
	// valid key — reject early before handing to the ssh parser.
	const maxKeyLen = 8 << 10 // 8 KiB — generous for RSA 4096; ed25519 is ~80 bytes
	if len(k) > maxKeyLen {
		return nil, "", fmt.Errorf("key too large (%d bytes, max %d)", len(k), maxKeyLen)
	}

	// Try age X25519 first
	if strings.HasPrefix(k, AgePrefix) {
		r, err := age.ParseX25519Recipient(k)
		if err != nil {
			return nil, Age, fmt.Errorf("invalid age key %q: %w", truncateKey(k), err)
		}
		return r, Age, nil
	}

	// Try SSH: let the ssh library parse and tell us the type
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(k))
	if err != nil {
		return nil, "", fmt.Errorf("unrecognized recipient format: %s", truncateKey(k))
	}

	kt := KeyType(pubKey.Type())

	// RSA needs explicit opt-in
	if pubKey.Type() == ssh.KeyAlgoRSA && !allowRSA {
		return nil, kt, fmt.Errorf(
			"%s keys are not accepted by default (slow, large ciphertexts); "+
				"use --allow-rsa to override", kt,
		)
	}

	// Let agessh handle the actual recipient creation.
	// It supports ed25519 and rsa; rejects everything else (ecdsa, dsa, etc.)
	r, err := agessh.ParseRecipient(k)
	if err != nil {
		return nil, kt, fmt.Errorf(
			"%s keys are not supported by age encryption — use ed25519 or age keys", kt,
		)
	}

	return r, kt, nil
}

// truncateKey shortens a key string for display in error messages.
// Keys longer than 40 characters are trimmed with "..." appended.
func truncateKey(key string) string {
	const maxLen = 40
	if len(key) <= maxLen {
		return key
	}
	return key[:maxLen] + utils.Ellipsis
}
