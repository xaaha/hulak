package gqlexplorer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/features/graphql"
)

func TestCreateHulakRequestFile_278_D3_EmptyHeaderValueStaysEmpty(t *testing.T) {
	m, dir := saveFromParent(t, "kind: GraphQL\nurl: https://api.example.com/gql\nheaders:\n  X-Foo:\n  X-Bar: b\n")
	saved := filepath.Join(dir, "getUser.hk.yaml")
	content, err := os.ReadFile(saved)
	if err != nil {
		t.Fatalf("no request saved: %v (notification %q)", err, m.notification.CopyText())
	}
	if strings.Contains(string(content), "<nil>") {
		t.Fatalf("saved request holds <nil>:\n%s", content)
	}
	t.Chdir(dir)
	if err := os.Mkdir(filepath.Join(dir, "env"), 0o755); err != nil {
		t.Fatal(err)
	}
	info, err := graphql.ProcessGraphQLFile(saved, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := info.Headers["x-foo"]; !ok || v != "" {
		t.Errorf("x-foo = %q (present %v), want an empty value; headers %v", v, ok, info.Headers)
	}
}
