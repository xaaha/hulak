package userflags

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

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
// init in a directory that already has .hulak/ refuses with an error,
// preventing two parallel sources of truth for env values.
func TestInitClassicProject_RefusesWhenVaultExists(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(chdirTemp(t, dir))

	// Pretend the vault is already initialized.
	if err := os.Mkdir(filepath.Join(dir, utils.HiddenProjectName), utils.DirPer); err != nil {
		t.Fatalf("create %s: %v", utils.HiddenProjectName, err)
	}

	err := InitClassicProject()
	if err == nil {
		t.Fatal("expected error when .hulak/ already exists, got nil")
	}

	// env/ must not have been created.
	envDir := filepath.Join(dir, utils.EnvironmentFolder)
	if utils.DirExists(envDir) {
		t.Errorf("env/ should not have been created when .hulak/ exists")
	}
}
