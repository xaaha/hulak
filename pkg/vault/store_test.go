package vault

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func TestStoreEnsureSection(t *testing.T) {
	t.Run("creates section on nil Envs map", func(t *testing.T) {
		s := &Store{}
		if !s.EnsureSection("global") {
			t.Error("EnsureSection on nil Envs should return true")
		}
		if s.GetEnv("global") == nil {
			t.Error("EnsureSection did not create section")
		}
	})

	t.Run("returns false when section already exists", func(t *testing.T) {
		s := &Store{Envs: map[string]Env{"prod": {"K": "v"}}}
		if s.EnsureSection("prod") {
			t.Error("EnsureSection on existing section should return false")
		}
		if s.Envs["prod"]["K"] != "v" {
			t.Error("EnsureSection clobbered existing section contents")
		}
	})

	t.Run("creates new section alongside existing", func(t *testing.T) {
		s := &Store{Envs: map[string]Env{"global": {"K": "v"}}}
		if !s.EnsureSection("staging") {
			t.Error("EnsureSection on new name should return true")
		}
		if s.GetEnv("staging") == nil || len(s.GetEnv("staging")) != 0 {
			t.Errorf("expected empty staging section, got %v", s.GetEnv("staging"))
		}
		if s.Envs["global"]["K"] != "v" {
			t.Error("EnsureSection disturbed unrelated section")
		}
	})
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
	got, err := DecryptStore(id)
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

func TestRoundTripNonASCIIAndHTMLChars(t *testing.T) {
	setupHulakProject(t)
	id, _ := age.GenerateX25519Identity()

	original := &Store{Envs: map[string]Env{
		"global": {
			"NAME":     "José",
			"GREET":    "こんにちは",
			"EMOJI":    "🚀",
			"HTML_URL": "https://example.com/api?a=1&b=2",
			"TAG":      "<script>alert('xss')</script>",
		},
	}}

	if err := WriteStore(original, id.Recipient()); err != nil {
		t.Fatalf("WriteStore error: %v", err)
	}

	got, err := DecryptStore(id)
	if err != nil {
		t.Fatalf("ReadStore error: %v", err)
	}

	env := got.GetEnv("global")
	cases := map[string]string{
		"NAME":     "José",
		"GREET":    "こんにちは",
		"EMOJI":    "🚀",
		"HTML_URL": "https://example.com/api?a=1&b=2",
		"TAG":      "<script>alert('xss')</script>",
	}
	for key, want := range cases {
		if env[key] != want {
			t.Errorf("%s = %v, want %s", key, env[key], want)
		}
	}
}

func TestWriteStoreNoHTMLEscaping(t *testing.T) {
	setupHulakProject(t)
	id, _ := age.GenerateX25519Identity()

	store := &Store{Envs: map[string]Env{
		"global": {
			"URL": "https://example.com?a=1&b=2",
			"TAG": "<div>hi</div>",
		},
	}}

	if err := WriteStore(store, id.Recipient()); err != nil {
		t.Fatalf("WriteStore error: %v", err)
	}

	// Read back and verify HTML chars survived un-escaped.
	got, err := DecryptStore(id)
	if err != nil {
		t.Fatalf("ReadStore error: %v", err)
	}

	env := got.GetEnv("global")
	if env["URL"] != "https://example.com?a=1&b=2" {
		t.Errorf("URL = %v, want literal &", env["URL"])
	}
	if env["TAG"] != "<div>hi</div>" {
		t.Errorf("TAG = %v, want literal < >", env["TAG"])
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

	got, err := DecryptStore(id)
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

	got, err := DecryptStore(id)
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

	_, err := DecryptStore(id2)
	if err == nil {
		t.Error("ReadStore() with wrong identity should return error")
	}
}

// TestReadStore_StaleIdentityFallsBackToSSH is the canonical integration
// test for the #222 onboarding fix. Scenario:
//
//   - A teammate has been added to a vault via their GitHub SSH key
//     (so SSH ed25519 is a recipient of store.age)
//   - The teammate's machine has a STALE identity.txt from another project
//     (an age key that is NOT a recipient of this vault)
//   - Pre-#222: ResolveIdentity short-circuits at identity.txt, never tries
//     SSH → decryption fails with a confusing error
//   - Post-#222: ResolveIdentityFor probes each source against the ciphertext,
//     falls through identity.txt → SSH, decrypts cleanly
//
// This exercises the full read path: ReadStore → ResolveIdentityFor →
// gatherIdentitySources → DecryptText, plus the announcement diagnostic.
func TestReadStore_StaleIdentityFallsBackToSSH(t *testing.T) {
	projectDir := setupHulakProject(t)

	// Isolate config dir
	cfgDir := t.TempDir()
	cfgDir, _ = filepath.EvalSymlinks(cfgDir)
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Setenv("HULAK_MASTER_KEY", "")
	t.Setenv("HOME", t.TempDir()) // empty home, no auto ~/.ssh/id_ed25519

	// 1. SSH key is the only recipient of the vault
	sshDir := t.TempDir()
	sshKeyPath, _ := writeTestSSHKey(t, sshDir)
	t.Setenv("HULAK_SSH_IDENTITY", sshKeyPath)

	sshPub, err := DeriveSSHPublicKey(sshKeyPath)
	if err != nil {
		t.Fatalf("DeriveSSHPublicKey: %v", err)
	}
	sshRecipient, _, err := ParseRecipientKey(sshPub, false)
	if err != nil {
		t.Fatalf("ParseRecipientKey: %v", err)
	}

	if err := SaveRecipients([]RecipientEntry{
		{Key: sshPub, Name: "alice-ssh"},
	}); err != nil {
		t.Fatalf("SaveRecipients: %v", err)
	}

	// 2. Write a vault encrypted to SSH only
	store := &Store{Envs: map[string]Env{
		"global": {"DATABASE_URL": "postgres://example"},
	}}
	if err := WriteStore(store, sshRecipient); err != nil {
		t.Fatalf("WriteStore: %v", err)
	}

	// 3. Plant a STALE identity.txt (age key that is NOT a recipient)
	staleID, _ := age.GenerateX25519Identity()
	if err := SetIdentity(staleID.String()); err != nil {
		t.Fatalf("SetIdentity (stale): %v", err)
	}

	// 4. Read via the auto-resolve path
	got, err := ReadStore()
	if err != nil {
		t.Fatalf(
			"ReadStore should succeed by falling through stale identity.txt to SSH: %v",
			err,
		)
	}

	// 5. Verify decryption succeeded with the right plaintext
	if got.GetEnv("global")["DATABASE_URL"] != "postgres://example" {
		t.Errorf("decrypted store data mismatch: %+v", got.GetEnv("global"))
	}

	// 6. Sanity: project dir was used
	if projectDir == "" {
		t.Error("project dir should be set")
	}
}

// TestReadStore_EnumeratesTriedSources verifies that when no available
// identity decrypts the store, the error lists every source that was tried —
// the diagnostic the review specifically called out as missing.
func TestReadStore_EnumeratesTriedSources(t *testing.T) {
	setupHulakProject(t)
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Setenv("HULAK_MASTER_KEY", "")
	t.Setenv("HULAK_SSH_IDENTITY", "")
	t.Setenv("HOME", t.TempDir())

	// Encrypt vault to a stranger key
	stranger, _ := age.GenerateX25519Identity()
	if err := SaveRecipients([]RecipientEntry{
		{Key: stranger.Recipient().String(), Name: "stranger"},
	}); err != nil {
		t.Fatalf("SaveRecipients: %v", err)
	}
	store := &Store{Envs: map[string]Env{"global": {"K": "v"}}}
	if err := WriteStore(store, stranger.Recipient()); err != nil {
		t.Fatalf("WriteStore: %v", err)
	}

	// Plant a useless identity.txt (not a recipient)
	useless, _ := age.GenerateX25519Identity()
	if err := SetIdentity(useless.String()); err != nil {
		t.Fatalf("SetIdentity: %v", err)
	}

	_, err := ReadStore()
	if err == nil {
		t.Fatal("expected error when no identity decrypts")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Tried:") {
		t.Errorf("error should enumerate tried sources: %v", err)
	}
	if !strings.Contains(msg, "identity.txt") {
		t.Errorf("error should mention identity.txt was tried: %v", err)
	}
	if !strings.Contains(msg, "add-recipient") {
		t.Errorf("error should suggest 'add-recipient' remediation: %v", err)
	}
}

func TestDetectStore(t *testing.T) {
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatal(err)
		}
	}()

	t.Run("returns StoreAge when store.age exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpDir, _ = filepath.EvalSymlinks(tmpDir)

		hulakDir := filepath.Join(tmpDir, utils.HiddenProjectName)
		if err := os.Mkdir(hulakDir, utils.DirPer); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(hulakDir, utils.StoreFile), []byte("encrypted"), utils.SecretPer); err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		if got := DetectStore(); got != StoreAge {
			t.Errorf("DetectStore() = %v, want StoreAge", got)
		}
	})

	t.Run("returns StoreClassic when only env/ exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpDir, _ = filepath.EvalSymlinks(tmpDir)

		if err := os.Mkdir(filepath.Join(tmpDir, utils.EnvironmentFolder), utils.DirPer); err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		if got := DetectStore(); got != StoreClassic {
			t.Errorf("DetectStore() = %v, want StoreClassic", got)
		}
	})

	t.Run("returns StoreNone when neither exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		if got := DetectStore(); got != StoreNone {
			t.Errorf("DetectStore() = %v, want StoreNone", got)
		}
	})

	t.Run("StoreAge takes priority over env/", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpDir, _ = filepath.EvalSymlinks(tmpDir)

		hulakDir := filepath.Join(tmpDir, utils.HiddenProjectName)
		if err := os.Mkdir(hulakDir, utils.DirPer); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(hulakDir, utils.StoreFile), []byte("encrypted"), utils.SecretPer); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(tmpDir, utils.EnvironmentFolder), utils.DirPer); err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		if got := DetectStore(); got != StoreAge {
			t.Errorf("DetectStore() = %v, want StoreAge when both exist", got)
		}
	})

	t.Run("returns StoreNone when .hulak is dir but no store.age", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpDir, _ = filepath.EvalSymlinks(tmpDir)

		if err := os.Mkdir(filepath.Join(tmpDir, utils.HiddenProjectName), utils.DirPer); err != nil {
			t.Fatal(err)
		}
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		if got := DetectStore(); got != StoreNone {
			t.Errorf("DetectStore() = %v, want StoreNone when .hulak has no store.age", got)
		}
	})
}

func TestWriteStoreEmbedsVersion(t *testing.T) {
	setupHulakProject(t)
	id, _ := age.GenerateX25519Identity()

	s := &Store{Envs: map[string]Env{"global": {"KEY": "value"}}}
	if err := WriteStore(s, id.Recipient()); err != nil {
		t.Fatalf("WriteStore() error: %v", err)
	}

	// Decrypt and inspect the raw JSON to confirm _version is present.
	cipher, err := os.ReadFile(filepath.Join(utils.HiddenProjectName, utils.StoreFile))
	if err != nil {
		t.Fatalf("read store: %v", err)
	}
	plain, err := DecryptText(cipher, id)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(plain, &raw); err != nil {
		t.Fatalf("parse plain: %v", err)
	}

	vRaw, ok := raw["_version"]
	if !ok {
		t.Fatal("_version field missing from written store")
	}
	var got int
	if err := json.Unmarshal(vRaw, &got); err != nil {
		t.Fatalf("parse _version: %v", err)
	}
	if got != StoreVersion {
		t.Errorf("_version = %d, want %d", got, StoreVersion)
	}
}

func TestReadStoreLegacyWithoutVersion(t *testing.T) {
	setupHulakProject(t)
	id, _ := age.GenerateX25519Identity()

	// Legacy store: pure env map, no _version key.
	legacy, err := json.Marshal(map[string]Env{
		"global": {"BASE_URL": "https://api.example.com"},
	})
	if err != nil {
		t.Fatalf("marshal legacy: %v", err)
	}
	cipher, err := EncryptText(legacy, id.Recipient())
	if err != nil {
		t.Fatalf("encrypt legacy: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(utils.HiddenProjectName, utils.StoreFile), cipher, utils.SecretPer,
	); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	got, err := DecryptStore(id)
	if err != nil {
		t.Fatalf("ReadStore() legacy error: %v", err)
	}
	if got.GetEnv("global")["BASE_URL"] != "https://api.example.com" {
		t.Errorf("legacy global.BASE_URL = %v, want %q", got.GetEnv("global")["BASE_URL"], "https://api.example.com")
	}
}

func TestReadStoreFutureVersionRejected(t *testing.T) {
	setupHulakProject(t)
	id, _ := age.GenerateX25519Identity()

	future, err := json.Marshal(map[string]any{
		"_version": StoreVersion + 1,
		"global":   map[string]any{"KEY": "value"},
	})
	if err != nil {
		t.Fatalf("marshal future: %v", err)
	}
	cipher, err := EncryptText(future, id.Recipient())
	if err != nil {
		t.Fatalf("encrypt future: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(utils.HiddenProjectName, utils.StoreFile), cipher, utils.SecretPer,
	); err != nil {
		t.Fatalf("write future: %v", err)
	}

	_, err = DecryptStore(id)
	if err == nil {
		t.Fatal("ReadStore() with future version should error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "newer hulak") || !strings.Contains(msg, "upgrade") {
		t.Errorf("error message %q should mention 'newer hulak' and 'upgrade'", msg)
	}
}

// captureStderr swaps os.Stderr for a pipe, runs fn, restores os.Stderr,
// and returns whatever fn wrote to stderr.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = orig })

	done := make(chan string, 1)
	go func() {
		var buf strings.Builder
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}

func TestReadStoreSizeWarning_LargeTriggersOnce(t *testing.T) {
	setupHulakProject(t)
	id, _ := age.GenerateX25519Identity()

	// Build a store whose decrypted JSON exceeds 1 MB.
	big := strings.Repeat("a", MaxStoreSizeWarnBytes+1024)
	original := &Store{Envs: map[string]Env{"global": {"BLOB": big}}}
	if err := WriteStore(original, id.Recipient()); err != nil {
		t.Fatalf("WriteStore: %v", err)
	}

	// Reset the once-gate so this test sees the warning regardless of order.
	storeSizeWarnOnce = sync.Once{}

	out := captureStderr(t, func() {
		if _, err := DecryptStore(id); err != nil {
			t.Fatalf("ReadStore #1: %v", err)
		}
		if _, err := DecryptStore(id); err != nil {
			t.Fatalf("ReadStore #2: %v", err)
		}
	})

	if !strings.Contains(out, "warning") || !strings.Contains(out, "MB") {
		t.Errorf("expected size warning in stderr, got %q", out)
	}
	if got := strings.Count(out, "warning"); got != 1 {
		t.Errorf("warning fired %d times, want 1 (once per process)", got)
	}
}

func TestReadStoreSizeWarning_SmallNoWarning(t *testing.T) {
	setupHulakProject(t)
	id, _ := age.GenerateX25519Identity()

	original := &Store{Envs: map[string]Env{"global": {"KEY": "value"}}}
	if err := WriteStore(original, id.Recipient()); err != nil {
		t.Fatalf("WriteStore: %v", err)
	}

	storeSizeWarnOnce = sync.Once{}

	out := captureStderr(t, func() {
		if _, err := DecryptStore(id); err != nil {
			t.Fatalf("ReadStore: %v", err)
		}
	})

	if out != "" {
		t.Errorf("expected no stderr output for small store, got %q", out)
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

func TestReadStoreFrom(t *testing.T) {
	cfg := setupConfigDir(t)
	id, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "identity.txt"), []byte(id.String()+"\n"), 0o600); err != nil {
		t.Fatalf("write identity: %v", err)
	}

	store := &Store{Envs: map[string]Env{
		"global": {"SECRET": "hello"},
	}}
	jsonData, err := json.Marshal(store)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	cipher, err := EncryptText(jsonData, id.Recipient())
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	backupPath := filepath.Join(t.TempDir(), "backup.age")
	if err := os.WriteFile(backupPath, cipher, 0o600); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	got, err := ReadStore(backupPath)
	if err != nil {
		t.Fatalf("ReadStore(backup): %v", err)
	}
	if v, ok := got.GetEnv("global")["SECRET"].(string); !ok || v != "hello" {
		t.Errorf("got SECRET=%v, want %q", got.GetEnv("global")["SECRET"], "hello")
	}
}

func TestReadStoreFrom_MissingFile(t *testing.T) {
	setupConfigDir(t)
	_, err := ReadStore("/nonexistent/path.age")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadStoreFrom_WrongIdentity(t *testing.T) {
	cfg := setupConfigDir(t)
	id1, _ := age.GenerateX25519Identity()
	id2, _ := age.GenerateX25519Identity()
	// Only id2 is configured locally — won't decrypt cipher encrypted to id1.
	if err := os.WriteFile(filepath.Join(cfg, "identity.txt"), []byte(id2.String()+"\n"), 0o600); err != nil {
		t.Fatalf("write identity: %v", err)
	}

	store := &Store{Envs: map[string]Env{"global": {"K": "V"}}}
	jsonData, _ := json.Marshal(store)
	cipher, _ := EncryptText(jsonData, id1.Recipient())

	path := filepath.Join(t.TempDir(), "backup.age")
	if err := os.WriteFile(path, cipher, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := ReadStore(path)
	if err == nil {
		t.Fatal("expected error when no configured identity decrypts")
	}
}
