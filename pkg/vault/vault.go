package vault

import (
	"bytes"
	"fmt"
	"io"

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
func EncryptText(plainText []byte, recipients ...age.Recipient) ([]byte, error) {
	var buf bytes.Buffer
	if err := Encrypt(bytes.NewReader(plainText), &buf, recipients...); err != nil {
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

// EnsureKeypair returns the age keypair for this project, lazily creating one
// on first use. Identity is the source of truth:
//
//   - identity exists → parse and derive recipient from it, return.
//   - identity missing, store.age exists → refuse. Generating a new identity
//     would make the existing ciphertext permanently undecryptable.
//   - identity missing, no store.age → generate a fresh keypair, write identity.
func EnsureKeypair() (AgeKey, error) {
	identityPath, err := IdentityPath()
	if err != nil {
		return AgeKey{}, err
	}

	if utils.FileExists(identityPath) {
		return LoadKeypair()
	}

	// Identity missing. Refuse if a store already exists — the new identity
	// wouldn't match the recipient store.age was encrypted to.
	if existingStore, err := StorePath(); err == nil && utils.FileExists(existingStore) {
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

	if err := SetIdentity(ageKey.Identity.String()); err != nil {
		return AgeKey{}, err
	}

	return ageKey, nil
}

// LoadKeypair reads the existing identity and derives the recipient from it.
// Unlike EnsureKeypair, does not generate keys if missing.
func LoadKeypair() (AgeKey, error) {
	identity, err := LoadIdentity()
	if err != nil {
		return AgeKey{}, err
	}
	return AgeKey{Identity: identity, Recipient: identity.Recipient()}, nil
}
