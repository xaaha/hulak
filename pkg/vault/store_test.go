package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

func TestStoreGetEnv(t *testing.T) {
	s := &Store{
		Envs: map[string]Env{
			"global": {"BASE_URL": "https://api.example.com"},
			"prod":   {"API_KEY": "sk-prod-xxx"},
		},
	}

	tests := []struct {
		name    string
		envName string
		wantNil bool
		wantKey string
		wantVal any
	}{
		{"existing env", "global", false, "BASE_URL", "https://api.example.com"},
		{"another env", "prod", false, "API_KEY", "sk-prod-xxx"},
		{"nonexistent env", "staging", true, "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.GetEnv(tt.envName)
			if tt.wantNil {
				if got != nil {
					t.Errorf("GetEnv(%q) = %v, want nil", tt.envName, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("GetEnv(%q) = nil, want non-nil", tt.envName)
			}
			if got[tt.wantKey] != tt.wantVal {
				t.Errorf("GetEnv(%q)[%q] = %v, want %v", tt.envName, tt.wantKey, got[tt.wantKey], tt.wantVal)
			}
		})
	}
}

func TestStoreListEnvs(t *testing.T) {
	tests := []struct {
		name string
		envs map[string]Env
		want []string
	}{
		{
			"multiple envs sorted",
			map[string]Env{"prod": {}, "global": {}, "staging": {}},
			[]string{"global", "prod", "staging"},
		},
		{
			"single env",
			map[string]Env{"global": {}},
			[]string{"global"},
		},
		{
			"empty store",
			map[string]Env{},
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Store{Envs: tt.envs}
			got := s.ListEnvs()
			if len(got) != len(tt.want) {
				t.Fatalf("ListEnvs() len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ListEnvs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestStoreSetKey(t *testing.T) {
	t.Run("set key in new env", func(t *testing.T) {
		s := &Store{Envs: make(map[string]Env)}
		s.SetKey("prod", "API_KEY", "sk-prod-xxx")

		got := s.GetEnv("prod")
		if got == nil {
			t.Fatal("SetKey() did not create environment")
		}
		if got["API_KEY"] != "sk-prod-xxx" {
			t.Errorf("SetKey() value = %v, want %q", got["API_KEY"], "sk-prod-xxx")
		}
	})

	t.Run("upsert existing key", func(t *testing.T) {
		s := &Store{Envs: map[string]Env{
			"prod": {"API_KEY": "old"},
		}}
		s.SetKey("prod", "API_KEY", "new")

		if s.Envs["prod"]["API_KEY"] != "new" {
			t.Errorf("SetKey() upsert failed: got %v, want %q", s.Envs["prod"]["API_KEY"], "new")
		}
	})

	t.Run("set key on nil envs map", func(t *testing.T) {
		s := &Store{}
		s.SetKey("global", "URL", "https://example.com")

		if s.Envs["global"]["URL"] != "https://example.com" {
			t.Errorf("SetKey() on nil Envs failed")
		}
	})

	t.Run("preserves different value types", func(t *testing.T) {
		s := &Store{Envs: make(map[string]Env)}
		s.SetKey("global", "str", "hello")
		s.SetKey("global", "num", json.Number("8000"))
		s.SetKey("global", "flag", true)

		if s.Envs["global"]["str"] != "hello" {
			t.Error("string value not preserved")
		}
		if s.Envs["global"]["num"] != json.Number("8000") {
			t.Error("json.Number value not preserved")
		}
		if s.Envs["global"]["flag"] != true {
			t.Error("bool value not preserved")
		}
	})
}

func TestStoreDeleteKey(t *testing.T) {
	t.Run("delete existing key", func(t *testing.T) {
		s := &Store{Envs: map[string]Env{
			"prod": {"API_KEY": "sk-xxx", "URL": "https://example.com"},
		}}
		s.DeleteKey("prod", "API_KEY")

		if _, exists := s.Envs["prod"]["API_KEY"]; exists {
			t.Error("DeleteKey() did not remove key")
		}
		if s.Envs["prod"]["URL"] != "https://example.com" {
			t.Error("DeleteKey() removed wrong key")
		}
	})

	t.Run("delete from nonexistent env is no-op", func(_ *testing.T) {
		s := &Store{Envs: make(map[string]Env)}
		s.DeleteKey("nonexistent", "KEY")
		// should not panic
	})

	t.Run("delete nonexistent key is no-op", func(t *testing.T) {
		s := &Store{Envs: map[string]Env{
			"prod": {"API_KEY": "sk-xxx"},
		}}
		s.DeleteKey("prod", "NOPE")

		if s.Envs["prod"]["API_KEY"] != "sk-xxx" {
			t.Error("DeleteKey() of nonexistent key affected other keys")
		}
	})
}

// setupHulakProject creates a temp directory with env/ and .hulak/ to simulate
// a hulak project, changes into it, and returns the directory path.
func setupHulakProject(t *testing.T) string {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}

	if err := os.Mkdir(filepath.Join(tmpDir, utils.EnvironmentFolder), utils.DirPer); err != nil {
		t.Fatalf("failed to create env dir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, utils.HiddenProjectName), utils.DirPer); err != nil {
		t.Fatalf("failed to create .hulak dir: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	})

	return tmpDir
}

func TestReadStoreWriteStoreRoundTrip(t *testing.T) {
	projectDir := setupHulakProject(t)
	id, _ := age.GenerateX25519Identity()

	// Write a store
	original := &Store{Envs: map[string]Env{
		"global": {"BASE_URL": "https://api.example.com", "DEBUG": true},
		"prod":   {"API_KEY": "sk-prod-xxx"},
	}}

	if err := WriteStore(original, id.Recipient()); err != nil {
		t.Fatalf("WriteStore() error: %v", err)
	}

	// Verify store.age was created
	storePath := filepath.Join(projectDir, utils.HiddenProjectName, utils.StoreFile)
	if _, err := os.Stat(storePath); err != nil {
		t.Fatalf("store.age not created: %v", err)
	}

	// Read it back
	got, err := ReadStore(id)
	if err != nil {
		t.Fatalf("ReadStore() error: %v", err)
	}

	// Verify environments
	if len(got.Envs) != 2 {
		t.Fatalf("ReadStore() envs count = %d, want 2", len(got.Envs))
	}

	globalEnv := got.GetEnv("global")
	if globalEnv == nil {
		t.Fatal("ReadStore() missing global env")
	}
	if globalEnv["BASE_URL"] != "https://api.example.com" {
		t.Errorf("global.BASE_URL = %v, want %q", globalEnv["BASE_URL"], "https://api.example.com")
	}
	if globalEnv["DEBUG"] != true {
		t.Errorf("global.DEBUG = %v, want true", globalEnv["DEBUG"])
	}

	prodEnv := got.GetEnv("prod")
	if prodEnv == nil {
		t.Fatal("ReadStore() missing prod env")
	}
	if prodEnv["API_KEY"] != "sk-prod-xxx" {
		t.Errorf("prod.API_KEY = %v, want %q", prodEnv["API_KEY"], "sk-prod-xxx")
	}
}

func TestReadStorePreservesNumberTypes(t *testing.T) {
	setupHulakProject(t)
	id, _ := age.GenerateX25519Identity()

	original := &Store{Envs: map[string]Env{
		"global": {
			"PORT":    json.Number("8000"),
			"TIMEOUT": json.Number("30"),
			"RATE":    json.Number("3.14"),
		},
	}}

	if err := WriteStore(original, id.Recipient()); err != nil {
		t.Fatalf("WriteStore() error: %v", err)
	}

	got, err := ReadStore(id)
	if err != nil {
		t.Fatalf("ReadStore() error: %v", err)
	}

	globalEnv := got.GetEnv("global")

	// UseNumber() should preserve numbers as json.Number
	port, ok := globalEnv["PORT"].(json.Number)
	if !ok {
		t.Fatalf("PORT type = %T, want json.Number", globalEnv["PORT"])
	}
	if port.String() != "8000" {
		t.Errorf("PORT = %q, want %q", port.String(), "8000")
	}

	rate, ok := globalEnv["RATE"].(json.Number)
	if !ok {
		t.Fatalf("RATE type = %T, want json.Number", globalEnv["RATE"])
	}
	if rate.String() != "3.14" {
		t.Errorf("RATE = %q, want %q", rate.String(), "3.14")
	}
}

func TestReadStoreNonexistentReturnsEmpty(t *testing.T) {
	setupHulakProject(t)
	id, _ := age.GenerateX25519Identity()

	got, err := ReadStore(id)
	if err != nil {
		t.Fatalf("ReadStore() on missing file error: %v", err)
	}

	if got.Envs == nil {
		t.Fatal("ReadStore() on missing file returned nil Envs")
	}
	if len(got.Envs) != 0 {
		t.Errorf("ReadStore() on missing file envs count = %d, want 0", len(got.Envs))
	}
}

func TestReadStoreWrongIdentity(t *testing.T) {
	setupHulakProject(t)
	id1, _ := age.GenerateX25519Identity()
	id2, _ := age.GenerateX25519Identity()

	original := &Store{Envs: map[string]Env{
		"global": {"KEY": "value"},
	}}

	if err := WriteStore(original, id1.Recipient()); err != nil {
		t.Fatalf("WriteStore() error: %v", err)
	}

	_, err := ReadStore(id2)
	if err == nil {
		t.Error("ReadStore() with wrong identity should return error")
	}
}

func TestWriteStoreAtomicCleanup(t *testing.T) {
	projectDir := setupHulakProject(t)
	id, _ := age.GenerateX25519Identity()

	s := &Store{Envs: map[string]Env{"global": {"KEY": "value"}}}

	if err := WriteStore(s, id.Recipient()); err != nil {
		t.Fatalf("WriteStore() error: %v", err)
	}

	// .tmp file should not exist after successful write
	tmpPath := filepath.Join(projectDir, utils.HiddenProjectName, utils.StoreFile+".tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("WriteStore() left behind .tmp file")
	}
}
