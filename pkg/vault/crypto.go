package vault

import (
	"io"

	"filippo.io/age"
)

// Contains low-level age encryption and decryption stream helpers.

// Encrypt reads plaintext from r, encrypts it for the given recipients, and writes ciphertext to w.
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

// Decrypt reads ciphertext from r and returns a reader that yields plaintext when read.
func Decrypt(r io.Reader, identities ...age.Identity) (io.Reader, error) {
	rdr, err := age.Decrypt(r, identities...)
	if err != nil {
		return nil, err
	}
	return rdr, nil
}
