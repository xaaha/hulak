package vault

import (
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

// func StoreIdentity() error {
// 	// get the config locaiton from utils getUserConfigLocation
// 	configPath, err := utils.UserConfigDir()
// 	if err != nil {
// 		return err
// 	}
//
// 	// generate the identity with GenerateKeyPair
// 	ageKey, err := GenerateKeyPair()
// 	if err != nil {
// 		return err
// 	}
//
// 	// Get the hulak working dir os.gwd
// 	cwd, err := os.Getwd()
// 	if err != nil {
// 		return err
// 	}
//
// 	// Save the private key in the config location with proper filePerm
// 	// save the public key in the cwd/.hulak/key.pub
// }
