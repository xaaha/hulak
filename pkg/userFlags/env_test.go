package userflags

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// setupVaultProject prepares an isolated hulak project + config dir for tests.
// It chdirs into a fresh temp dir and points XDG_CONFIG_HOME at another temp dir
// so vault.EnsureKeypair stores the identity outside the user's real config.
func setupVaultProject(t *testing.T) string {
	t.Helper()

	configDir := t.TempDir()
	configDir, err := filepath.EvalSymlinks(configDir)
	if err != nil {
		t.Fatalf("resolve symlinks: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", configDir)

	projectDir := t.TempDir()
	projectDir, err = filepath.EvalSymlinks(projectDir)
	if err != nil {
		t.Fatalf("resolve symlinks: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, utils.HiddenProjectName), utils.DirPer); err != nil {
		t.Fatalf("mkdir .hulak: %v", err)
	}

	t.Cleanup(chdirTemp(t, projectDir))

	// EnsureKeypair creates identity + recipients.txt (via ensureRecipientsFile
	// in init.go flow). Tests that call runEnvSet get this for free, but tests
	// that call runEnvGet/Delete/List/Keys need the recipients file pre-seeded.
	ageKey, err := vault.EnsureKeypair()
	if err != nil {
		t.Fatalf("EnsureKeypair: %v", err)
	}
	if err := vault.SaveRecipients([]vault.RecipientEntry{
		{Key: ageKey.Recipient.String()},
	}); err != nil {
		t.Fatalf("SaveRecipients: %v", err)
	}

	return projectDir
}

// readStoredValue decrypts the store and returns the value at envName/key.
func readStoredValue(t *testing.T, envName, key string) any {
	t.Helper()
	store, err := vault.ReadStore()
	if err != nil {
		t.Fatalf("ReadStore: %v", err)
	}
	env := store.GetEnv(envName)
	if env == nil {
		t.Fatalf("env %q not found in store", envName)
	}
	return env[key]
}
