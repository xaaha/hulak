package vault

import (
	"bytes"
	"io"
	"testing"

	"filippo.io/age"
)

func TestEncryptDecryptStream(t *testing.T) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}

	plaintext := []byte("hello hulak stream encryption")
	var cipherBuf bytes.Buffer

	if err := Encrypt(bytes.NewReader(plaintext), &cipherBuf, id.Recipient()); err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	if cipherBuf.Len() == 0 {
		t.Fatal("Encrypt() produced empty output")
	}

	rdr, err := Decrypt(bytes.NewReader(cipherBuf.Bytes()), id)
	if err != nil {
		t.Fatalf("Decrypt() error: %v", err)
	}

	got, err := io.ReadAll(rdr)
	if err != nil {
		t.Fatalf("io.ReadAll() error: %v", err)
	}

	if !bytes.Equal(got, plaintext) {
		t.Errorf("Decrypt() = %q, want %q", got, plaintext)
	}
}

func TestDecryptWithWrongIdentity(t *testing.T) {
	id1, _ := age.GenerateX25519Identity()
	id2, _ := age.GenerateX25519Identity()

	plaintext := []byte("secret data")
	var cipherBuf bytes.Buffer

	if err := Encrypt(bytes.NewReader(plaintext), &cipherBuf, id1.Recipient()); err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	_, err := Decrypt(bytes.NewReader(cipherBuf.Bytes()), id2)
	if err == nil {
		t.Error("Decrypt() with wrong identity should return error")
	}
}

func TestEncryptNoRecipients(t *testing.T) {
	var buf bytes.Buffer
	err := Encrypt(bytes.NewReader([]byte("data")), &buf)
	if err == nil {
		t.Error("Encrypt() with no recipients should return error")
	}
}

func TestEncryptDecryptEmpty(t *testing.T) {
	id, _ := age.GenerateX25519Identity()

	var cipherBuf bytes.Buffer
	if err := Encrypt(bytes.NewReader([]byte{}), &cipherBuf, id.Recipient()); err != nil {
		t.Fatalf("Encrypt() error on empty input: %v", err)
	}

	rdr, err := Decrypt(bytes.NewReader(cipherBuf.Bytes()), id)
	if err != nil {
		t.Fatalf("Decrypt() error on empty input: %v", err)
	}

	got, err := io.ReadAll(rdr)
	if err != nil {
		t.Fatalf("io.ReadAll() error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("Decrypt() = %q, want empty", got)
	}
}

func TestEncryptDecryptMultiRecipient(t *testing.T) {
	id1, _ := age.GenerateX25519Identity()
	id2, _ := age.GenerateX25519Identity()

	plaintext := []byte("shared secret")
	var cipherBuf bytes.Buffer

	if err := Encrypt(bytes.NewReader(plaintext), &cipherBuf, id1.Recipient(), id2.Recipient()); err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	ciphertext := cipherBuf.Bytes()

	// Both identities should be able to decrypt
	for i, id := range []*age.X25519Identity{id1, id2} {
		rdr, err := Decrypt(bytes.NewReader(ciphertext), id)
		if err != nil {
			t.Fatalf("Decrypt() with identity %d error: %v", i, err)
		}

		got, err := io.ReadAll(rdr)
		if err != nil {
			t.Fatalf("io.ReadAll() with identity %d error: %v", i, err)
		}

		if !bytes.Equal(got, plaintext) {
			t.Errorf("Decrypt() with identity %d = %q, want %q", i, got, plaintext)
		}
	}
}
