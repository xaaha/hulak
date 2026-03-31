package vault

import (
	"bytes"
	"io"
	"os"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

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

func EncryptText(plainText []byte, receipients ...age.Recipient) ([]byte, error) {
	var buf bytes.Buffer
	if err := Encrypt(bytes.NewReader(plainText), &buf, receipients...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

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
