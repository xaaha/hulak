package vault

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

// Contains keypair generation, persistence, and age text encryption helpers.

// AgeKey holds a matched age X25519 keypair.
type AgeKey struct {
	Recipient *age.X25519Recipient
	Identity  *age.X25519Identity
}

// GenerateKeyPair generates a new X25519 age keypair.
func GenerateKeyPair() (AgeKey, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return AgeKey{}, err
	}

	return AgeKey{
		Recipient: id.Recipient(),
		Identity:  id,
	}, nil
}

// EncryptText encrypts plainText bytes for the given age recipients and returns the ciphertext.
func EncryptText(plainText []byte, receipients ...age.Recipient) ([]byte, error) {
	var buf bytes.Buffer
	if err := Encrypt(bytes.NewReader(plainText), &buf, receipients...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecryptText decrypts ciphertext bytes using the given age identities and returns the plaintext.
func DecryptText(cypherText []byte, identities ...age.Identity) ([]byte, error) {
	rdr, err := Decrypt(bytes.NewReader(cypherText), identities...)
	if err != nil {
		return nil, err
	}
	plaintext, err := io.ReadAll(rdr)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

// SetPublicKey writes the public encryption key to .hulak/key.pub in the project root.
func SetPublicKey(publicEncKey string) error {
	pubKeyPath, err := getPublicKeyFilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(pubKeyPath, []byte(publicEncKey+"\n"), utils.FilePer)
}

// EnsureKeypair returns the age keypair for this project, lazily creating one
// on first use. Identity is the source of truth; key.pub is derived. So:
//
//   - identity + pubkey both present  → verify they match, return.
//   - identity present, pubkey missing → derive pubkey from identity and write
//     it back. Recovers from accidental key.pub deletion without generating
//     a new identity (which would silently brick any existing store.age).
//   - identity missing, store.age exists → refuse. Generating a new identity
//     would make the existing ciphertext permanently undecryptable. Likely
//     cause: $XDG_CONFIG_HOME changed between runs, so the resolved path
//     no longer points at the original identity file.
//   - identity missing, no store.age   → generate a fresh keypair and write
//     both files. Any orphan pubkey from a previous identity is overwritten.
func EnsureKeypair() (AgeKey, error) {
	identityPath, err := getIdentityFilePath()
	if err != nil {
		return AgeKey{}, err
	}

	pubKeyPath, err := getPublicKeyFilePath()
	if err != nil {
		return AgeKey{}, err
	}

	identityExists := utils.FileExists(identityPath)
	pubKeyExists := utils.FileExists(pubKeyPath)

	if identityExists && pubKeyExists {
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

	if identityExists {
		// Pubkey is missing but identity is present — derive and re-write
		// the pubkey instead of generating a new keypair.
		privKeyStr, err := GetIdentity()
		if err != nil {
			return AgeKey{}, err
		}
		identity, err := age.ParseX25519Identity(strings.TrimSpace(privKeyStr))
		if err != nil {
			return AgeKey{}, fmt.Errorf("failed to parse identity: %w", err)
		}
		recipient := identity.Recipient()
		if err := SetPublicKey(recipient.String()); err != nil {
			return AgeKey{}, err
		}
		return AgeKey{Identity: identity, Recipient: recipient}, nil
	}

	// Identity missing. Refuse to generate a new keypair if a store already
	// exists — the new identity wouldn't match the recipient store.age was
	// encrypted to, and the ciphertext would be unrecoverable.
	if existingStore, err := storePath(); err == nil && utils.FileExists(existingStore) {
		return AgeKey{}, utils.HelpfulError(
			fmt.Sprintf(
				"identity not found at %s but %s exists. Refusing to generate a new identity that would not match the existing store",
				identityPath, existingStore,
			),
			"Likely fixes",
			[]string{
				"Restore the original identity at " + identityPath,
				"Or set $XDG_CONFIG_HOME to point at the directory holding the original identity",
			},
		)
	}

	// No identity, no store — safe to generate a fresh keypair.
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
