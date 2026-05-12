package userflags

import (
	"bytes"
	"crypto/ed25519"
	"encoding/pem"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
	"golang.org/x/crypto/ssh"
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

// ── SSH identity init tests ─────────────────────────────────────────────────

// writeTestSSHKey generates an unencrypted ed25519 SSH private key in dir,
// returns the file path and the public key in authorized_keys format.
func writeTestSSHKey(t *testing.T, dir string) (keyPath, pubKey string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("ed25519.GenerateKey: %v", err)
	}
	pemBlock, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("MarshalPrivateKey: %v", err)
	}
	keyPath = filepath.Join(dir, "id_ed25519")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(pemBlock), utils.SecretPer); err != nil {
		t.Fatalf("write key: %v", err)
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("NewPublicKey: %v", err)
	}
	pubKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))
	return keyPath, pubKey
}

func TestInitVaultProject_SSHIdentity_FreshSetup(t *testing.T) {
	dir := vaultTestSetup(t)

	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	keyPath, expectedPub := writeTestSSHKey(t, sshDir)

	if err := InitVaultProject(nil, keyPath); err != nil {
		t.Fatalf("InitVaultProject SSH: %v", err)
	}

	// .hulak/store.age exists
	storePath := filepath.Join(dir, utils.HiddenProjectName, utils.StoreFile)
	if !utils.FileExists(storePath) {
		t.Error("store.age not created")
	}

	// recipients.txt has SSH key, not age key
	recipientsPath := filepath.Join(dir, utils.HiddenProjectName, utils.RecipientsFile)
	data, err := os.ReadFile(recipientsPath)
	if err != nil {
		t.Fatalf("read recipients.txt: %v", err)
	}
	if !strings.Contains(string(data), expectedPub) {
		t.Errorf("recipients.txt should contain SSH pub key, got: %s", data)
	}
	if strings.Contains(string(data), "age1") {
		t.Error("recipients.txt should not contain an age key")
	}

	// identity.txt should NOT exist
	if vault.IdentityExists() {
		t.Error("identity.txt should not exist for SSH init")
	}

	// Store decrypts with SSH identity
	identity, err := vault.LoadSSHIdentity(keyPath)
	if err != nil {
		t.Fatalf("LoadSSHIdentity: %v", err)
	}
	store, err := vault.ReadStore(identity)
	if err != nil {
		t.Fatalf("ReadStore: %v", err)
	}
	if store.GetEnv(utils.DefaultEnvVal) == nil {
		t.Error("expected global section in store")
	}
}

func TestInitVaultProject_SSHIdentity_Idempotent(t *testing.T) {
	dir := vaultTestSetup(t)

	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	keyPath, _ := writeTestSSHKey(t, sshDir)

	if err := InitVaultProject(nil, keyPath); err != nil {
		t.Fatalf("first init: %v", err)
	}

	// Second init should not error
	if err := InitVaultProject(nil, keyPath); err != nil {
		t.Fatalf("second init: %v", err)
	}
}

func TestInitVaultProject_AgeInitThenAddSSH(t *testing.T) {
	dir := vaultTestSetup(t)

	// First init with age
	if err := InitVaultProject(nil, ""); err != nil {
		t.Fatalf("age init: %v", err)
	}

	// Second init with SSH — should add SSH as a recipient
	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	keyPath, expectedPub := writeTestSSHKey(t, sshDir)

	if err := InitVaultProject(nil, keyPath); err != nil {
		t.Fatalf("SSH additive init: %v", err)
	}

	// recipients.txt should have both age and SSH keys
	recipientsPath := filepath.Join(dir, utils.HiddenProjectName, utils.RecipientsFile)
	data, err := os.ReadFile(recipientsPath)
	if err != nil {
		t.Fatalf("read recipients.txt: %v", err)
	}
	if !strings.Contains(string(data), "age1") {
		t.Error("recipients.txt should still contain age key")
	}
	if !strings.Contains(string(data), expectedPub) {
		t.Error("recipients.txt should now contain SSH key")
	}

	// Store should decrypt with SSH identity
	sshIdentity, err := vault.LoadSSHIdentity(keyPath)
	if err != nil {
		t.Fatalf("LoadSSHIdentity: %v", err)
	}
	if _, err := vault.ReadStore(sshIdentity); err != nil {
		t.Fatalf("store should decrypt with SSH: %v", err)
	}
}

func TestInitVaultProject_SSHInitThenAddAge(t *testing.T) {
	dir := vaultTestSetup(t)

	// First init with SSH
	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	keyPath, _ := writeTestSSHKey(t, sshDir)

	if err := InitVaultProject(nil, keyPath); err != nil {
		t.Fatalf("SSH init: %v", err)
	}

	// Point ResolveIdentity at the test SSH key so it can decrypt for re-encryption
	t.Setenv(utils.SSHIdentityEnvVar, keyPath)

	// Second init without flags — should generate age key and add as recipient
	if err := InitVaultProject(nil, ""); err != nil {
		t.Fatalf("age additive init: %v", err)
	}

	// identity.txt should now exist
	if !vault.IdentityExists() {
		t.Error("identity.txt should exist after age additive init")
	}

	// recipients.txt should have both SSH and age keys
	recipientsPath := filepath.Join(dir, utils.HiddenProjectName, utils.RecipientsFile)
	data, err := os.ReadFile(recipientsPath)
	if err != nil {
		t.Fatalf("read recipients.txt: %v", err)
	}
	if !strings.Contains(string(data), "ssh-ed25519") {
		t.Error("recipients.txt should still contain SSH key")
	}
	if !strings.Contains(string(data), "age1") {
		t.Error("recipients.txt should now contain age key")
	}

	// Store should decrypt with age identity
	ageIdentity, err := vault.LoadIdentity()
	if err != nil {
		t.Fatalf("LoadIdentity: %v", err)
	}
	if _, err := vault.ReadStore(ageIdentity); err != nil {
		t.Fatalf("store should decrypt with age: %v", err)
	}
}

func TestInitVaultProject_SSHIdentity_WithEnvNames(t *testing.T) {
	dir := vaultTestSetup(t)

	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	keyPath, _ := writeTestSSHKey(t, sshDir)

	if err := InitVaultProject([]string{"staging", "prod"}, keyPath); err != nil {
		t.Fatalf("SSH init with envs: %v", err)
	}

	identity, err := vault.LoadSSHIdentity(keyPath)
	if err != nil {
		t.Fatalf("LoadSSHIdentity: %v", err)
	}
	store, err := vault.ReadStore(identity)
	if err != nil {
		t.Fatalf("ReadStore: %v", err)
	}

	for _, env := range []string{"global", "staging", "prod"} {
		if store.GetEnv(env) == nil {
			t.Errorf("expected %q section in store", env)
		}
	}
}

// TestInitVaultProject_SSHFlag_UsesDefaultPath tests the --ssh flag path
// where DefaultSSHIdentityPath is resolved automatically.
func TestInitVaultProject_SSHFlag_UsesDefaultPath(t *testing.T) {
	dir := vaultTestSetup(t)

	// Place the key at the default SSH path (~/.ssh/id_ed25519)
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	_, expectedPub := writeTestSSHKey(t, sshDir)

	// Simulate what newInitCmd does when --ssh is set: resolve default path
	sshPath := vault.DefaultSSHIdentityPath()
	if sshPath == "" {
		t.Fatal("DefaultSSHIdentityPath returned empty")
	}

	if err := InitVaultProject(nil, sshPath); err != nil {
		t.Fatalf("InitVaultProject with default SSH path: %v", err)
	}

	// Verify recipients.txt has the SSH key
	recipientsPath := filepath.Join(dir, utils.HiddenProjectName, utils.RecipientsFile)
	data, err := os.ReadFile(recipientsPath)
	if err != nil {
		t.Fatalf("read recipients.txt: %v", err)
	}
	if !strings.Contains(string(data), expectedPub) {
		t.Errorf("recipients.txt should contain SSH pub key from default path, got: %s", data)
	}
}

func TestInitVaultProject_ThirdRunIdempotent(t *testing.T) {
	dir := vaultTestSetup(t)

	// 1st: init with age
	if err := InitVaultProject(nil, ""); err != nil {
		t.Fatalf("age init: %v", err)
	}

	// 2nd: add SSH
	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, utils.DirPer); err != nil {
		t.Fatal(err)
	}
	keyPath, _ := writeTestSSHKey(t, sshDir)
	if err := InitVaultProject(nil, keyPath); err != nil {
		t.Fatalf("SSH additive init: %v", err)
	}

	// Snapshot recipients after both identities added
	recipientsPath := filepath.Join(dir, utils.HiddenProjectName, utils.RecipientsFile)
	before, err := os.ReadFile(recipientsPath)
	if err != nil {
		t.Fatal(err)
	}

	// 3rd: run again (no flags) — should be "Vault ready", no changes
	if err := InitVaultProject(nil, ""); err != nil {
		t.Fatalf("3rd init: %v", err)
	}

	after, err := os.ReadFile(recipientsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("recipients.txt changed on 3rd run — should be idempotent")
	}
}
