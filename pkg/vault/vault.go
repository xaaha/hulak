package vault

import (
	"filippo.io/age"
)

func GenerateKeyPair() (string, string, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", err
	}

	publicKey := id.Recipient().String()
	privateKey := id.String()

	return publicKey, privateKey, nil
}
