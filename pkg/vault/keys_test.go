package vault

import (
	"errors"
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

	// Point XDG_CONFIG_HOME at our temp dir so utils.UserConfigDir resolves
	// to <tmpDir>/hulak — isolated from the user's real ~/.config/hulak.
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir, err := utils.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir() error: %v", err)
	}
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
	identityPath := filepath.Join(configDir, utils.IdentityFile)
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

func TestEnsureKeypairDerivesRecipient(t *testing.T) {
	setupHulakProject(t)
	setupConfigDir(t)

	first, err := EnsureKeypair()
	if err != nil {
		t.Fatalf("EnsureKeypair() initial: %v", err)
	}

	// Call again — should derive same recipient from identity on disk.
	second, err := EnsureKeypair()
	if err != nil {
		t.Fatalf("EnsureKeypair() second call: %v", err)
	}

	if second.Identity.String() != first.Identity.String() {
		t.Error("identity changed between calls")
	}
	if second.Recipient.String() != first.Recipient.String() {
		t.Errorf(
			"recipient mismatch: got %q, want %q",
			second.Recipient.String(),
			first.Recipient.String(),
		)
	}
}

// TestEnsureKeypairRefusesToOverwriteExistingStore verifies that if the
// identity is missing but a store.age exists, EnsureKeypair refuses to
// generate a fresh keypair. Generating one would make the existing
// ciphertext permanently undecryptable — the new pubkey wouldn't match
// the recipient store.age was originally encrypted to.
//
// Real-world trigger: $XDG_CONFIG_HOME resolves to a different path than
// when the store was created (changed dotfiles, different shell, direnv,
// SSH session with no XDG, etc.). The error guides the user to fix the
// path rather than silently destroying their data.
func TestEnsureKeypairRefusesToOverwriteExistingStore(t *testing.T) {
	setupHulakProject(t)
	configDir := setupConfigDir(t)

	// Create a real keypair and write a store encrypted to it.
	ageKey, err := EnsureKeypair()
	if err != nil {
		t.Fatalf("initial EnsureKeypair: %v", err)
	}
	store := &Store{Envs: map[string]Env{"global": {"FOO": "bar"}}}
	if err := WriteStore(store, ageKey.Recipient); err != nil {
		t.Fatalf("WriteStore: %v", err)
	}

	// Simulate the "XDG changed; identity not at the resolved path" scenario:
	// remove identity.txt while leaving store.age intact.
	if err := os.Remove(filepath.Join(configDir, utils.IdentityFile)); err != nil {
		t.Fatalf("remove identity: %v", err)
	}

	_, err = EnsureKeypair()
	if err == nil {
		t.Fatal("expected EnsureKeypair to refuse generation when store.age exists, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "store") || !strings.Contains(msg, "Refusing") {
		t.Errorf("error %q should mention the existing store and refusal", msg)
	}
	if !strings.Contains(msg, "XDG_CONFIG_HOME") {
		t.Errorf("error %q should hint at XDG_CONFIG_HOME as a likely cause", msg)
	}
}

func TestLoadIdentity(t *testing.T) {
	t.Run("loads existing identity", func(t *testing.T) {
		setupConfigDir(t)

		id, _ := age.GenerateX25519Identity()
		if err := SetIdentity(id.String()); err != nil {
			t.Fatalf("SetIdentity() error: %v", err)
		}

		got, err := LoadIdentity()
		if err != nil {
			t.Fatalf("LoadIdentity() error: %v", err)
		}
		if got.String() != id.String() {
			t.Errorf("LoadIdentity() = %q, want %q", got.String(), id.String())
		}
	})

	t.Run("errors when no identity exists", func(t *testing.T) {
		setupConfigDir(t)

		_, err := LoadIdentity()
		if err == nil {
			t.Error("LoadIdentity() should error when identity is missing")
		}
	})
}

func TestResolveIdentity(t *testing.T) {
	t.Run("prefers HULAK_MASTER_KEY over identity file", func(t *testing.T) {
		setupConfigDir(t)

		// Write a file-based identity
		fileID, _ := age.GenerateX25519Identity()
		if err := SetIdentity(fileID.String()); err != nil {
			t.Fatalf("SetIdentity: %v", err)
		}

		// Set env var to a different identity
		envID, _ := age.GenerateX25519Identity()
		t.Setenv("HULAK_MASTER_KEY", envID.String())

		got, err := ResolveIdentity()
		if err != nil {
			t.Fatalf("ResolveIdentity() error: %v", err)
		}
		if got.String() != envID.String() {
			t.Errorf("got %q, want env var identity %q", got.String(), envID.String())
		}
	})

	t.Run("falls back to identity file when env var unset", func(t *testing.T) {
		setupConfigDir(t)
		t.Setenv("HULAK_MASTER_KEY", "")

		fileID, _ := age.GenerateX25519Identity()
		if err := SetIdentity(fileID.String()); err != nil {
			t.Fatalf("SetIdentity: %v", err)
		}

		got, err := ResolveIdentity()
		if err != nil {
			t.Fatalf("ResolveIdentity() error: %v", err)
		}
		if got.String() != fileID.String() {
			t.Errorf("got %q, want file identity %q", got.String(), fileID.String())
		}
	})

	t.Run("empty env var is treated as unset", func(t *testing.T) {
		setupConfigDir(t)
		t.Setenv("HULAK_MASTER_KEY", "")

		fileID, _ := age.GenerateX25519Identity()
		if err := SetIdentity(fileID.String()); err != nil {
			t.Fatalf("SetIdentity: %v", err)
		}

		got, err := ResolveIdentity()
		if err != nil {
			t.Fatalf("ResolveIdentity() error: %v", err)
		}
		if got.String() != fileID.String() {
			t.Errorf("got %q, want file identity %q", got.String(), fileID.String())
		}
	})

	t.Run("env var with whitespace is trimmed", func(t *testing.T) {
		setupConfigDir(t)
		envID, _ := age.GenerateX25519Identity()
		t.Setenv("HULAK_MASTER_KEY", "  "+envID.String()+"\n")

		got, err := ResolveIdentity()
		if err != nil {
			t.Fatalf("ResolveIdentity() error: %v", err)
		}
		if got.String() != envID.String() {
			t.Errorf("got %q, want %q", got.String(), envID.String())
		}
	})

	t.Run("env var with public key gives helpful error", func(t *testing.T) {
		setupConfigDir(t)
		envID, _ := age.GenerateX25519Identity()
		t.Setenv("HULAK_MASTER_KEY", envID.Recipient().String())

		_, err := ResolveIdentity()
		if err == nil {
			t.Fatal("expected error for public key in HULAK_MASTER_KEY")
		}
		if !strings.Contains(err.Error(), "public key") {
			t.Errorf("error %q should mention 'public key'", err.Error())
		}
	})

	t.Run("env var with garbage gives helpful error", func(t *testing.T) {
		setupConfigDir(t)
		t.Setenv("HULAK_MASTER_KEY", "not-a-real-key")

		_, err := ResolveIdentity()
		if err == nil {
			t.Fatal("expected error for garbage HULAK_MASTER_KEY")
		}
		if !strings.Contains(err.Error(), "HULAK_MASTER_KEY") {
			t.Errorf("error %q should mention HULAK_MASTER_KEY", err.Error())
		}
		if !strings.Contains(err.Error(), "AGE-SECRET-KEY-") {
			t.Errorf("error %q should hint at correct format", err.Error())
		}
	})

	t.Run("no env var and no file gives original error", func(t *testing.T) {
		setupConfigDir(t)
		t.Setenv("HULAK_MASTER_KEY", "")

		_, err := ResolveIdentity()
		if err == nil {
			t.Fatal("expected error when no identity available")
		}
		if !strings.Contains(err.Error(), "no identity found") {
			t.Errorf("error %q should say 'no identity found'", err.Error())
		}
	})
}

func TestWrapDecryptError(t *testing.T) {
	t.Run("wraps no-match when HULAK_MASTER_KEY set", func(t *testing.T) {
		raw := errors.New("no identity matched any of the recipients")
		t.Setenv("HULAK_MASTER_KEY", "AGE-SECRET-KEY-fake")

		got := WrapDecryptError(raw)
		msg := got.Error()
		if !strings.Contains(msg, "HULAK_MASTER_KEY") {
			t.Errorf("error %q should mention HULAK_MASTER_KEY", msg)
		}
		if !strings.Contains(msg, "different project") {
			t.Errorf("error %q should suggest 'different project'", msg)
		}
	})

	t.Run("passes through when HULAK_MASTER_KEY unset", func(t *testing.T) {
		raw := errors.New("no identity matched any of the recipients")
		t.Setenv("HULAK_MASTER_KEY", "")

		got := WrapDecryptError(raw)
		if got.Error() != raw.Error() {
			t.Errorf("got %q, want original error %q", got.Error(), raw.Error())
		}
	})

	t.Run("passes through non-matching errors", func(t *testing.T) {
		raw := errors.New("some other error")
		t.Setenv("HULAK_MASTER_KEY", "something")

		got := WrapDecryptError(raw)
		if got.Error() != raw.Error() {
			t.Errorf("got %q, want original error %q", got.Error(), raw.Error())
		}
	})
}

func TestExportKey(t *testing.T) {
	t.Run("returns identity string when file exists", func(t *testing.T) {
		setupConfigDir(t)

		id, _ := age.GenerateX25519Identity()
		if err := SetIdentity(id.String()); err != nil {
			t.Fatalf("SetIdentity: %v", err)
		}

		got, err := ExportKey()
		if err != nil {
			t.Fatalf("ExportKey() error: %v", err)
		}
		if got != id.String() {
			t.Errorf("ExportKey() = %q, want %q", got, id.String())
		}
	})

	t.Run("errors when no identity exists", func(t *testing.T) {
		setupConfigDir(t)

		_, err := ExportKey()
		if err == nil {
			t.Fatal("ExportKey() should error when no identity")
		}
		if !strings.Contains(err.Error(), "hulak init") {
			t.Errorf("error %q should mention 'hulak init'", err.Error())
		}
	})
}

func TestImportKey(t *testing.T) {
	t.Run("imports valid key", func(t *testing.T) {
		configDir := setupConfigDir(t)
		t.Setenv("HULAK_MASTER_KEY", "")

		id, _ := age.GenerateX25519Identity()

		err := ImportKey(id.String(), false)
		if err != nil {
			t.Fatalf("ImportKey() error: %v", err)
		}

		got, err := GetIdentity()
		if err != nil {
			t.Fatalf("GetIdentity() after import: %v", err)
		}
		if strings.TrimSpace(got) != id.String() {
			t.Errorf("imported key = %q, want %q", strings.TrimSpace(got), id.String())
		}

		info, _ := os.Stat(filepath.Join(configDir, utils.IdentityFile))
		if info.Mode().Perm() != utils.SecretPer {
			t.Errorf("permissions = %o, want %o", info.Mode().Perm(), utils.SecretPer)
		}
	})

	t.Run("trims whitespace and extra newlines", func(t *testing.T) {
		setupConfigDir(t)
		t.Setenv("HULAK_MASTER_KEY", "")

		id, _ := age.GenerateX25519Identity()

		err := ImportKey("  "+id.String()+"\n\n", false)
		if err != nil {
			t.Fatalf("ImportKey() error: %v", err)
		}

		loaded, _ := LoadIdentity()
		if loaded.String() != id.String() {
			t.Errorf("got %q, want %q", loaded.String(), id.String())
		}
	})

	t.Run("rejects malformed key", func(t *testing.T) {
		setupConfigDir(t)
		t.Setenv("HULAK_MASTER_KEY", "")

		err := ImportKey("not-a-key", false)
		if err == nil {
			t.Fatal("expected error for malformed key")
		}
	})

	t.Run("refuses overwrite without force", func(t *testing.T) {
		setupConfigDir(t)
		t.Setenv("HULAK_MASTER_KEY", "")

		id1, _ := age.GenerateX25519Identity()
		if err := SetIdentity(id1.String()); err != nil {
			t.Fatalf("SetIdentity: %v", err)
		}

		id2, _ := age.GenerateX25519Identity()
		err := ImportKey(id2.String(), false)
		if err == nil {
			t.Fatal("expected error when overwriting without --force")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("error %q should mention 'already exists'", err.Error())
		}

		loaded, _ := LoadIdentity()
		if loaded.String() != id1.String() {
			t.Error("original identity was overwritten without --force")
		}
	})

	t.Run("overwrites with force", func(t *testing.T) {
		setupConfigDir(t)
		t.Setenv("HULAK_MASTER_KEY", "")

		id1, _ := age.GenerateX25519Identity()
		if err := SetIdentity(id1.String()); err != nil {
			t.Fatalf("SetIdentity: %v", err)
		}

		id2, _ := age.GenerateX25519Identity()
		err := ImportKey(id2.String(), true)
		if err != nil {
			t.Fatalf("ImportKey with force error: %v", err)
		}

		loaded, _ := LoadIdentity()
		if loaded.String() != id2.String() {
			t.Errorf("got %q, want %q", loaded.String(), id2.String())
		}
	})

	t.Run("refuses when HULAK_MASTER_KEY is set", func(t *testing.T) {
		setupConfigDir(t)
		envID, _ := age.GenerateX25519Identity()
		t.Setenv("HULAK_MASTER_KEY", envID.String())

		id, _ := age.GenerateX25519Identity()
		err := ImportKey(id.String(), false)
		if err == nil {
			t.Fatal("expected error when HULAK_MASTER_KEY is set")
		}
		if !strings.Contains(err.Error(), "HULAK_MASTER_KEY") {
			t.Errorf("error %q should mention HULAK_MASTER_KEY", err.Error())
		}
	})

	t.Run("extracts key from multi-line input with comments", func(t *testing.T) {
		setupConfigDir(t)
		t.Setenv("HULAK_MASTER_KEY", "")

		id, _ := age.GenerateX25519Identity()
		input := "# created: 2026-04-28\n# public key: " + id.Recipient().String() + "\n" + id.String() + "\n"

		err := ImportKey(input, false)
		if err != nil {
			t.Fatalf("ImportKey() error: %v", err)
		}
		loaded, _ := LoadIdentity()
		if loaded.String() != id.String() {
			t.Errorf("got %q, want %q", loaded.String(), id.String())
		}
	})
}

func TestBackupIdentity(t *testing.T) {
	setupConfigDir(t)

	id, _ := age.GenerateX25519Identity()
	if err := SetIdentity(id.String()); err != nil {
		t.Fatal(err)
	}

	t.Run("backs up identity to .old", func(t *testing.T) {
		if err := BackupIdentity(); err != nil {
			t.Fatalf("BackupIdentity() error: %v", err)
		}

		oldPath, _ := IdentityOldPath()
		data, err := os.ReadFile(oldPath)
		if err != nil {
			t.Fatalf("identity.txt.old not created: %v", err)
		}
		if strings.TrimSpace(string(data)) != id.String() {
			t.Errorf("backup content = %q, want %q", strings.TrimSpace(string(data)), id.String())
		}

		info, err := os.Stat(oldPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != utils.SecretPer {
			t.Errorf("backup permissions = %o, want %o", info.Mode().Perm(), utils.SecretPer)
		}
	})

	t.Run("overwrites existing .old on second backup", func(t *testing.T) {
		id2, _ := age.GenerateX25519Identity()
		if err := SetIdentity(id2.String()); err != nil {
			t.Fatal(err)
		}
		if err := BackupIdentity(); err != nil {
			t.Fatal(err)
		}

		oldPath, _ := IdentityOldPath()
		data, _ := os.ReadFile(oldPath)
		if strings.TrimSpace(string(data)) != id2.String() {
			t.Errorf("second backup should overwrite first")
		}
	})

	t.Run("errors when no identity exists", func(t *testing.T) {
		setupConfigDir(t) // fresh config dir, no identity
		err := BackupIdentity()
		if err == nil {
			t.Error("BackupIdentity() should error when no identity exists")
		}
	})
}

func TestLoadIdentityOld(t *testing.T) {
	setupConfigDir(t)

	id, _ := age.GenerateX25519Identity()
	if err := SetIdentity(id.String()); err != nil {
		t.Fatal(err)
	}
	if err := BackupIdentity(); err != nil {
		t.Fatal(err)
	}

	t.Run("loads backed up identity", func(t *testing.T) {
		got, err := LoadIdentityOld()
		if err != nil {
			t.Fatalf("LoadIdentityOld() error: %v", err)
		}
		if got.String() != id.String() {
			t.Errorf("LoadIdentityOld() = %q, want %q", got.String(), id.String())
		}
	})

	t.Run("errors when no .old file", func(t *testing.T) {
		setupConfigDir(t) // fresh dir, no .old
		_, err := LoadIdentityOld()
		if err == nil {
			t.Error("LoadIdentityOld() should error when no .old file")
		}
	})
}
