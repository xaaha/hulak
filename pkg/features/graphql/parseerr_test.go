package graphql

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindGraphQLFiles_278_KeepsGraphQLFilesThatFailToParse(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"stray.hk.yaml":  "kind: GraphQL\nurl: https://a.example/gql\n// stray\n",
		"repeat.hk.yaml": "kind: GraphQL\nurl: \"https://b.example/gql\"\nurl: https://b.example/other\n",
		"tab.hk.yaml":    "kind: graphql\nurl: '{{.base}}/gql'\nheaders:\n\tX: b\n",
		"compose.yaml":   "services:\n\tweb: x\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	found, needsEnv, err := FindGraphQLFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"https://a.example/gql": filepath.Join(dir, "stray.hk.yaml"),
		"https://b.example/gql": filepath.Join(dir, "repeat.hk.yaml"),
		"{{.base}}/gql":         filepath.Join(dir, "tab.hk.yaml"),
	}
	if len(found) != len(want) {
		t.Fatalf("found %v, want %v", found, want)
	}
	for url, path := range want {
		if found[url] != path {
			t.Errorf("found[%q] = %q, want %q", url, found[url], path)
		}
	}
	if needsEnv || NeedsEnvResolution(found) {
		t.Error("files that fail to parse must not ask for an environment")
	}
	for _, r := range ProcessFilesConcurrent([]string{want["https://a.example/gql"], want["https://b.example/gql"], want["{{.base}}/gql"]}, nil) {
		if r.Error == nil || !strings.Contains(r.Error.Error(), r.FilePath+": [") {
			t.Errorf("%s: processing must report the parse error with its line, got %v", r.FilePath, r.Error)
		}
	}
}
