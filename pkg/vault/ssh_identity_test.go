package vault

import (
	"crypto/ed25519"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// writeTestSSHKey generates an unencrypted ed25519 SSH private key, writes it
// to dir/id_ed25519, and returns the file path plus the public key in
// authorized_keys format.
func writeTestSSHKey(t *testing.T, dir string) (keyPath string, pubKeyStr string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	pemBlock, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("MarshalPrivateKey: %v", err)
	}
	keyPath = filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(pemBlock), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("NewPublicKey: %v", err)
	}
	pubKeyStr = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
	return keyPath, pubKeyStr
}

func TestDeriveSSHPublicKey(t *testing.T) {
	t.Run("derives matching public key from ed25519 private key", func(t *testing.T) {
		dir := t.TempDir()
		keyPath, expectedPub := writeTestSSHKey(t, dir)

		got, err := DeriveSSHPublicKey(keyPath)
		if err != nil {
			t.Fatalf("DeriveSSHPublicKey() error: %v", err)
		}
		if got != expectedPub {
			t.Errorf("DeriveSSHPublicKey() = %q, want %q", got, expectedPub)
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		_, err := DeriveSSHPublicKey("/tmp/nonexistent_key_12345")
		if err == nil {
			t.Fatal("DeriveSSHPublicKey() should error on missing file")
		}
	})

	t.Run("errors on non-SSH content", func(t *testing.T) {
		dir := t.TempDir()
		junkPath := filepath.Join(dir, "junk")
		if err := os.WriteFile(junkPath, []byte("not a key"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := DeriveSSHPublicKey(junkPath)
		if err == nil {
			t.Fatal("DeriveSSHPublicKey() should error on non-SSH content")
		}
	})
}

func TestLoadSSHIdentity(t *testing.T) {
	t.Run("loads unencrypted ed25519 key", func(t *testing.T) {
		dir := t.TempDir()
		keyPath, _ := writeTestSSHKey(t, dir)

		identity, err := LoadSSHIdentity(keyPath)
		if err != nil {
			t.Fatalf("LoadSSHIdentity() error: %v", err)
		}
		if identity == nil {
			t.Fatal("LoadSSHIdentity() returned nil identity")
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		_, err := LoadSSHIdentity("/tmp/nonexistent_key_file_12345")
		if err == nil {
			t.Fatal("LoadSSHIdentity() should error on missing file")
		}
	})

	t.Run("errors on non-SSH content", func(t *testing.T) {
		dir := t.TempDir()
		badPath := filepath.Join(dir, "not_ssh")
		if err := os.WriteFile(badPath, []byte("this is plain text, not an SSH key"), 0o600); err != nil {
			t.Fatalf("write file: %v", err)
		}

		_, err := LoadSSHIdentity(badPath)
		if err == nil {
			t.Fatal("LoadSSHIdentity() should error on non-SSH content")
		}
		if !strings.Contains(err.Error(), "OpenSSH") {
			t.Errorf("error %q should mention OPENSSH format", err.Error())
		}
	})

	t.Run("errors on empty file", func(t *testing.T) {
		dir := t.TempDir()
		emptyPath := filepath.Join(dir, "empty_key")
		if err := os.WriteFile(emptyPath, []byte(""), 0o600); err != nil {
			t.Fatalf("write file: %v", err)
		}

		_, err := LoadSSHIdentity(emptyPath)
		if err == nil {
			t.Fatal("LoadSSHIdentity() should error on empty file")
		}
		if !strings.Contains(err.Error(), "OpenSSH") {
			t.Errorf("error %q should mention OPENSSH format", err.Error())
		}
	})
}

func TestDefaultSSHIdentityPath(t *testing.T) {
	got := DefaultSSHIdentityPath()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}

	want := filepath.Join(home, ".ssh", "id_ed25519")
	if got != want {
		t.Errorf("DefaultSSHIdentityPath() = %q, want %q", got, want)
	}
}
