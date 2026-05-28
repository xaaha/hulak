package userflags

import (
	"os"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/vault"
)

func TestRunEnvCreate(t *testing.T) {
	t.Run("creates a new empty environment", func(t *testing.T) {
		setupVaultProject(t)

		if err := runEnvCreate(nil, "staging"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		store, err := vault.ReadStore()
		if err != nil {
			t.Fatalf("ReadStore: %v", err)
		}
		env := store.GetEnv("staging")
		if env == nil {
			t.Fatal("expected staging env to exist")
		}
		if len(env) != 0 {
			t.Errorf("expected empty env, got %d keys", len(env))
		}
	})

	t.Run("errors if env already exists", func(t *testing.T) {
		setupVaultProject(t)

		if err := runEnvCreate(nil, "prod"); err != nil {
			t.Fatalf("first create: %v", err)
		}

		err := runEnvCreate(nil, "prod")
		if err == nil {
			t.Fatal("expected error on second create, got nil")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("error should mention 'already exists', got: %v", err)
		}
	})

	t.Run("preserves siblings", func(t *testing.T) {
		setupVaultProject(t)

		if err := runEnvCreate(nil, "staging"); err != nil {
			t.Fatal(err)
		}
		// seed a key in staging by hand to verify create-prod doesn't touch it
		store, err := vault.ReadStore()
		if err != nil {
			t.Fatal(err)
		}
		store.SetKey("staging", "API_KEY", "sk-staging")
		if err := vault.WriteStoreToRecipients(store); err != nil {
			t.Fatal(err)
		}

		if err := runEnvCreate(nil, "prod"); err != nil {
			t.Fatalf("create prod: %v", err)
		}

		store2, err := vault.ReadStore()
		if err != nil {
			t.Fatal(err)
		}
		if store2.GetEnv("staging")["API_KEY"] != "sk-staging" {
			t.Error("create disturbed unrelated env")
		}
		if store2.GetEnv("prod") == nil || len(store2.GetEnv("prod")) != 0 {
			t.Error("expected empty prod env to be created")
		}
	})

	t.Run("requires --env", func(t *testing.T) {
		setupVaultProject(t)

		err := runEnvCreate(nil, "")
		if err == nil {
			t.Fatal("expected error without --env, got nil")
		}
		if !strings.Contains(err.Error(), "--env is required") {
			t.Errorf("error should mention --env required, got: %v", err)
		}
	})

	t.Run("rejects invalid env name", func(t *testing.T) {
		setupVaultProject(t)

		err := runEnvCreate(nil, "bad/name")
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
	})

	t.Run("rejects extra positional args", func(t *testing.T) {
		setupVaultProject(t)

		err := runEnvCreate([]string{"unexpected"}, "staging")
		if err == nil {
			t.Fatal("expected error on extra positional, got nil")
		}
		if !strings.Contains(err.Error(), "too many arguments") {
			t.Errorf("expected 'too many arguments', got: %v", err)
		}
	})

	t.Run("errors outside vault project", func(t *testing.T) {
		// Fresh cwd with no .hulak/.
		t.Cleanup(chdirTemp(t, t.TempDir()))
		// Avoid picking up an env identity from a previous test setup.
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		// Make sure no master-key shortcut covers for a missing project.
		_ = os.Unsetenv("HULAK_MASTER_KEY")

		err := runEnvCreate(nil, "staging")
		if err == nil {
			t.Fatal("expected error outside vault project, got nil")
		}
		if !strings.Contains(err.Error(), "no vault project") {
			t.Errorf("expected 'no vault project' error, got: %v", err)
		}
	})
}
