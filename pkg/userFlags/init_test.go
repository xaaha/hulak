package userflags

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// vaultTestSetup chdirs into a fresh project root and points
// $XDG_CONFIG_HOME at a separate tmpdir so the user's real identity file is
// never touched. Returns the project dir for follow-up assertions.
func vaultTestSetup(t *testing.T) string {
	t.Helper()
	projectDir := t.TempDir()
	cfgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Cleanup(chdirTemp(t, projectDir))
	return projectDir
}

// readVaultStore decrypts and returns the store at .hulak/store.age in the
// current working directory. Tests use this to assert post-init state.
func readVaultStore(t *testing.T) *vault.Store {
	t.Helper()
	identity, err := vault.LoadIdentity()
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
	store, err := vault.ReadStore(identity)
	if err != nil {
		t.Fatalf("ReadStore: %v", err)
	}
	return store
}

// TestInitClassicProject_PreservesUserCustomizedAPIOptions verifies that
// re-running `hulak init classic` does NOT overwrite a user-edited
// apiOptions.hk.yaml. Init is designed to be safe to re-run; clobbering
// customizations would defeat that property.
func TestInitClassicProject_PreservesUserCustomizedAPIOptions(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(chdirTemp(t, dir))

	// First init: creates the example file.
	if err := InitClassicProject(); err != nil {
		t.Fatalf("first InitClassicProject: %v", err)
	}

	apiPath := filepath.Join(dir, utils.APIOptions)
	custom := []byte("# user has edited this file\nkind: API\nfoo: bar\n")
	if err := os.WriteFile(apiPath, custom, utils.FilePer); err != nil {
		t.Fatalf("simulate user edit: %v", err)
	}

	// Second init: must not clobber the custom content.
	if err := InitClassicProject(); err != nil {
		t.Fatalf("second InitClassicProject: %v", err)
	}

	got, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("read after re-init: %v", err)
	}
	if !bytes.Equal(got, custom) {
		t.Errorf("re-init overwrote customized %s", utils.APIOptions)
	}
}

// TestInitClassicProject_RefusesWhenVaultExists verifies that running classic
// init in a directory with an initialized vault (store.age present) refuses
// with an error, preventing two parallel sources of truth for env values.
func TestInitClassicProject_RefusesWhenVaultExists(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(chdirTemp(t, dir))

	hulakDir := filepath.Join(dir, utils.HiddenProjectName)
	if err := os.Mkdir(hulakDir, utils.DirPer); err != nil {
		t.Fatalf("create %s: %v", utils.HiddenProjectName, err)
	}
	// store.age presence — not just the .hulak/ dir — is what marks the
	// vault as initialized. An empty .hulak/ (e.g. from a partially failed
	// vault init) must not lock the user out of the classic path.
	if err := os.WriteFile(
		filepath.Join(hulakDir, utils.StoreFile), []byte("encrypted"), utils.SecretPer,
	); err != nil {
		t.Fatalf("create %s: %v", utils.StoreFile, err)
	}

	err := InitClassicProject()
	if err == nil {
		t.Fatal("expected error when store.age exists, got nil")
	}

	envDir := filepath.Join(dir, utils.EnvironmentFolder)
	if utils.DirExists(envDir) {
		t.Errorf("env/ should not have been created when store.age exists")
	}
}

// TestInitClassicProject_AllowsEmptyHulakDir verifies that an empty .hulak/
// (e.g. left behind by a partially-failed vault init) does NOT block classic
// init. Only an actual store.age signals "vault initialized."
func TestInitClassicProject_AllowsEmptyHulakDir(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(chdirTemp(t, dir))

	if err := os.Mkdir(filepath.Join(dir, utils.HiddenProjectName), utils.DirPer); err != nil {
		t.Fatalf("create %s: %v", utils.HiddenProjectName, err)
	}

	if err := InitClassicProject(); err != nil {
		t.Fatalf("InitClassicProject should succeed when .hulak/ is empty, got: %v", err)
	}

	if !utils.DirExists(filepath.Join(dir, utils.EnvironmentFolder)) {
		t.Error("env/ should have been created")
	}
}

// TestInitVaultProject_FreshSetup verifies the happy path: empty directory
// gets .hulak/store.age, .hulak/recipients.txt, an identity file under the
// test XDG dir, an apiOptions example, and a decryptable store containing
// only the implicit "global" section.
func TestInitVaultProject_FreshSetup(t *testing.T) {
	dir := vaultTestSetup(t)

	if err := InitVaultProject(nil, ""); err != nil {
		t.Fatalf("InitVaultProject: %v", err)
	}

	// .hulak/ + store.age + recipients.txt
	wantFiles := []string{
		filepath.Join(dir, utils.HiddenProjectName, utils.StoreFile),
		filepath.Join(dir, utils.HiddenProjectName, utils.RecipientsFile),
		filepath.Join(dir, utils.APIOptions),
	}
	for _, f := range wantFiles {
		if !utils.FileExists(f) {
			t.Errorf("expected %s to exist", f)
		}
	}

	// Identity must land in XDG_CONFIG_HOME/hulak (not the user's real config)
	if !vault.IdentityExists() {
		t.Error("expected identity file to exist in XDG_CONFIG_HOME")
	}

	// Store must decrypt and contain exactly { global: {} }
	store := readVaultStore(t)
	if len(store.Envs) != 1 {
		t.Errorf("expected 1 env section, got %d: %v", len(store.Envs), store.ListEnvs())
	}
	if store.GetEnv(utils.DefaultEnvVal) == nil {
		t.Errorf("expected %q section to exist", utils.DefaultEnvVal)
	}
}

// TestInitVaultProject_Idempotent verifies that re-running init on an already
// initialized vault does not regenerate the identity, does not overwrite
// existing store contents, and does not clobber a customized apiOptions.
func TestInitVaultProject_Idempotent(t *testing.T) {
	dir := vaultTestSetup(t)

	if err := InitVaultProject(nil, ""); err != nil {
		t.Fatalf("first InitVaultProject: %v", err)
	}

	identityPath, err := vault.IdentityPath()
	if err != nil {
		t.Fatalf("IdentityPath: %v", err)
	}
	identityBefore, err := os.ReadFile(identityPath)
	if err != nil {
		t.Fatalf("read identity: %v", err)
	}

	// Customize apiOptions to verify it survives re-init.
	apiPath := filepath.Join(dir, utils.APIOptions)
	custom := []byte("# custom edits\nkind: API\n")
	if err := os.WriteFile(apiPath, custom, utils.FilePer); err != nil {
		t.Fatalf("simulate user edit: %v", err)
	}

	// Set a value in the vault, so we can also verify the store wasn't reset.
	identity, err := vault.LoadIdentity()
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
	store, err := vault.ReadStore(identity)
	if err != nil {
		t.Fatalf("ReadStore: %v", err)
	}
	store.SetKey(utils.DefaultEnvVal, "FOO", "bar")
	if err := vault.WriteStore(store, identity.Recipient()); err != nil {
		t.Fatalf("WriteStore: %v", err)
	}

	// Second init: must not regenerate identity, must not lose the FOO key.
	if err := InitVaultProject(nil, ""); err != nil {
		t.Fatalf("second InitVaultProject: %v", err)
	}

	identityAfter, err := os.ReadFile(identityPath)
	if err != nil {
		t.Fatalf("re-read identity: %v", err)
	}
	if !bytes.Equal(identityBefore, identityAfter) {
		t.Error("identity file changed across re-init — must be idempotent")
	}

	got, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("re-read apiOptions: %v", err)
	}
	if !bytes.Equal(got, custom) {
		t.Error("re-init clobbered customized apiOptions.hk.yaml")
	}

	storeAfter := readVaultStore(t)
	env := storeAfter.GetEnv(utils.DefaultEnvVal)
	if env == nil || env["FOO"] != "bar" {
		t.Errorf("re-init lost previously-set value: env=%v", env)
	}
}

// TestInitVaultProject_WithExtraEnvs verifies that -env arg names land as
// empty sections in the store alongside the implicit "global".
func TestInitVaultProject_WithExtraEnvs(t *testing.T) {
	vaultTestSetup(t)

	if err := InitVaultProject([]string{"staging", "prod"}, ""); err != nil {
		t.Fatalf("InitVaultProject: %v", err)
	}

	store := readVaultStore(t)
	got := store.ListEnvs() // sorted
	want := []string{utils.DefaultEnvVal, "prod", "staging"}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("ListEnvs() = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("ListEnvs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	for _, name := range []string{"staging", "prod"} {
		env := store.GetEnv(name)
		if env == nil {
			t.Errorf("expected %q section to exist", name)
			continue
		}
		if len(env) != 0 {
			t.Errorf("expected %q to be empty, got %d keys", name, len(env))
		}
	}
}

// TestInitVaultProject_LegacyEnvNudge verifies that init in a directory with
// pre-existing env/ but no .hulak/ returns nil without creating .hulak/, so
// the user can run `hulak secrets migrate` deliberately.
func TestInitVaultProject_LegacyEnvNudge(t *testing.T) {
	dir := vaultTestSetup(t)

	// Pre-create env/ to simulate a legacy classic project.
	envDir := filepath.Join(dir, utils.EnvironmentFolder)
	if err := os.Mkdir(envDir, utils.DirPer); err != nil {
		t.Fatalf("mkdir env/: %v", err)
	}

	if err := InitVaultProject(nil, ""); err != nil {
		t.Fatalf("expected nil err on legacy nudge path, got: %v", err)
	}

	hulakDir := filepath.Join(dir, utils.HiddenProjectName)
	if utils.DirExists(hulakDir) {
		t.Error(".hulak/ should not have been created when env/ exists")
	}
	if vault.IdentityExists() {
		t.Error("identity should not have been generated when nudging")
	}
}

// TestInitVaultProject_RejectsInvalidEnvName verifies that an invalid env
// name aborts before any filesystem mutation — no .hulak/, no identity.
func TestInitVaultProject_RejectsInvalidEnvName(t *testing.T) {
	dir := vaultTestSetup(t)

	err := InitVaultProject([]string{"good_name", "bad name!"}, "")
	if err == nil {
		t.Fatal("expected error for invalid env name, got nil")
	}

	hulakDir := filepath.Join(dir, utils.HiddenProjectName)
	if utils.DirExists(hulakDir) {
		t.Error(".hulak/ should not have been created when validation fails")
	}
	if vault.IdentityExists() {
		t.Error("identity should not have been generated when validation fails")
	}
}

// TestInitVaultProject_DedupesGlobal verifies that passing "Global" (any
// case) in -env args does not create a duplicate section — the
// case-insensitive match folds it into the implicit "global".
func TestInitVaultProject_DedupesGlobal(t *testing.T) {
	vaultTestSetup(t)

	if err := InitVaultProject([]string{"Global", "GLOBAL"}, ""); err != nil {
		t.Fatalf("InitVaultProject: %v", err)
	}

	store := readVaultStore(t)
	if len(store.Envs) != 1 {
		t.Errorf(
			"expected exactly 1 section after dedup, got %d: %v",
			len(store.Envs), store.ListEnvs(),
		)
	}
	if store.GetEnv(utils.DefaultEnvVal) == nil {
		t.Errorf("expected canonical %q section", utils.DefaultEnvVal)
	}
}
