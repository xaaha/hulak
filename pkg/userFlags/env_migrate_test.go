package userflags

import (
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// setupMigrateTest creates a temp project dir with env/ files and optional pre-existing vault.
func setupMigrateTest(t *testing.T, envFiles map[string]string, existingStore *vault.Store) {
	t.Helper()

	tmpDir := t.TempDir()
	tmpDir, _ = filepath.EvalSymlinks(tmpDir)

	envDir := filepath.Join(tmpDir, utils.EnvironmentFolder)
	if err := os.Mkdir(envDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	for name, content := range envFiles {
		path := filepath.Join(envDir, name)
		if err := os.WriteFile(path, []byte(content), utils.FilePer); err != nil {
			t.Fatal(err)
		}
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

	// Chdir before any vault calls so FindProjectRoot resolves to tmpDir.
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	if existingStore != nil {
		hulakDir := filepath.Join(tmpDir, utils.HiddenProjectName)
		if err := os.Mkdir(hulakDir, utils.DirPer); err != nil {
			t.Fatal(err)
		}

		id, _ := age.GenerateX25519Identity()
		if err := vault.SetIdentity(id.String()); err != nil {
			t.Fatal(err)
		}
		if err := vault.SaveRecipients([]vault.RecipientEntry{
			{Key: id.Recipient().String()},
		}); err != nil {
			t.Fatal(err)
		}
		if err := vault.WriteStore(existingStore, id.Recipient()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunEnvMigrate(t *testing.T) {
	t.Run("migrates single env file", func(t *testing.T) {
		setupMigrateTest(t, map[string]string{
			"global.env": "KEY1=value1\nKEY2=42\n",
		}, nil)

		if err := runEnvMigrate(); err != nil {
			t.Fatalf("runEnvMigrate error: %v", err)
		}

		identity, err := vault.ResolveIdentity()
		if err != nil {
			t.Fatal(err)
		}
		store, err := vault.DecryptStore(identity)
		if err != nil {
			t.Fatal(err)
		}

		env := store.GetEnv("global")
		if env == nil {
			t.Fatal("expected 'global' section in store")
		}
		if env["KEY1"] != "value1" {
			t.Errorf("KEY1 = %v, want value1", env["KEY1"])
		}
	})

	t.Run("migrates multiple env files", func(t *testing.T) {
		setupMigrateTest(t, map[string]string{
			"global.env":  "URL=https://example.com\n",
			"staging.env": "URL=https://staging.example.com\nAPI_KEY=sk-123\n",
		}, nil)

		if err := runEnvMigrate(); err != nil {
			t.Fatalf("runEnvMigrate error: %v", err)
		}

		identity, _ := vault.ResolveIdentity()
		store, _ := vault.DecryptStore(identity)

		global := store.GetEnv("global")
		if global["URL"] != "https://example.com" {
			t.Errorf("global URL = %v", global["URL"])
		}

		staging := store.GetEnv("staging")
		if staging == nil {
			t.Fatal("expected 'staging' section")
		}
		if staging["API_KEY"] != "sk-123" {
			t.Errorf("staging API_KEY = %v", staging["API_KEY"])
		}
	})

	t.Run("existing store wins on conflicts", func(t *testing.T) {
		setupMigrateTest(t, map[string]string{
			"prod.env": "TOKEN=from_env\nNEW_KEY=new_value\n",
		}, &vault.Store{Envs: map[string]vault.Env{
			"prod": {"TOKEN": "from_store"},
		}})

		if err := runEnvMigrate(); err != nil {
			t.Fatalf("runEnvMigrate error: %v", err)
		}

		identity, _ := vault.ResolveIdentity()
		store, _ := vault.DecryptStore(identity)

		prod := store.GetEnv("prod")
		if prod["TOKEN"] != "from_store" {
			t.Errorf("TOKEN = %v, want from_store (existing wins)", prod["TOKEN"])
		}
		if prod["NEW_KEY"] != "new_value" {
			t.Errorf("NEW_KEY = %v, want new_value (new key added)", prod["NEW_KEY"])
		}
	})

	t.Run("preserves dollar var as literal", func(t *testing.T) {
		setupMigrateTest(t, map[string]string{
			"global.env": "TOKEN=$GITHUB_TOKEN\n",
		}, nil)

		if err := runEnvMigrate(); err != nil {
			t.Fatalf("runEnvMigrate error: %v", err)
		}

		identity, _ := vault.ResolveIdentity()
		store, _ := vault.DecryptStore(identity)

		env := store.GetEnv("global")
		if env["TOKEN"] != "$GITHUB_TOKEN" {
			t.Errorf("TOKEN = %v, want literal $GITHUB_TOKEN", env["TOKEN"])
		}
	})

	t.Run("skips non-dot-env files", func(t *testing.T) {
		setupMigrateTest(t, map[string]string{
			"global.env": "KEY=val\n",
			".env.bak":   "OLD=stale\n",
			"notes.txt":  "not an env file\n",
		}, nil)

		if err := runEnvMigrate(); err != nil {
			t.Fatalf("runEnvMigrate error: %v", err)
		}

		identity, _ := vault.ResolveIdentity()
		store, _ := vault.DecryptStore(identity)

		envs := store.ListEnvs()
		for _, name := range envs {
			if name == ".env" || name == "notes" {
				t.Errorf("unexpected env section %q — non-.env file was migrated", name)
			}
		}
	})

	t.Run("empty env file creates empty section", func(t *testing.T) {
		setupMigrateTest(t, map[string]string{
			"staging.env": "# only comments\n\n",
		}, nil)

		if err := runEnvMigrate(); err != nil {
			t.Fatalf("runEnvMigrate error: %v", err)
		}

		identity, _ := vault.ResolveIdentity()
		store, _ := vault.DecryptStore(identity)

		staging := store.GetEnv("staging")
		if staging == nil {
			t.Fatal("expected 'staging' section for empty .env file")
		}
		if len(staging) != 0 {
			t.Errorf("expected empty section, got %d keys", len(staging))
		}
	})

	t.Run("skips invalid UTF-8 values", func(t *testing.T) {
		setupMigrateTest(t, map[string]string{
			"global.env": "GOOD=hello\n",
		}, nil)

		// Write a second env file with raw invalid UTF-8 bytes.
		cwd, _ := os.Getwd()
		badContent := append([]byte("VALID=ok\nBAD="), []byte{0xFF, 0xFE, 0x80}...)
		badContent = append(badContent, '\n')
		if err := os.WriteFile(
			filepath.Join(cwd, utils.EnvironmentFolder, "staging.env"),
			badContent, utils.FilePer,
		); err != nil {
			t.Fatal(err)
		}

		if err := runEnvMigrate(); err != nil {
			t.Fatalf("runEnvMigrate error: %v", err)
		}

		identity, _ := vault.ResolveIdentity()
		store, _ := vault.DecryptStore(identity)

		staging := store.GetEnv("staging")
		if staging == nil {
			t.Fatal("expected 'staging' section")
		}
		if staging["VALID"] != "ok" {
			t.Errorf("VALID = %v, want ok", staging["VALID"])
		}
		if _, exists := staging["BAD"]; exists {
			t.Error("BAD key should have been skipped (invalid UTF-8)")
		}
	})

	t.Run("preserves valid non-ASCII in migration", func(t *testing.T) {
		setupMigrateTest(t, map[string]string{
			"global.env": "NAME=José\nGREET=こんにちは\n",
		}, nil)

		if err := runEnvMigrate(); err != nil {
			t.Fatalf("runEnvMigrate error: %v", err)
		}

		identity, _ := vault.ResolveIdentity()
		store, _ := vault.DecryptStore(identity)

		env := store.GetEnv("global")
		if env["NAME"] != "José" {
			t.Errorf("NAME = %v, want José", env["NAME"])
		}
		if env["GREET"] != "こんにちは" {
			t.Errorf("GREET = %v, want こんにちは", env["GREET"])
		}
	})

	t.Run("errors when env dir missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpDir, _ = filepath.EvalSymlinks(tmpDir)
		configTmp := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", configTmp)
		oldWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(oldWd) })

		err := runEnvMigrate()
		if err == nil {
			t.Fatal("expected error when env/ missing")
		}
	})

	t.Run("errors when env is a file not directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpDir, _ = filepath.EvalSymlinks(tmpDir)
		envPath := filepath.Join(tmpDir, utils.EnvironmentFolder)
		if err := os.WriteFile(envPath, []byte("not a dir"), utils.FilePer); err != nil {
			t.Fatal(err)
		}
		configTmp := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", configTmp)
		oldWd, _ := os.Getwd()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chdir(oldWd) })

		err := runEnvMigrate()
		if err == nil {
			t.Fatal("expected error when env/ is a file")
		}
	})
}
