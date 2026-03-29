package vault

import (
	"io"

	"filippo.io/age"
)

type AgeKey struct {
	EncKey string
	DecKey string
}

type AgeRS struct {
	Receipient age.Recipient
}

func GenerateKeyPair() (AgeKey, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return AgeKey{EncKey: "", DecKey: ""}, err
	}

	publicKey := id.Recipient().String()
	privateKey := id.String()

	return AgeKey{EncKey: publicKey, DecKey: privateKey}, nil
}

func Encrypt(r io.Reader, w io.Writer, recipients ...age.Recipient) error {
	ew, err := age.Encrypt(w, recipients...)
	if err != nil {
		return err
	}

	if _, err := io.Copy(ew, r); err != nil {
		_ = ew.Close()
		return err
	}

	return ew.Close()
}

func Decrypt(r io.Reader, identities []age.Identity) ([]byte, error) {
	plaintextReader, err := age.Decrypt(r, identities...)
	if err != nil {
		return nil, err
	}

	plainText, err := io.ReadAll(plaintextReader)
	if err != nil {
		return nil, err
	}
	return plainText, nil
}
