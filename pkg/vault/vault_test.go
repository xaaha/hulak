package vault

import (
	"bytes"
	"testing"

	"filippo.io/age"
)

func TestGenerateKeyPair(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error: %v", err)
	}

	if key.Identity == nil {
		t.Error("GenerateKeyPair() Identity is nil")
	}
	if key.Recipient == nil {
		t.Error("GenerateKeyPair() Recipient is nil")
	}

	// Identity and Recipient should be a matching pair
	derived := key.Identity.Recipient()
	if derived.String() != key.Recipient.String() {
		t.Errorf("keypair mismatch: derived %q != recipient %q", derived.String(), key.Recipient.String())
	}
}

func TestGenerateKeyPairUniqueness(t *testing.T) {
	key1, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() first call error: %v", err)
	}

	key2, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() second call error: %v", err)
	}

	if key1.Identity.String() == key2.Identity.String() {
		t.Error("two GenerateKeyPair() calls produced identical identities")
	}
	if key1.Recipient.String() == key2.Recipient.String() {
		t.Error("two GenerateKeyPair() calls produced identical recipients")
	}
}

func TestEncryptTextDecryptText(t *testing.T) {
	id, _ := age.GenerateX25519Identity()

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"simple string", []byte("hello hulak")},
		{"empty input", []byte{}},
		{"binary data", []byte{0x00, 0xFF, 0x01, 0xFE}},
		{"json payload", []byte(`{"key": "value", "num": 42}`)},
		{"multiline", []byte("line1\nline2\nline3\n")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := EncryptText(tt.plaintext, id.Recipient())
			if err != nil {
				t.Fatalf("EncryptText() error: %v", err)
			}

			got, err := DecryptText(ciphertext, id)
			if err != nil {
				t.Fatalf("DecryptText() error: %v", err)
			}

			if !bytes.Equal(got, tt.plaintext) {
				t.Errorf("round-trip failed: got %q, want %q", got, tt.plaintext)
			}
		})
	}
}

func TestEncryptTextNoRecipients(t *testing.T) {
	_, err := EncryptText([]byte("data"))
	if err == nil {
		t.Error("EncryptText() with no recipients should return error")
	}
}

func TestDecryptTextWithWrongIdentity(t *testing.T) {
	id1, _ := age.GenerateX25519Identity()
	id2, _ := age.GenerateX25519Identity()

	ciphertext, err := EncryptText([]byte("secret"), id1.Recipient())
	if err != nil {
		t.Fatalf("EncryptText() error: %v", err)
	}

	_, err = DecryptText(ciphertext, id2)
	if err == nil {
		t.Error("DecryptText() with wrong identity should return error")
	}
}

func TestDecryptTextInvalidCiphertext(t *testing.T) {
	id, _ := age.GenerateX25519Identity()

	_, err := DecryptText([]byte("not-valid-ciphertext"), id)
	if err == nil {
		t.Error("DecryptText() with invalid ciphertext should return error")
	}
}
