package userflags

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// setupGenIdentityTest creates an isolated config dir and returns its path.
// No vault is created — gen-identity is meant to run outside a vault.
func setupGenIdentityTest(t *testing.T) string {
	t.Helper()

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

	return configDir
}

func TestRunGenIdentity(t *testing.T) {
	t.Run("creates identity.txt when absent", func(t *testing.T) {
		configDir := setupGenIdentityTest(t)

		if err := runGenIdentity(nil, ""); err != nil {
			t.Fatalf("runGenIdentity: %v", err)
		}

		identityPath := filepath.Join(configDir, utils.IdentityFile)
		if !utils.FileExists(identityPath) {
			t.Fatalf("expected %s to exist after gen-identity", identityPath)
		}

		// File should be parseable as an age identity
		if _, err := vault.LoadIdentity(); err != nil {
			t.Errorf("identity should be parseable: %v", err)
		}
	})

	t.Run("refuses to overwrite existing identity", func(t *testing.T) {
		setupGenIdentityTest(t)

		// First run succeeds
		if err := runGenIdentity(nil, ""); err != nil {
			t.Fatalf("first runGenIdentity: %v", err)
		}
		first, err := vault.LoadIdentity()
		if err != nil {
			t.Fatalf("LoadIdentity after first run: %v", err)
		}

		// Second run errors with helpful message
		err = runGenIdentity(nil, "")
		if err == nil {
			t.Fatal("expected error when identity already exists")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("error should mention 'already exists': %v", err)
		}
		if !strings.Contains(err.Error(), "identity rotate") {
			t.Errorf("error should point at identity rotate: %v", err)
		}

		// Identity unchanged
		second, err := vault.LoadIdentity()
		if err != nil {
			t.Fatalf("LoadIdentity after refused run: %v", err)
		}
		if first.String() != second.String() {
			t.Error("identity was changed despite refusal")
		}
	})

	t.Run("rejects positional arguments", func(t *testing.T) {
		setupGenIdentityTest(t)

		err := runGenIdentity([]string{"unexpected"}, "")
		if err == nil {
			t.Fatal("expected error for unexpected argument")
		}
		if !strings.Contains(err.Error(), "too many arguments") {
			t.Errorf("error should mention 'too many arguments': %v", err)
		}
	})

	t.Run("does not create .hulak/ in cwd (no orphan vault)", func(t *testing.T) {
		setupGenIdentityTest(t)
		cwd := t.TempDir()
		oldWd, _ := os.Getwd()
		_ = os.Chdir(cwd)
		t.Cleanup(func() { _ = os.Chdir(oldWd) })

		if err := runGenIdentity(nil, ""); err != nil {
			t.Fatalf("runGenIdentity: %v", err)
		}

		entries, err := os.ReadDir(cwd)
		if err != nil {
			t.Fatalf("read cwd: %v", err)
		}
		for _, e := range entries {
			if e.Name() == utils.HiddenProjectName {
				t.Errorf("gen-identity created %s in cwd — should not touch cwd", e.Name())
			}
		}
	})
}
