package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

func TestVerifyKeypair(t *testing.T) {
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("failed to generate identity: %v", err)
	}

	privKey := id.String()
	pubKey := id.Recipient().String()

	t.Run("valid matching pair", func(t *testing.T) {
		got, err := VerifyKeypair(privKey, pubKey)
		if err != nil {
			t.Fatalf("VerifyKeypair() error: %v", err)
		}
		if got.Identity.String() != privKey {
			t.Errorf("Identity = %q, want %q", got.Identity.String(), privKey)
		}
		if got.Recipient.String() != pubKey {
			t.Errorf("Recipient = %q, want %q", got.Recipient.String(), pubKey)
		}
	})

	t.Run("valid pair with whitespace", func(t *testing.T) {
		_, err := VerifyKeypair(privKey+"\n", "  "+pubKey+"\n")
		if err != nil {
			t.Errorf("VerifyKeypair() should trim whitespace, got error: %v", err)
		}
	})

	t.Run("mismatched pair", func(t *testing.T) {
		id2, _ := age.GenerateX25519Identity()
		_, err := VerifyKeypair(privKey, id2.Recipient().String())
		if err == nil {
			t.Error("VerifyKeypair() with mismatched keys should return error")
		}
		if !strings.Contains(err.Error(), "mismatch") {
			t.Errorf("error = %q, want it to contain 'mismatch'", err.Error())
		}
	})

	t.Run("invalid private key", func(t *testing.T) {
		_, err := VerifyKeypair("not-a-key", pubKey)
		if err == nil {
			t.Error("VerifyKeypair() with invalid private key should return error")
		}
	})

	t.Run("invalid public key", func(t *testing.T) {
		_, err := VerifyKeypair(privKey, "not-a-key")
		if err == nil {
			t.Error("VerifyKeypair() with invalid public key should return error")
		}
	})

	t.Run("empty strings", func(t *testing.T) {
		_, err := VerifyKeypair("", "")
		if err == nil {
			t.Error("VerifyKeypair() with empty strings should return error")
		}
	})
}

// setupConfigDir creates a temp config directory and sets XDG_CONFIG_HOME
// so that UserConfigDir() returns a path inside it. Returns the hulak config dir path.
func setupConfigDir(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpDir, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}

	// UserConfigDir on unix checks XDG_CONFIG_HOME first
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Cleanup(func() {
		os.Setenv("XDG_CONFIG_HOME", oldXDG)
	})

	// UserConfigDir appends "hulak" to XDG_CONFIG_HOME
	// but our getIdentityFilePath calls UserConfigDir which returns tmpDir directly
	// (since XDG_CONFIG_HOME is set, it returns it as-is without appending hulak)
	// Let's check what UserConfigDir actually returns
	configDir, err := utils.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error: %v", err)
	}

	// Create the config dir if needed
	if err := os.MkdirAll(configDir, utils.DirPer); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	return configDir
}

func TestSetIdentityGetIdentityRoundTrip(t *testing.T) {
	configDir := setupConfigDir(t)

	id, _ := age.GenerateX25519Identity()
	privKey := id.String()

	if err := SetIdentity(privKey); err != nil {
		t.Fatalf("SetIdentity() error: %v", err)
	}

	// Verify file exists with correct permissions
	identityPath := filepath.Join(configDir, identityFile)
	info, err := os.Stat(identityPath)
	if err != nil {
		t.Fatalf("identity file not created: %v", err)
	}
	if info.Mode().Perm() != utils.SecretPer {
		t.Errorf("identity file permissions = %o, want %o", info.Mode().Perm(), utils.SecretPer)
	}

	got, err := GetIdentity()
	if err != nil {
		t.Fatalf("GetIdentity() error: %v", err)
	}

	if strings.TrimSpace(got) != privKey {
		t.Errorf("GetIdentity() = %q, want %q", strings.TrimSpace(got), privKey)
	}
}

func TestDeleteIdentity(t *testing.T) {
	setupConfigDir(t)

	id, _ := age.GenerateX25519Identity()
	if err := SetIdentity(id.String()); err != nil {
		t.Fatalf("SetIdentity() error: %v", err)
	}

	if err := DeleteIdentity(); err != nil {
		t.Fatalf("DeleteIdentity() error: %v", err)
	}

	_, err := GetIdentity()
	if err == nil {
		t.Error("GetIdentity() after DeleteIdentity() should return error")
	}
}

func TestDeleteIdentityNonexistent(t *testing.T) {
	setupConfigDir(t)

	err := DeleteIdentity()
	if err == nil {
		t.Error("DeleteIdentity() on nonexistent file should return error")
	}
}

func TestSetPublicKey(t *testing.T) {
	projectDir := setupHulakProject(t)

	id, _ := age.GenerateX25519Identity()
	pubKey := id.Recipient().String()

	if err := SetPublicKey(pubKey); err != nil {
		t.Fatalf("SetPublicKey() error: %v", err)
	}

	pubKeyPath := filepath.Join(projectDir, utils.HiddenProjectName, publicKeyFile)
	got, err := os.ReadFile(pubKeyPath)
	if err != nil {
		t.Fatalf("failed to read key.pub: %v", err)
	}

	if strings.TrimSpace(string(got)) != pubKey {
		t.Errorf("key.pub content = %q, want %q", strings.TrimSpace(string(got)), pubKey)
	}

	// Verify file permissions
	info, err := os.Stat(pubKeyPath)
	if err != nil {
		t.Fatalf("failed to stat key.pub: %v", err)
	}
	if info.Mode().Perm() != utils.FilePer {
		t.Errorf("key.pub permissions = %o, want %o", info.Mode().Perm(), utils.FilePer)
	}
}

func TestEnsureKeypairGeneratesNew(t *testing.T) {
	setupHulakProject(t)
	setupConfigDir(t)

	ageKey, err := EnsureKeypair()
	if err != nil {
		t.Fatalf("EnsureKeypair() error: %v", err)
	}

	if ageKey.Identity == nil {
		t.Error("EnsureKeypair() Identity is nil")
	}
	if ageKey.Recipient == nil {
		t.Error("EnsureKeypair() Recipient is nil")
	}

	// Verify the pair matches
	derived := ageKey.Identity.Recipient()
	if derived.String() != ageKey.Recipient.String() {
		t.Error("EnsureKeypair() generated mismatched pair")
	}

	// Verify files were written
	privKeyStr, err := GetIdentity()
	if err != nil {
		t.Fatalf("GetIdentity() after EnsureKeypair: %v", err)
	}
	if strings.TrimSpace(privKeyStr) != ageKey.Identity.String() {
		t.Error("stored identity doesn't match returned identity")
	}
}

func TestEnsureKeypairIdempotent(t *testing.T) {
	setupHulakProject(t)
	setupConfigDir(t)

	first, err := EnsureKeypair()
	if err != nil {
		t.Fatalf("EnsureKeypair() first call error: %v", err)
	}

	second, err := EnsureKeypair()
	if err != nil {
		t.Fatalf("EnsureKeypair() second call error: %v", err)
	}

	if first.Identity.String() != second.Identity.String() {
		t.Error("EnsureKeypair() is not idempotent: identities differ")
	}
	if first.Recipient.String() != second.Recipient.String() {
		t.Error("EnsureKeypair() is not idempotent: recipients differ")
	}
}

func TestEnsureKeypairDetectsMismatch(t *testing.T) {
	projectDir := setupHulakProject(t)
	setupConfigDir(t)

	// Generate and store a keypair
	_, err := EnsureKeypair()
	if err != nil {
		t.Fatalf("EnsureKeypair() error: %v", err)
	}

	// Replace key.pub with a different key
	id2, _ := age.GenerateX25519Identity()
	pubKeyPath := filepath.Join(projectDir, utils.HiddenProjectName, publicKeyFile)
	if err := os.WriteFile(pubKeyPath, []byte(id2.Recipient().String()+"\n"), utils.FilePer); err != nil {
		t.Fatalf("failed to overwrite key.pub: %v", err)
	}

	_, err = EnsureKeypair()
	if err == nil {
		t.Error("EnsureKeypair() with mismatched keys should return error")
	}
	if !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("error = %q, want it to contain 'mismatch'", err.Error())
	}
}
