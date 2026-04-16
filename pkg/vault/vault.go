package vault

import (
	"bytes"
	"io"
	"os"

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
