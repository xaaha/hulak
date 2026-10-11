package gqlexplorer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xaaha/hulak/pkg/utils/testutil"
)

func TestReadRawParentFields_RejectsSplitFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "parent.hk.yaml")
	content := "kind: GraphQL\nurl: \"{{.base}}/gql\"\n---\nheaders:\n  Authorization: \"Bearer {{.token}}\"\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := readRawParentFields(path)
	if err == nil || err.Error() != testutil.SingleDocError(path, 3) {
		t.Errorf("err = %v, want %q", err, testutil.SingleDocError(path, 3))
	}
	if raw.url != "" || raw.headers != nil {
		t.Errorf("a split file must yield no fields, got %+v", raw)
	}
}
