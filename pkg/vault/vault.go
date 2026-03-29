package vault

import (
	"bytes"
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

func Encrypt(plaintext []byte, recepients []age.Recipient) ([]byte, error) {
	var buf bytes.Buffer

	wc, err := age.Encrypt(&buf, recepients...)
	if err != nil {
		return nil, err
	}

	// feed plaintext to encryptor
	if _, err = wc.Write(plaintext); err != nil {
		return nil, err
	}

	err = wc.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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
