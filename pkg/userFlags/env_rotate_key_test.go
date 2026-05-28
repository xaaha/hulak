package userflags

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/userFlags/initcmd"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// setupRotateKeyTest creates a temp project with .hulak dir, identity, recipients,
// and an encrypted store. Returns the old identity for assertions.
func setupRotateKeyTest(t *testing.T, extraRecipients []vault.RecipientEntry) *age.X25519Identity {
	t.Helper()

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	hulakDir := filepath.Join(tmpDir, utils.HiddenProjectName)
	if err := os.Mkdir(hulakDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}

	configTmp := t.TempDir()
	configTmp, _ = filepath.EvalSymlinks(configTmp)
	t.Setenv("XDG_CONFIG_HOME", configTmp)

	configDir, err := utils.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	// Generate identity
	id, _ := age.GenerateX25519Identity()
	if err := vault.SetIdentity(id.String()); err != nil {
		t.Fatal(err)
	}

	// Build recipient entries
	entries := []vault.RecipientEntry{
		{Key: id.Recipient().String(), Name: "me (added 2026-01-01)"},
	}
	entries = append(entries, extraRecipients...)

	if err := vault.SaveRecipients(entries); err != nil {
		t.Fatal(err)
	}

	// Create and encrypt a store
	store := &vault.Store{Envs: map[string]vault.Env{
		"global": {"URL": "https://example.com"},
		"prod":   {"API_KEY": "sk-secret", "COUNT": 42},
	}}
	recipients, err := vault.RecipientsFromEntries(entries)
	if err != nil {
		t.Fatal(err)
	}
	if err := vault.WriteStore(store, recipients...); err != nil {
		t.Fatal(err)
	}

	return id
}

func TestRunRotateKey(t *testing.T) {
	t.Run("single recipient happy path", func(t *testing.T) {
		oldID := setupRotateKeyTest(t, nil)

		if err := runRotateKey(nil); err != nil {
			t.Fatalf("runRotateKey error: %v", err)
		}

		// New identity can decrypt store
		newIdentity, err := vault.ResolveIdentity()
		if err != nil {
			t.Fatal(err)
		}
		newX25519 := newIdentity.(*age.X25519Identity)
		if newX25519.String() == oldID.String() {
			t.Error("identity should have changed")
		}

		store, err := vault.DecryptStore(newIdentity)
		if err != nil {
			t.Fatalf("new identity can't decrypt store: %v", err)
		}
		if store.GetEnv("global")["URL"] != "https://example.com" {
			t.Error("store data not preserved")
		}
		if store.GetEnv("prod")["API_KEY"] != "sk-secret" {
			t.Error("store data not preserved")
		}

		// Old identity cannot decrypt store
		_, err = vault.DecryptStore(oldID)
		if err == nil {
			t.Error("old identity should NOT decrypt store after rotation")
		}

		// Backup exists
		oldBackup, err := vault.LoadIdentityOld()
		if err != nil {
			t.Fatal("identity.txt.old should exist")
		}
		if oldBackup.String() != oldID.String() {
			t.Error("backup should contain old identity")
		}

		// Recipients updated
		recipientPath, _ := vault.RecipientsFilePath()
		data, _ := os.ReadFile(recipientPath)
		content := string(data)
		if strings.Contains(content, oldID.Recipient().String()) {
			t.Error("old public key should not be in recipients.txt")
		}
		if !strings.Contains(content, newX25519.Recipient().String()) {
			t.Error("new public key should be in recipients.txt")
		}
	})

	t.Run("multi-recipient preserves teammates", func(t *testing.T) {
		teammate, _ := age.GenerateX25519Identity()
		oldID := setupRotateKeyTest(t, []vault.RecipientEntry{
			{Key: teammate.Recipient().String(), Name: "teammate"},
		})

		if err := runRotateKey(nil); err != nil {
			t.Fatalf("runRotateKey error: %v", err)
		}

		// Teammate can still decrypt
		store, err := vault.DecryptStore(teammate)
		if err != nil {
			t.Fatalf("teammate can't decrypt store after rotation: %v", err)
		}
		if store.GetEnv("prod")["API_KEY"] != "sk-secret" {
			t.Error("store data not preserved for teammate")
		}

		// Old identity cannot decrypt
		_, err = vault.DecryptStore(oldID)
		if err == nil {
			t.Error("old identity should NOT decrypt store")
		}

		// Teammate's entry in recipients.txt is byte-for-byte unchanged
		recipientPath, _ := vault.RecipientsFilePath()
		data, _ := os.ReadFile(recipientPath)
		entries, _ := vault.ParseRecipientsFileContent(data)
		found := false
		for _, e := range entries {
			if e.Key == teammate.Recipient().String() {
				found = true
				if e.Name != "teammate" {
					t.Errorf("teammate name changed to %q", e.Name)
				}
			}
		}
		if !found {
			t.Error("teammate key missing from recipients.txt after rotation")
		}
	})

	t.Run("refuses when HULAK_MASTER_KEY set", func(t *testing.T) {
		setupRotateKeyTest(t, nil)
		t.Setenv(utils.MasterKey, "AGE-SECRET-KEY-1FAKE")

		err := runRotateKey(nil)
		if err == nil {
			t.Fatal("should refuse when HULAK_MASTER_KEY set")
		}
		if !strings.Contains(err.Error(), "identity import") {
			t.Errorf("error should mention identity import, got: %v", err)
		}
	})

	t.Run("refuses with too many arguments", func(t *testing.T) {
		err := runRotateKey([]string{"extra"})
		if err == nil {
			t.Error("should refuse extra arguments")
		}
	})

	t.Run("preserves store data types after rotation", func(t *testing.T) {
		setupRotateKeyTest(t, nil)

		if err := runRotateKey(nil); err != nil {
			t.Fatalf("runRotateKey error: %v", err)
		}

		store, _ := vault.ReadStore()
		prod := store.GetEnv("prod")

		if prod["API_KEY"] != "sk-secret" {
			t.Errorf("API_KEY = %v, want sk-secret", prod["API_KEY"])
		}
	})

	t.Run("identity.txt.old overwritten on second rotation", func(t *testing.T) {
		firstID := setupRotateKeyTest(t, nil)

		if err := runRotateKey(nil); err != nil {
			t.Fatalf("first rotation error: %v", err)
		}

		// After first rotation, .old has firstID
		old1, _ := vault.LoadIdentityOld()
		if old1.String() != firstID.String() {
			t.Error("first .old should be original identity")
		}

		// Second rotation
		midID, _ := vault.ResolveIdentity()
		midX25519 := midID.(*age.X25519Identity)
		if err := runRotateKey(nil); err != nil {
			t.Fatalf("second rotation error: %v", err)
		}

		// Now .old should have the mid identity, not firstID
		old2, _ := vault.LoadIdentityOld()
		if old2.String() != midX25519.String() {
			t.Error("second .old should be mid identity, not first")
		}
	})
}

func TestRunRotateKeyRecovery(t *testing.T) {
	t.Run("recovers from interrupted rotation", func(t *testing.T) {
		oldID := setupRotateKeyTest(t, nil)

		// Simulate interrupted rotation:
		// 1. Backup old identity
		if err := vault.BackupIdentity(); err != nil {
			t.Fatal(err)
		}
		// 2. Write a new identity (but DON'T re-encrypt store)
		newID, _ := age.GenerateX25519Identity()
		if err := vault.SetIdentity(newID.String()); err != nil {
			t.Fatal(err)
		}

		// Now: identity.txt has newID, store.age encrypted to oldID.
		// Running rotate-key should detect this and recover.

		if err := runRotateKey(nil); err != nil {
			t.Fatalf("recovery rotation error: %v", err)
		}

		// After recovery: vault decrypts via the auto-resolved new identity
		store, err := vault.ReadStore()
		if err != nil {
			t.Fatalf("can't decrypt after recovery: %v", err)
		}
		if store.GetEnv("global")["URL"] != "https://example.com" {
			t.Error("store data lost during recovery")
		}

		// Old identity no longer works
		_, err = vault.DecryptStore(oldID)
		if err == nil {
			t.Error("old identity should not decrypt after recovery")
		}
	})

	t.Run("errors when both keys dead", func(t *testing.T) {
		setupRotateKeyTest(t, nil)

		// Write a completely unrelated identity (no .old backup)
		unrelated, _ := age.GenerateX25519Identity()
		if err := vault.SetIdentity(unrelated.String()); err != nil {
			t.Fatal(err)
		}

		err := runRotateKey(nil)
		if err == nil {
			t.Fatal("should error when current identity can't decrypt store")
		}
	})
}

func TestRunRotateKey_BlocksSSHOnlyVault(t *testing.T) {
	// Set up a vault project without identity.txt (SSH-only)
	dir := vaultTestSetup(t)

	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	keyPath, _ := writeTestSSHKey(t, sshDir)

	if err := initcmd.InitVaultProject(nil, keyPath); err != nil {
		t.Fatalf("SSH init: %v", err)
	}

	err := runRotateKey(nil)
	if err == nil {
		t.Fatal("expected error for SSH-only vault")
	}
	if !strings.Contains(err.Error(), "age identity") {
		t.Errorf("expected age identity error, got: %v", err)
	}
}
