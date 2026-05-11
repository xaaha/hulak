package userflags

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// ── test helpers ────────────────────────────────────────────────────────────

// setupDoctorVaultProject creates a minimal vault project in tmpDir with .hulak/
// marker, store.age, recipients.txt, and identity.txt in a temp config dir.
// Returns the config dir path so the caller can set XDG_CONFIG_HOME.
func setupDoctorVaultProject(t *testing.T, tmpDir string) string {
	t.Helper()

	// Create .hulak/ marker directory
	hulakDir := filepath.Join(tmpDir, utils.HiddenProjectName)
	if err := os.MkdirAll(hulakDir, utils.DirPer); err != nil {
		t.Fatalf("mkdir .hulak: %v", err)
	}

	// Generate keypair and write identity
	key, err := vault.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	// Set up config dir for identity
	configDir := filepath.Join(tmpDir, "config")
	hulakConfigDir := filepath.Join(configDir, utils.ProjectName)
	if err := os.MkdirAll(hulakConfigDir, utils.SecretDirPer); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	identityPath := filepath.Join(hulakConfigDir, utils.IdentityFile)
	if err := os.WriteFile(identityPath, []byte(key.Identity.String()+"\n"), utils.SecretPer); err != nil {
		t.Fatalf("write identity: %v", err)
	}

	// Write recipients.txt
	recipientsContent := fmt.Sprintf("# test\n%s\n", key.Recipient.String())
	recipientsPath := filepath.Join(hulakDir, utils.RecipientsFile)
	if err := os.WriteFile(recipientsPath, []byte(recipientsContent), utils.FilePer); err != nil {
		t.Fatalf("write recipients: %v", err)
	}

	// Encrypt and write a minimal store
	store := &vault.Store{Envs: map[string]vault.Env{
		"global": {"TEST_KEY": "test_value"},
	}}
	storeJSON, err := store.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal store: %v", err)
	}
	ciphertext, err := vault.EncryptText(storeJSON, key.Recipient)
	if err != nil {
		t.Fatalf("encrypt store: %v", err)
	}
	storePath := filepath.Join(hulakDir, utils.StoreFile)
	if err := os.WriteFile(storePath, ciphertext, utils.SecretPer); err != nil {
		t.Fatalf("write store.age: %v", err)
	}

	return configDir
}

// assertFindingSeverity asserts a finding exists with the expected severity.
func assertFindingSeverity(t *testing.T, f *finding, checkID string, expected severity) {
	t.Helper()
	if f == nil {
		t.Fatalf("finding %q not found", checkID)
	}
	if f.severity != expected {
		t.Errorf("finding %q: got severity %v, want %v (message: %s)",
			checkID, f.severity, expected, f.message)
	}
}

// ── identity checks ────────────────────────────────────────────────────────

func TestCheckIdentityMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission checks not reliable on Windows")
	}

	t.Run("ok when 0600", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := setupDoctorVaultProject(t, tmpDir)
		t.Setenv("XDG_CONFIG_HOME", configDir)

		f := checkIdentityMode()
		assertFindingSeverity(t, &f, "identity-mode", sevOk)
	})

	t.Run("error when 0640", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := setupDoctorVaultProject(t, tmpDir)
		t.Setenv("XDG_CONFIG_HOME", configDir)

		identityPath := filepath.Join(configDir, utils.ProjectName, utils.IdentityFile)
		if err := os.Chmod(identityPath, 0o640); err != nil {
			t.Fatal(err)
		}

		f := checkIdentityMode()
		assertFindingSeverity(t, &f, "identity-mode", sevError)
		if f.auto == nil {
			t.Error("expected auto-fixable")
		}
	})

	t.Run("auto fix restores 0600", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := setupDoctorVaultProject(t, tmpDir)
		t.Setenv("XDG_CONFIG_HOME", configDir)

		identityPath := filepath.Join(configDir, utils.ProjectName, utils.IdentityFile)
		if err := os.Chmod(identityPath, 0o640); err != nil {
			t.Fatal(err)
		}

		f := checkIdentityMode()
		if f.auto == nil {
			t.Fatal("expected auto-fixable")
		}
		if err := f.auto(); err != nil {
			t.Fatalf("auto fix failed: %v", err)
		}

		info, err := os.Stat(identityPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != utils.SecretPer {
			t.Errorf("after fix: mode is %04o, want 0600", info.Mode().Perm())
		}
	})

	t.Run("info when identity not present", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "no-config"))

		f := checkIdentityMode()
		assertFindingSeverity(t, &f, "identity-mode", sevInfo)
	})
}

func TestCheckIdentityNotInGit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests not reliable on Windows")
	}

	t.Run("ok when not in git repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := setupDoctorVaultProject(t, tmpDir)
		t.Setenv("XDG_CONFIG_HOME", configDir)

		f := checkIdentityNotInGit()
		assertFindingSeverity(t, &f, "identity-in-git", sevOk)
	})

	t.Run("warn when identity is inside git repo but not tracked", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a fake git repo dir and put identity inside it
		gitDir := filepath.Join(tmpDir, "dotfiles")
		if err := os.MkdirAll(filepath.Join(gitDir, ".git"), utils.DirPer); err != nil {
			t.Fatal(err)
		}

		configDir := filepath.Join(gitDir, "config")
		hulakConfigDir := filepath.Join(configDir, utils.ProjectName)
		if err := os.MkdirAll(hulakConfigDir, utils.SecretDirPer); err != nil {
			t.Fatal(err)
		}
		identityPath := filepath.Join(hulakConfigDir, utils.IdentityFile)
		if err := os.WriteFile(identityPath, []byte("AGE-SECRET-KEY-test\n"), utils.SecretPer); err != nil {
			t.Fatal(err)
		}

		t.Setenv("XDG_CONFIG_HOME", configDir)

		f := checkIdentityNotInGit()
		assertFindingSeverity(t, &f, "identity-in-git", sevWarn)
		if !strings.Contains(f.message, "not tracked") {
			t.Errorf("expected 'not tracked' in message, got: %s", f.message)
		}
	})

	t.Run("error when identity is actually tracked by git", func(t *testing.T) {
		if _, lookErr := exec.LookPath("git"); lookErr != nil {
			t.Skip("git not available")
		}

		tmpDir := t.TempDir()
		repoDir := filepath.Join(tmpDir, "dotfiles")
		configDir := filepath.Join(repoDir, "config")
		hulakConfigDir := filepath.Join(configDir, utils.ProjectName)
		if err := os.MkdirAll(hulakConfigDir, utils.SecretDirPer); err != nil {
			t.Fatal(err)
		}
		identityPath := filepath.Join(hulakConfigDir, utils.IdentityFile)
		if err := os.WriteFile(identityPath, []byte("AGE-SECRET-KEY-test\n"), utils.SecretPer); err != nil {
			t.Fatal(err)
		}

		// Init real git repo and track the identity file
		restore := chdirTemp(t, repoDir)
		gitInit(t)
		gitAddCommit(t, "track identity")
		restore()

		t.Setenv("XDG_CONFIG_HOME", configDir)

		f := checkIdentityNotInGit()
		assertFindingSeverity(t, &f, "identity-in-git", sevError)
		if !strings.Contains(f.message, "tracked by git") {
			t.Errorf("expected 'tracked by git' in message, got: %s", f.message)
		}
	})

	t.Run("warn when identity is symlinked into git repo but not tracked", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a fake git repo (simulates dotfiles repo)
		dotfilesDir := filepath.Join(tmpDir, "dotfiles")
		if err := os.MkdirAll(filepath.Join(dotfilesDir, ".git"), utils.DirPer); err != nil {
			t.Fatal(err)
		}

		// Actual config dir inside dotfiles repo
		realConfigDir := filepath.Join(dotfilesDir, "config", utils.ProjectName)
		if err := os.MkdirAll(realConfigDir, utils.SecretDirPer); err != nil {
			t.Fatal(err)
		}
		realIdentityPath := filepath.Join(realConfigDir, utils.IdentityFile)
		if err := os.WriteFile(realIdentityPath, []byte("AGE-SECRET-KEY-test\n"), utils.SecretPer); err != nil {
			t.Fatal(err)
		}

		// Symlink from outside the git repo pointing in
		symlinkConfigDir := filepath.Join(tmpDir, "symlinked-config", utils.ProjectName)
		if err := os.MkdirAll(filepath.Dir(symlinkConfigDir), utils.DirPer); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(realConfigDir, symlinkConfigDir); err != nil {
			t.Fatal(err)
		}

		t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "symlinked-config"))

		f := checkIdentityNotInGit()
		assertFindingSeverity(t, &f, "identity-in-git", sevWarn)
	})
}

func TestCheckIdentityLeakedInProject(t *testing.T) {
	t.Run("ok when no secret keys in project", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := setupDoctorVaultProject(t, tmpDir)
		t.Setenv("XDG_CONFIG_HOME", configDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		f := checkIdentityLeakedInProject()
		assertFindingSeverity(t, &f, "identity-leaked-in-project", sevOk)
	})

	t.Run("error when YAML contains secret key", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := setupDoctorVaultProject(t, tmpDir)
		t.Setenv("XDG_CONFIG_HOME", configDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		// Write a file with a secret key
		leakyContent := "config:\n  key: AGE-SECRET-KEY-1XYZABC123\n"
		if err := os.WriteFile(
			filepath.Join(tmpDir, "config.yaml"), []byte(leakyContent), utils.FilePer,
		); err != nil {
			t.Fatal(err)
		}

		f := checkIdentityLeakedInProject()
		assertFindingSeverity(t, &f, "identity-leaked-in-project", sevError)
		if !strings.Contains(f.message, "config.yaml") {
			t.Errorf("expected leaked file name in message, got: %s", f.message)
		}
	})
}

// ── config dir check ───────────────────────────────────────────────────────

func TestCheckConfigDirMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission checks not reliable on Windows")
	}

	t.Run("ok when 0700", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := setupDoctorVaultProject(t, tmpDir)
		t.Setenv("XDG_CONFIG_HOME", configDir)

		f := checkConfigDirMode()
		assertFindingSeverity(t, &f, "config-dir-mode", sevOk)
	})

	t.Run("warn when 0755", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := setupDoctorVaultProject(t, tmpDir)
		t.Setenv("XDG_CONFIG_HOME", configDir)

		hulakConfigDir := filepath.Join(configDir, utils.ProjectName)
		if err := os.Chmod(hulakConfigDir, 0o755); err != nil {
			t.Fatal(err)
		}

		f := checkConfigDirMode()
		assertFindingSeverity(t, &f, "config-dir-mode", sevWarn)
		if f.auto == nil {
			t.Error("expected auto-fixable")
		}
	})
}

// ── store checks ───────────────────────────────────────────────────────────

func TestCheckStoreMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission checks not reliable on Windows")
	}

	t.Run("ok when 0600", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		f := checkStoreMode()
		assertFindingSeverity(t, &f, "store-mode", sevOk)
	})

	t.Run("error when 0644", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		storePath := filepath.Join(tmpDir, utils.HiddenProjectName, utils.StoreFile)
		if err := os.Chmod(storePath, 0o644); err != nil {
			t.Fatal(err)
		}

		f := checkStoreMode()
		assertFindingSeverity(t, &f, "store-mode", sevError)
		if f.auto == nil {
			t.Error("expected auto-fixable")
		}
	})
}

func TestCheckStoreEncrypted(t *testing.T) {
	t.Run("ok with valid age header", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		f := checkStoreEncrypted()
		assertFindingSeverity(t, &f, "store-encrypted", sevOk)
	})

	t.Run("error when plaintext", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		// Overwrite store.age with plaintext
		storePath := filepath.Join(tmpDir, utils.HiddenProjectName, utils.StoreFile)
		if err := os.WriteFile(storePath, []byte(`{"global":{"key":"value"}}`), utils.SecretPer); err != nil {
			t.Fatal(err)
		}

		f := checkStoreEncrypted()
		assertFindingSeverity(t, &f, "store-encrypted", sevError)
	})
}

func TestCheckStoreDecrypts(t *testing.T) {
	t.Run("ok with matching identity", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := setupDoctorVaultProject(t, tmpDir)
		t.Setenv("XDG_CONFIG_HOME", configDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		f := checkStoreDecrypts()
		assertFindingSeverity(t, &f, "store-decrypts", sevOk)
	})

	t.Run("error with wrong identity", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := setupDoctorVaultProject(t, tmpDir)
		t.Setenv("XDG_CONFIG_HOME", configDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		// Generate a different key and overwrite identity
		wrongKey, err := vault.GenerateKeyPair()
		if err != nil {
			t.Fatal(err)
		}
		identityPath := filepath.Join(configDir, utils.ProjectName, utils.IdentityFile)
		if err := os.WriteFile(identityPath, []byte(wrongKey.Identity.String()+"\n"), utils.SecretPer); err != nil {
			t.Fatal(err)
		}

		f := checkStoreDecrypts()
		assertFindingSeverity(t, &f, "store-decrypts", sevError)
	})
}

// ── recipients checks ──────────────────────────────────────────────────────

func TestCheckRecipientsExist(t *testing.T) {
	t.Run("ok when exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		f := checkRecipientsExist()
		assertFindingSeverity(t, &f, "recipients-exist", sevOk)
	})

	t.Run("error when missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		recipientsPath := filepath.Join(tmpDir, utils.HiddenProjectName, utils.RecipientsFile)
		os.Remove(recipientsPath)

		f := checkRecipientsExist()
		assertFindingSeverity(t, &f, "recipients-exist", sevError)
	})
}

func TestCheckRecipientsValid(t *testing.T) {
	t.Run("ok with valid entries", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		f := checkRecipientsValid()
		assertFindingSeverity(t, &f, "recipients-valid", sevOk)
	})

	t.Run("error when all comments", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		recipientsPath := filepath.Join(tmpDir, utils.HiddenProjectName, utils.RecipientsFile)
		if err := os.WriteFile(recipientsPath, []byte("# just a comment\n\n"), utils.FilePer); err != nil {
			t.Fatal(err)
		}

		f := checkRecipientsValid()
		assertFindingSeverity(t, &f, "recipients-valid", sevError)
	})
}

func TestCheckRecipientsMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission checks not reliable on Windows")
	}

	t.Run("ok when 0644", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		f := checkRecipientsMode()
		assertFindingSeverity(t, &f, "recipients-mode", sevOk)
	})

	t.Run("warn when 0600", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		recipientsPath := filepath.Join(tmpDir, utils.HiddenProjectName, utils.RecipientsFile)
		if err := os.Chmod(recipientsPath, 0o600); err != nil {
			t.Fatal(err)
		}

		f := checkRecipientsMode()
		assertFindingSeverity(t, &f, "recipients-mode", sevWarn)
		if f.auto == nil {
			t.Error("expected auto-fixable")
		}
	})
}

// ── drift check ────────────────────────────────────────────────────────────

func TestCheckRecipientDrift(t *testing.T) {
	t.Run("ok when counts match", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		f := checkRecipientDrift()
		assertFindingSeverity(t, &f, "recipient-drift", sevOk)
	})

	t.Run("warn when recipients added but store not re-encrypted", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		// Add a second recipient to recipients.txt without re-encrypting
		key2, err := vault.GenerateKeyPair()
		if err != nil {
			t.Fatal(err)
		}

		recipientsPath := filepath.Join(tmpDir, utils.HiddenProjectName, utils.RecipientsFile)
		data, err := os.ReadFile(recipientsPath)
		if err != nil {
			t.Fatal(err)
		}
		newData := string(data) + fmt.Sprintf("# extra\n%s\n", key2.Recipient.String())
		if err := os.WriteFile(recipientsPath, []byte(newData), utils.FilePer); err != nil {
			t.Fatal(err)
		}

		f := checkRecipientDrift()
		assertFindingSeverity(t, &f, "recipient-drift", sevWarn)
		if !strings.Contains(f.message, "2 entries") {
			t.Errorf("expected mention of 2 entries, got: %s", f.message)
		}
	})
}

func TestCountStanzas(t *testing.T) {
	t.Run("counts single recipient", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)

		storePath := filepath.Join(tmpDir, utils.HiddenProjectName, utils.StoreFile)
		count, err := countStanzas(storePath)
		if err != nil {
			t.Fatalf("countStanzas: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 stanza, got %d", count)
		}
	})

	t.Run("counts multi-recipient", func(t *testing.T) {
		tmpDir := t.TempDir()
		hulakDir := filepath.Join(tmpDir, utils.HiddenProjectName)
		if err := os.MkdirAll(hulakDir, utils.DirPer); err != nil {
			t.Fatal(err)
		}

		key1, _ := vault.GenerateKeyPair()
		key2, _ := vault.GenerateKeyPair()

		store := &vault.Store{Envs: map[string]vault.Env{
			"global": {"KEY": "val"},
		}}
		storeJSON, _ := store.MarshalJSON()
		ciphertext, err := vault.EncryptText(storeJSON, key1.Recipient, key2.Recipient)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		storePath := filepath.Join(hulakDir, utils.StoreFile)
		if err := os.WriteFile(storePath, ciphertext, utils.SecretPer); err != nil {
			t.Fatal(err)
		}

		count, err := countStanzas(storePath)
		if err != nil {
			t.Fatalf("countStanzas: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 stanzas, got %d", count)
		}
	})

	t.Run("errors on armored format", func(t *testing.T) {
		tmpDir := t.TempDir()
		armoredPath := filepath.Join(tmpDir, "armored.age")
		content := "-----BEGIN AGE ENCRYPTED FILE-----\nYWdlLWVuY3J5cHRpb24=\n-----END AGE ENCRYPTED FILE-----\n"
		if err := os.WriteFile(armoredPath, []byte(content), utils.SecretPer); err != nil {
			t.Fatal(err)
		}

		_, err := countStanzas(armoredPath)
		if err == nil {
			t.Error("expected error for armored format")
		}
	})

	t.Run("errors on non-age file", func(t *testing.T) {
		tmpDir := t.TempDir()
		junkPath := filepath.Join(tmpDir, "junk.age")
		if err := os.WriteFile(junkPath, []byte("not an age file"), utils.SecretPer); err != nil {
			t.Fatal(err)
		}

		_, err := countStanzas(junkPath)
		if err == nil {
			t.Error("expected error for non-age file")
		}
	})
}

// ── remaining checks ───────────────────────────────────────────────────────

func TestCheckLegacyKeyPub(t *testing.T) {
	t.Run("ok when no key.pub", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		f := checkLegacyKeyPub()
		assertFindingSeverity(t, &f, "legacy-key-pub", sevOk)
	})

	t.Run("info when key.pub exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		keyPubPath := filepath.Join(tmpDir, utils.HiddenProjectName, "key.pub")
		if err := os.WriteFile(keyPubPath, []byte("age1..."), utils.FilePer); err != nil {
			t.Fatal(err)
		}

		f := checkLegacyKeyPub()
		assertFindingSeverity(t, &f, "legacy-key-pub", sevInfo)
	})
}

func TestCheckDualBackend(t *testing.T) {
	t.Run("ok when only vault", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		f := checkDualBackend()
		assertFindingSeverity(t, &f, "dual-backend", sevOk)
	})

	t.Run("error when both env/ and store.age", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		// Create env/ alongside .hulak/
		createEnvDir(t, tmpDir)

		f := checkDualBackend()
		assertFindingSeverity(t, &f, "dual-backend", sevError)
	})
}

func TestCheckDualIdentity(t *testing.T) {
	t.Run("ok when only file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := setupDoctorVaultProject(t, tmpDir)
		t.Setenv("XDG_CONFIG_HOME", configDir)
		// Make sure HULAK_MASTER_KEY is not set
		t.Setenv(utils.MasterKey, "")

		f := checkDualIdentity()
		assertFindingSeverity(t, &f, "dual-identity", sevOk)
	})

	t.Run("info when both set", func(t *testing.T) {
		tmpDir := t.TempDir()
		configDir := setupDoctorVaultProject(t, tmpDir)
		t.Setenv("XDG_CONFIG_HOME", configDir)

		key, _ := vault.GenerateKeyPair()
		t.Setenv(utils.MasterKey, key.Identity.String())

		f := checkDualIdentity()
		assertFindingSeverity(t, &f, "dual-identity", sevInfo)
	})
}

func TestCheckStoreSize(t *testing.T) {
	t.Run("ok when small", func(t *testing.T) {
		tmpDir := t.TempDir()
		setupDoctorVaultProject(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		f := checkStoreSize()
		assertFindingSeverity(t, &f, "store-size", sevOk)
	})

	// Testing the >1 MiB case would require writing a large file;
	// skipping to keep tests fast. The logic is a straightforward size comparison.
}

// ── report and output ──────────────────────────────────────────────────────

func TestDoctorReportSummary(t *testing.T) {
	r := &doctorReport{
		findings: []finding{
			{check: "a", severity: sevOk},
			{check: "b", severity: sevOk},
			{check: "c", severity: sevWarn},
			{check: "d", severity: sevError},
			{check: "e", severity: sevInfo},
		},
	}

	s := r.summary()
	if s.Ok != 2 || s.Warn != 1 || s.Error != 1 || s.Info != 1 {
		t.Errorf("summary: got %+v", s)
	}
}

func TestDoctorReportExitCode(t *testing.T) {
	tests := []struct {
		name     string
		findings []finding
		want     int
	}{
		{"ok only", []finding{{severity: sevOk}}, 0},
		{"info only", []finding{{severity: sevInfo}}, 0},
		{"warn", []finding{{severity: sevOk}, {severity: sevWarn}}, 1},
		{"error", []finding{{severity: sevOk}, {severity: sevError}}, 2},
		{"error trumps warn", []finding{{severity: sevWarn}, {severity: sevError}}, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &doctorReport{findings: tc.findings}
			if got := r.exitCode(); got != tc.want {
				t.Errorf("exitCode: got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestSeverityString(t *testing.T) {
	tests := []struct {
		s    severity
		want string
	}{
		{sevInfo, "info"},
		{sevOk, "ok"},
		{sevWarn, "warn"},
		{sevError, "error"},
	}
	for _, tc := range tests {
		if got := tc.s.String(); got != tc.want {
			t.Errorf("severity(%d).String() = %q, want %q", tc.s, got, tc.want)
		}
	}
}

// ── JSON output ─────────────────────────────────────────────────────────

func TestPrintJSON(t *testing.T) {
	r := &doctorReport{
		project: "/test/project",
		backend: "vault (.hulak/store.age)",
		findings: []finding{
			{check: "test-ok", severity: sevOk, message: "all good"},
			{check: "test-warn", severity: sevWarn, message: "fixme", fix: "do X", auto: func() error { return nil }},
			{check: "test-info", severity: sevInfo, message: "fyi"},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	rp, wp, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = wp
	r.printJSON()
	wp.Close()
	os.Stdout = oldStdout

	var jr jsonReport
	if err := json.NewDecoder(rp).Decode(&jr); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if jr.Project != "/test/project" {
		t.Errorf("project: got %q", jr.Project)
	}
	if jr.Backend != "vault (.hulak/store.age)" {
		t.Errorf("backend: got %q", jr.Backend)
	}
	if len(jr.Findings) != 3 {
		t.Fatalf("findings: got %d, want 3", len(jr.Findings))
	}

	// Verify check IDs and severity strings
	if jr.Findings[0].Check != "test-ok" || jr.Findings[0].Severity != "ok" {
		t.Errorf("finding[0]: %+v", jr.Findings[0])
	}
	if jr.Findings[1].Check != "test-warn" || jr.Findings[1].Severity != "warn" {
		t.Errorf("finding[1]: %+v", jr.Findings[1])
	}
	if !jr.Findings[1].AutoFixable {
		t.Error("finding[1] should be auto_fixable")
	}
	if jr.Findings[2].Check != "test-info" || jr.Findings[2].Severity != "info" {
		t.Errorf("finding[2]: %+v", jr.Findings[2])
	}

	// Summary
	if jr.Summary.Ok != 1 || jr.Summary.Warn != 1 || jr.Summary.Info != 1 || jr.Summary.Error != 0 {
		t.Errorf("summary: %+v", jr.Summary)
	}
}

// ── runFixes ────────────────────────────────────────────────────────────

func TestRunFixesWithFailure(t *testing.T) {
	callCount := 0
	r := &doctorReport{
		findings: []finding{
			{
				check:    "will-fail",
				severity: sevWarn,
				message:  "bad perms",
				fix:      "chmod 600",
				auto:     func() error { return fmt.Errorf("permission denied") },
			},
			{
				check:    "will-succeed",
				severity: sevWarn,
				message:  "other issue",
				fix:      "fix it",
				auto: func() error {
					callCount++
					return nil
				},
			},
		},
	}

	// Run with --yes to skip prompts
	r.runFixes(true)

	// Failed fix should keep severity
	if r.findings[0].severity != sevWarn {
		t.Errorf("failed fix should keep sevWarn, got %v", r.findings[0].severity)
	}
	if r.findings[0].auto == nil {
		t.Error("failed fix should keep auto func")
	}

	// Successful fix should upgrade to sevOk
	if r.findings[1].severity != sevOk {
		t.Errorf("successful fix should be sevOk, got %v", r.findings[1].severity)
	}
	if r.findings[1].auto != nil {
		t.Error("successful fix should clear auto func")
	}
	if callCount != 1 {
		t.Errorf("expected successful auto to be called once, got %d", callCount)
	}
}

// ── runDoctor integration ───────────────────────────────────────────────

func TestRunDoctor(t *testing.T) {
	t.Run("not a hulak project returns 0", func(t *testing.T) {
		tmpDir := t.TempDir()
		restore := chdirTemp(t, tmpDir)
		defer restore()

		code := runDoctor(doctorOpts{})
		if code != 0 {
			t.Errorf("expected exit code 0 for non-project dir, got %d", code)
		}
	})

	t.Run("healthy classic project returns 0", func(t *testing.T) {
		tmpDir := t.TempDir()
		envDir := createEnvDir(t, tmpDir)
		createEnvFile(t, envDir, "global", utils.SecretPer, "KEY=value")

		content := utils.EnvironmentFolder + "/\n"
		if err := os.WriteFile(
			filepath.Join(tmpDir, ".gitignore"), []byte(content), utils.FilePer,
		); err != nil {
			t.Fatal(err)
		}

		restore := chdirTemp(t, tmpDir)
		defer restore()

		code := runDoctor(doctorOpts{})
		// Classic always has migrate-suggestion (info), so 0 is expected
		if code != 0 {
			t.Errorf("expected exit code 0 for healthy classic project, got %d", code)
		}
	})

	t.Run("classic project with warnings returns 1", func(t *testing.T) {
		tmpDir := t.TempDir()
		envDir := createEnvDir(t, tmpDir)
		// Loose permissions → warning
		createEnvFile(t, envDir, "global", 0o644, "KEY=value")

		restore := chdirTemp(t, tmpDir)
		defer restore()

		code := runDoctor(doctorOpts{})
		if code != 1 {
			t.Errorf("expected exit code 1 for classic project with warnings, got %d", code)
		}
	})

	t.Run("fix and json are mutually exclusive", func(t *testing.T) {
		tmpDir := t.TempDir()
		createEnvDir(t, tmpDir)
		restore := chdirTemp(t, tmpDir)
		defer restore()

		code := runDoctor(doctorOpts{fix: true, jsonOut: true})
		if code != 1 {
			t.Errorf("expected exit code 1 for --fix --json, got %d", code)
		}
	})

	t.Run("json output is valid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		envDir := createEnvDir(t, tmpDir)
		createEnvFile(t, envDir, "global", utils.SecretPer, "KEY=value")
		content := utils.EnvironmentFolder + "/\n"
		if err := os.WriteFile(
			filepath.Join(tmpDir, ".gitignore"), []byte(content), utils.FilePer,
		); err != nil {
			t.Fatal(err)
		}

		restore := chdirTemp(t, tmpDir)
		defer restore()

		// Capture stdout
		oldStdout := os.Stdout
		rp, wp, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout = wp
		code := runDoctor(doctorOpts{jsonOut: true})
		wp.Close()
		os.Stdout = oldStdout

		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}

		var jr jsonReport
		if err := json.NewDecoder(rp).Decode(&jr); err != nil {
			t.Fatalf("JSON output is not valid: %v", err)
		}
		if jr.Backend != "classic (env/)" {
			t.Errorf("backend: got %q, want classic", jr.Backend)
		}
	})
}

// ── isFileGitTracked ────────────────────────────────────────────────────

func TestIsFileGitTracked(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	t.Run("tracked file returns true", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "tracked.txt")
		if err := os.WriteFile(filePath, []byte("data"), utils.FilePer); err != nil {
			t.Fatal(err)
		}

		restore := chdirTemp(t, tmpDir)
		gitInit(t)
		gitAddCommit(t, "add file")
		restore()

		if !isFileGitTracked(tmpDir, filePath) {
			t.Error("expected tracked file to return true")
		}
	})

	t.Run("untracked file returns false", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a tracked file so git init has something to commit
		if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("hi"), utils.FilePer); err != nil {
			t.Fatal(err)
		}
		restore := chdirTemp(t, tmpDir)
		gitInit(t)
		gitAddCommit(t, "initial")
		restore()

		// Now add an untracked file
		untrackedPath := filepath.Join(tmpDir, "untracked.txt")
		if err := os.WriteFile(untrackedPath, []byte("data"), utils.FilePer); err != nil {
			t.Fatal(err)
		}

		if isFileGitTracked(tmpDir, untrackedPath) {
			t.Error("expected untracked file to return false")
		}
	})

	t.Run("non-git dir returns false", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "file.txt")
		if err := os.WriteFile(filePath, []byte("data"), utils.FilePer); err != nil {
			t.Fatal(err)
		}

		if isFileGitTracked(tmpDir, filePath) {
			t.Error("expected non-git dir to return false")
		}
	})
}
