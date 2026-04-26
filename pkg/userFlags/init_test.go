package userflags

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

// TestInitDefaultProject_PreservesUserCustomizedAPIOptions verifies that
// re-running `hulak init` does NOT overwrite a user-edited apiOptions.hk.yaml.
// Init is designed to be safe to re-run; clobbering customizations would
// defeat that property.
func TestInitDefaultProject_PreservesUserCustomizedAPIOptions(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(chdirTemp(t, dir))

	// First init: creates the example file.
	if err := InitDefaultProject(); err != nil {
		t.Fatalf("first InitDefaultProject: %v", err)
	}

	apiPath := filepath.Join(dir, utils.APIOptions)
	custom := []byte("# user has edited this file\nkind: API\nfoo: bar\n")
	if err := os.WriteFile(apiPath, custom, utils.FilePer); err != nil {
		t.Fatalf("simulate user edit: %v", err)
	}

	// Second init: must not clobber the custom content.
	if err := InitDefaultProject(); err != nil {
		t.Fatalf("second InitDefaultProject: %v", err)
	}

	got, err := os.ReadFile(apiPath)
	if err != nil {
		t.Fatalf("read after re-init: %v", err)
	}
	if !bytes.Equal(got, custom) {
		t.Errorf("re-init overwrote customized %s", utils.APIOptions)
	}
}
