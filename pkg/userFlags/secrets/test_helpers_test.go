package secrets

import (
	"bytes"
	"crypto/ed25519"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/xaaha/hulak/pkg/utils"
)

// chdirTemp changes the working directory to dir and returns a cleanup func
// that restores the original cwd. Test-only.
func chdirTemp(t *testing.T, dir string) func() {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	return func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	}
}

// captureStdout redirects os.Stdout to a pipe, runs fn, and returns
// everything that was written to stdout as a string.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("could not read from pipe: %v", err)
	}
	return buf.String()
}

// vaultTestSetup chdirs into a fresh project root and points
// $XDG_CONFIG_HOME at a separate tmpdir so the user's real identity file is
// never touched. Returns the project dir for follow-up assertions.
func vaultTestSetup(t *testing.T) string {
	t.Helper()
	projectDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Cleanup(chdirTemp(t, projectDir))
	return projectDir
}

// writeTestSSHKey generates an unencrypted ed25519 SSH private key in dir,
// returns the file path and the public key in authorized_keys format.
func writeTestSSHKey(t *testing.T, dir string) (keyPath, pubKey string) {
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
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(pemBlock), utils.SecretPer); err != nil {
		t.Fatalf("write key: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("NewPublicKey: %v", err)
	}
	pubKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
	return keyPath, pubKey
}
