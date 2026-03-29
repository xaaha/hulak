package vault

import "filippo.io/age"

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
