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

// seedEnv creates envName in the current vault project and populates it with
// the given key-value pairs. Test helper — fails the test on any vault error.
func seedEnv(t *testing.T, envName string, kv map[string]any) {
	t.Helper()
	store, err := vault.ReadStore()
	if err != nil {
		t.Fatalf("ReadStore: %v", err)
	}
	store.EnsureSection(envName)
	for k, v := range kv {
		store.SetKey(envName, k, v)
	}
	if err := vault.WriteStoreToRecipients(store); err != nil {
		t.Fatalf("WriteStoreToRecipients: %v", err)
	}
}

func TestRunDeleteEnv(t *testing.T) {
	t.Run("deletes empty env without prompting", func(t *testing.T) {
		setupVaultProject(t)
		if err := runEnvCreate(nil, "temp"); err != nil {
			t.Fatal(err)
		}
		// Sentinel: if confirmDestroy reaches the prompt for count=0, fail.
		prompt := stubConfirm(t, false, nil)

		if err := runDeleteEnv(nil, "temp", false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *prompt != "" {
			t.Errorf("empty env should skip prompt, got %q", *prompt)
		}

		store, _ := vault.ReadStore()
		if store.GetEnv("temp") != nil {
			t.Error("env should be gone after delete")
		}
	})

	t.Run("non-empty env: prompts and deletes on accept", func(t *testing.T) {
		setupVaultProject(t)
		seedEnv(t, "prod", map[string]any{"API_KEY": "sk-xxx", "URL": "https://x"})
		prompt := stubConfirm(t, true, nil)

		if err := runDeleteEnv(nil, "prod", false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(*prompt, "2 keys in \"prod\"") {
			t.Errorf("expected prompt to include count + env, got %q", *prompt)
		}

		store, _ := vault.ReadStore()
		if store.GetEnv("prod") != nil {
			t.Error("env should be gone after confirmed delete")
		}
	})

	t.Run("non-empty env: declines preserves env", func(t *testing.T) {
		setupVaultProject(t)
		seedEnv(t, "prod", map[string]any{"API_KEY": "sk-xxx"})
		stubConfirm(t, false /*decline*/, nil)

		if err := runDeleteEnv(nil, "prod", false); err != nil {
			t.Fatalf("unexpected error on decline: %v", err)
		}

		store, _ := vault.ReadStore()
		env := store.GetEnv("prod")
		if env == nil {
			t.Fatal("env should still exist after decline")
		}
		if env["API_KEY"] != "sk-xxx" {
			t.Error("decline should not mutate env contents")
		}
	})

	t.Run("--yes skips prompt and deletes regardless of count", func(t *testing.T) {
		setupVaultProject(t)
		seedEnv(t, "prod", map[string]any{"K1": "v1", "K2": "v2", "K3": "v3"})
		// Sentinel: prompt must not be called when force=true.
		prompt := stubConfirm(t, false, nil)

		if err := runDeleteEnv(nil, "prod", true /*force*/); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *prompt != "" {
			t.Errorf("--yes should skip prompt entirely, got %q", *prompt)
		}

		store, _ := vault.ReadStore()
		if store.GetEnv("prod") != nil {
			t.Error("env should be gone after --yes delete")
		}
	})

	t.Run("singular phrasing for count == 1", func(t *testing.T) {
		setupVaultProject(t)
		seedEnv(t, "prod", map[string]any{"only": "v"})
		prompt := stubConfirm(t, true, nil)

		if err := runDeleteEnv(nil, "prod", false); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(*prompt, "1 key in \"prod\"") {
			t.Errorf("expected singular phrasing, got %q", *prompt)
		}
	})

	t.Run("missing env errors", func(t *testing.T) {
		setupVaultProject(t)
		err := runDeleteEnv(nil, "ghost", false)
		if err == nil {
			t.Fatal("expected error for missing env, got nil")
		}
		if !strings.Contains(err.Error(), `environment "ghost"`) {
			t.Errorf("error should name the env, got: %v", err)
		}
	})

	t.Run("preserves siblings", func(t *testing.T) {
		setupVaultProject(t)
		seedEnv(t, "prod", map[string]any{"K": "v"})
		seedEnv(t, "staging", map[string]any{"S": "stagingv"})
		stubConfirm(t, true, nil)

		if err := runDeleteEnv(nil, "prod", false); err != nil {
			t.Fatal(err)
		}

		store, _ := vault.ReadStore()
		if store.GetEnv("prod") != nil {
			t.Error("prod should be deleted")
		}
		if store.GetEnv("staging")["S"] != "stagingv" {
			t.Error("delete disturbed sibling env")
		}
	})

	t.Run("rejects extra positional args", func(t *testing.T) {
		setupVaultProject(t)
		err := runDeleteEnv([]string{"unexpected"}, "prod", false)
		if err == nil {
			t.Fatal("expected error on extra positional, got nil")
		}
	})
}
