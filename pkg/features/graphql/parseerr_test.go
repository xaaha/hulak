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

func TestFindGraphQLFiles_278_TextFallbackEdges(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"dq.hk.yaml":      "kind: \"GraphQL\"\nurl: https://dq.example/gql\n// stray\n",
		"sq.hk.yaml":      "kind: 'graphql'\nurl: https://sq.example/gql\n// stray\n",
		"comment.hk.yaml": "kind: GraphQL # c\nurl: https://cm.example/gql # note\n// stray\n",
		"crlf.hk.yaml":    "kind: GraphQL\r\nurl: https://crlf.example/gql\r\n// stray\r\n",
		"indent.hk.yaml":  "kind: GraphQL\nbody:\n  url: https://nested.example/x\nurl: https://top.example/gql\n// stray\n",
		"api.hk.yaml":     "kind: API\nurl: https://api.example/x\n// stray\n",
		"nested.hk.yaml":  "kind: API\nurl: https://nk.example/x\nbody:\n    kind: graphql\n// stray\n",
		"suffix.hk.yaml":  "kind: graphqlx\nurl: https://sx.example/x\n// stray\n",
		"a_good.hk.yaml":  "kind: GraphQL\nurl: https://same.example/gql\n",
		"b_bad.hk.yaml":   "kind: GraphQL\nurl: https://same.example/gql\nheaders: {a: b\n",
		"upper.hk.yaml":   "Kind: GraphQL\nURL: https://up.example/gql\n// stray\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	found, _, err := FindGraphQLFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"https://dq.example/gql":            "dq.hk.yaml",
		"https://sq.example/gql":            "sq.hk.yaml",
		"https://cm.example/gql":            "comment.hk.yaml",
		"https://crlf.example/gql":          "crlf.hk.yaml",
		"https://top.example/gql":           "indent.hk.yaml",
		"https://same.example/gql":          "a_good.hk.yaml",
		"https://up.example/gql":            "upper.hk.yaml",
		filepath.Join(dir, "b_bad.hk.yaml"): "b_bad.hk.yaml",
	}
	for url, name := range want {
		if found[url] != filepath.Join(dir, name) {
			t.Errorf("found[%q] = %q, want %s", url, found[url], name)
		}
	}
	if len(found) != len(want) {
		t.Errorf("found %d files, want %d: %v", len(found), len(want), found)
	}
}

func TestPeekFileInfo_278_DecodeErrorIsKeptWhenTheParseSucceeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "alias.hk.yaml")
	if err := os.WriteFile(path, []byte("kind: GraphQL\nurl: https://a.example/gql\nheaders: *nope\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := peekFileInfo(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.parseErr == nil || !strings.HasPrefix(info.parseErr.Error(), "parsing "+path+": ") {
		t.Errorf("parseErr = %v, want a decode error naming the file", info.parseErr)
	}
	if _, _, err := ValidateGraphQLFile(path); err == nil || !strings.Contains(err.Error(), path) {
		t.Errorf("ValidateGraphQLFile error = %v, want it to name the file", err)
	}
}

func TestFindGraphQLFiles_278_FallbackReadsOnlyTheFirstDocument(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"later.hk.yaml":    "x: 1\n---\nkind: GraphQL\nurl: https://later.example/gql\nheaders:\n\tX: y\n",
		"lead.hk.yaml":     "---\nkind: GraphQL\nurl: https://lead.example/gql\n// stray\n",
		"spaced.hk.yaml":   "kind : GraphQL\nurl : https://spaced.example/gql\n// stray\n",
		"frag.hk.yaml":     "kind: GraphQL\nurl: https://frag.example/gql#v2\n// stray\n",
		"unclosed.hk.yaml": "kind: GraphQL\nurl: \"https://open.example/gql\n// stray\n",
		"a_bad.hk.yaml":    "kind: GraphQL\nurl: https://same.example/gql\nheaders: {a: b\n",
		"b_good.hk.yaml":   "kind: GraphQL\nurl: https://same.example/gql\n",
		"cmt.hk.yaml":      "# c\n---\nkind: GraphQL\nurl: https://cmt.example/gql\n// stray\n",
		"dir.hk.yaml":      "%YAML 1.2\n---\nkind: GraphQL\nurl: https://dir.example/gql\n// stray\n",
		"dots.hk.yaml":     "x: 1\n...\nkind: GraphQL\nurl: https://dots.example/gql\nheaders:\n\tX: y\n",
		"sepc.hk.yaml":     "x: 1\n--- # c\nkind: GraphQL\nurl: https://sepc.example/gql\nheaders:\n\tX: y\n",
		"todo.hk.yaml":     "kind: GraphQL\nurl: # todo\nheaders:\n\tX: y\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	found, _, err := FindGraphQLFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"https://lead.example/gql":          "lead.hk.yaml",
		"https://spaced.example/gql":        "spaced.hk.yaml",
		"https://frag.example/gql#v2":       "frag.hk.yaml",
		"https://open.example/gql":          "unclosed.hk.yaml",
		"https://same.example/gql":          "b_good.hk.yaml",
		filepath.Join(dir, "a_bad.hk.yaml"): "a_bad.hk.yaml",
		"https://cmt.example/gql":           "cmt.hk.yaml",
		"https://dir.example/gql":           "dir.hk.yaml",
	}
	for url, name := range want {
		if found[url] != filepath.Join(dir, name) {
			t.Errorf("found[%q] = %q, want %s", url, found[url], name)
		}
	}
	if len(found) != len(want) {
		t.Errorf("found %d files, want %d: %v", len(found), len(want), found)
	}
}

func TestFindGraphQLFiles_278_BrokenFilesNeverHideEachOther(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.hk.yaml", "b.hk.yaml"} {
		content := "kind: GraphQL\nurl: >-\n  https://" + name + "/gql\nx: 1\nx: 2\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	found, _, err := FindGraphQLFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, path := range found {
		listed[filepath.Base(path)] = true
	}
	if !listed["a.hk.yaml"] || !listed["b.hk.yaml"] {
		t.Errorf("both broken files must be listed so each error is reported, got %v", found)
	}
}
