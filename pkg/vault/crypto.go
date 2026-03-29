package vault

import (
	"io"

	"filippo.io/age"
)

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

func Decrypt(r io.Reader, identities ...age.Identity) (io.Reader, error) {
	rdr, err := age.Decrypt(r, identities...)
	if err != nil {
		return nil, err
	}
	return rdr, nil
}
