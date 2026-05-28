package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// setupImportKeyTest creates a temp project, isolates the config dir, and
// optionally creates an encrypted store with the given identity as a recipient.
// Returns the keyPath to write candidate import material into.
func setupImportKeyTest(t *testing.T, withVault bool, vaultIdentity *age.X25519Identity) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	configTmp := t.TempDir()
	configTmp, _ = filepath.EvalSymlinks(configTmp)
	t.Setenv("XDG_CONFIG_HOME", configTmp)
	t.Setenv("HULAK_MASTER_KEY", "")
	t.Setenv("HULAK_SSH_IDENTITY", "")
	t.Setenv("HOME", t.TempDir())

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

	if !withVault {
		return tmpDir
	}

	// Create a vault encrypted to vaultIdentity
	hulakDir := filepath.Join(tmpDir, utils.HiddenProjectName)
	if err := os.Mkdir(hulakDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}

	entries := []vault.RecipientEntry{
		{Key: vaultIdentity.Recipient().String(), Name: "test"},
	}
	if err := vault.SaveRecipients(entries); err != nil {
		t.Fatal(err)
	}
	recipients, err := vault.RecipientsFromEntries(entries)
	if err != nil {
		t.Fatal(err)
	}
	store := &vault.Store{Envs: map[string]vault.Env{"global": {"K": "v"}}}
	if err := vault.WriteStore(store, recipients...); err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func writeKeyFile(t *testing.T, dir string, identity *age.X25519Identity) string {
	t.Helper()
	keyPath := filepath.Join(dir, "candidate-key.txt")
	if err := os.WriteFile(keyPath, []byte(identity.String()+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return keyPath
}

func TestRunImportKey(t *testing.T) {
	t.Run("rejects key that is not a vault recipient", func(t *testing.T) {
		recipientID, _ := age.GenerateX25519Identity()
		dir := setupImportKeyTest(t, true, recipientID)

		// Candidate is a DIFFERENT identity — not a recipient
		otherID, _ := age.GenerateX25519Identity()
		keyPath := writeKeyFile(t, dir, otherID)

		err := runImportKey([]string{keyPath}, false, false, "")
		if err == nil {
			t.Fatal("expected error when importing non-recipient key")
		}
		if !strings.Contains(err.Error(), "not a recipient") {
			t.Errorf("error should mention 'not a recipient': %v", err)
		}
		if !strings.Contains(err.Error(), "add-recipient") {
			t.Errorf("error should point at add-recipient: %v", err)
		}

		// Identity file should NOT exist (rejected before write)
		if vault.IdentityExists() {
			t.Error("identity.txt should not exist after rejected import")
		}
	})

	t.Run("accepts key that IS a vault recipient", func(t *testing.T) {
		recipientID, _ := age.GenerateX25519Identity()
		dir := setupImportKeyTest(t, true, recipientID)

		// Candidate IS the recipient
		keyPath := writeKeyFile(t, dir, recipientID)

		if err := runImportKey([]string{keyPath}, false, false, ""); err != nil {
			t.Fatalf("import should succeed: %v", err)
		}

		if !vault.IdentityExists() {
			t.Error("identity.txt should exist after successful import")
		}
	})

	t.Run("--name auto-registers a non-recipient key when SSH decrypts", func(t *testing.T) {
		recipientID, _ := age.GenerateX25519Identity()
		dir := setupImportKeyTest(t, true, recipientID)

		// Point HULAK_SSH_IDENTITY at a key that's NOT the vault recipient —
		// but write the vault with an SSH recipient instead so we can demo
		// "another working identity unlocks the store while we register a
		// new age key." Simpler: drop the existing recipient key into the
		// identity.txt path temporarily so it serves as the decrypt path.
		identityPath, _ := vault.IdentityPath()
		if err := os.WriteFile(identityPath, []byte(recipientID.String()+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(identityPath) })

		// Candidate is a NEW age key, not yet a recipient.
		newID, _ := age.GenerateX25519Identity()
		keyPath := writeKeyFile(t, dir, newID)

		// --name triggers auto-register: recipientID decrypts → store
		// re-encrypted to include newID → identity.txt then overwritten
		// with newID.
		if err := runImportKey([]string{keyPath}, false, true, "alice-laptop"); err != nil {
			t.Fatalf("--name auto-register should succeed: %v", err)
		}

		// recipients.txt should now contain newID's pubkey
		recipPath, _ := vault.RecipientsFilePath()
		data, _ := os.ReadFile(recipPath)
		if !strings.Contains(string(data), newID.Recipient().String()) {
			t.Errorf("recipients.txt should contain new pubkey: %s", string(data))
		}
		// And the new identity should now decrypt the store directly.
		if _, err := vault.ReadStore(); err != nil {
			t.Errorf("store should decrypt after auto-register: %v", err)
		}
	})

	t.Run("corrupt store.age is not misreported as 'not a recipient'", func(t *testing.T) {
		recipientID, _ := age.GenerateX25519Identity()
		dir := setupImportKeyTest(t, true, recipientID)

		// Corrupt the store: overwrite with garbage. Decryption will fail
		// with a parse/format error, NOT "no identity matched."
		storePath, _ := vault.StorePath()
		if err := os.WriteFile(storePath, []byte("not an age file"), 0o600); err != nil {
			t.Fatal(err)
		}

		// Candidate is the legitimate recipient. Pre-fix this would have
		// reported "not a recipient" misleadingly.
		keyPath := writeKeyFile(t, dir, recipientID)

		err := runImportKey([]string{keyPath}, false, false, "")
		if err == nil {
			t.Fatal("expected error on corrupt store")
		}
		if strings.Contains(err.Error(), "not a recipient") {
			t.Errorf("error should not say 'not a recipient' for a corrupt store: %v", err)
		}
		if !strings.Contains(err.Error(), "corrupt") {
			t.Errorf("error should indicate corruption: %v", err)
		}
	})

	t.Run("no vault in cwd → no validation (pre-clone staging)", func(t *testing.T) {
		dir := setupImportKeyTest(t, false, nil)

		// No vault in cwd — validation should be skipped
		otherID, _ := age.GenerateX25519Identity()
		keyPath := writeKeyFile(t, dir, otherID)

		if err := runImportKey([]string{keyPath}, false, false, ""); err != nil {
			t.Fatalf("import should succeed when no vault present: %v", err)
		}

		if !vault.IdentityExists() {
			t.Error("identity.txt should exist after pre-vault import")
		}
	})
}
