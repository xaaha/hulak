package gqlexplorer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils/testutil"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func TestReadRawParentFields_269_RejectsSplitFile(t *testing.T) {
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

func saveFromParent(t *testing.T, parent string) (*Model, string) {
	t.Helper()
	dir := t.TempDir()
	parentPath := filepath.Join(dir, "parent.hk.yaml")
	if err := os.WriteFile(parentPath, []byte(parent), 0o600); err != nil {
		t.Fatal(err)
	}
	m := &Model{
		filtered:        []UnifiedOperation{{Name: "getUser", Endpoint: "e"}},
		schemaFilePaths: map[string]string{"e": parentPath},
		apiInfos: map[string]yamlparser.APIInfo{"e": {
			URL:     "https://api.example.com/gql",
			Headers: map[string]string{"Authorization": "Bearer s3cret-token"},
		}},
	}
	if cmd := m.createHulakRequestFile(); cmd == nil {
		t.Fatal("createHulakRequestFile returned no notification")
	}
	return m, dir
}

func TestCreateHulakRequestFile_269_RefusesUnreadableParent(t *testing.T) {
	m, dir := saveFromParent(t, "kind: GraphQL\nurl: \"{{.base}}/gql\"\n---\nheaders:\n  Authorization: \"Bearer {{.token}}\"\n")
	want := "Save failed: " + testutil.SingleDocError(filepath.Join(dir, "parent.hk.yaml"), 3)
	if got := m.notification.CopyText(); got != want {
		t.Errorf("notification = %q, want %q", got, want)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "parent.hk.yaml" {
			t.Errorf("a refused save wrote %s", e.Name())
		}
	}
}

func TestCreateHulakRequestFile_269_KeepsTemplatesFromParent(t *testing.T) {
	_, dir := saveFromParent(t, "kind: GraphQL\nurl: \"{{.base}}/gql\"\nheaders:\n  Authorization: \"Bearer {{.token}}\"\n")
	saved, err := os.ReadFile(filepath.Join(dir, "getUser.hk.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(saved), "s3cret-token") || !strings.Contains(string(saved), "{{.token}}") {
		t.Errorf("saved request must keep the parent's templates:\n%s", saved)
	}
}
