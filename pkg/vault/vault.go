package vault

import (
	"filippo.io/age"
)

type Key struct {
	EncKey string
	DecKey string
}

func GenerateKeyPair() (Key, error) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return Key{EncKey: "", DecKey: ""}, err
	}

	publicKey := id.Recipient().String()
	privateKey := id.String()

	return Key{EncKey: publicKey, DecKey: privateKey}, nil
}
