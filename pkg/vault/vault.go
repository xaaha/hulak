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

func GenerateKeyPair() (AgeKey, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return AgeKey{EncKey: "", DecKey: ""}, err
	}

	publicKey := id.Recipient().String()
	privateKey := id.String()

	return AgeKey{EncKey: publicKey, DecKey: privateKey}, nil
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
