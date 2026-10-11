package gqlexplorer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"

	"github.com/xaaha/hulak/pkg/features/graphql"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func savedHeaders(t *testing.T, dir string) any {
	t.Helper()
	saved, err := os.ReadFile(filepath.Join(dir, "getUser.hk.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(saved), "s3cret-token") {
		t.Fatalf("saved request holds a resolved secret:\n%s", saved)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(saved, &doc); err != nil {
		t.Fatalf("saved request is not valid YAML: %v\n%s", err, saved)
	}
	return doc["headers"]
}

func TestCreateHulakRequestFile_269_D7_HeadersComeOnlyFromParent(t *testing.T) {
	defaultHeaders := map[string]any{"Content-Type": "application/json"}
	tests := []struct {
		name    string
		headers string
		want    any
	}{
		{"template string", "headers: '{{getValueOf \"hdrs\" \"tok.json\"}}'\n", `{{getValueOf "hdrs" "tok.json"}}`},
		{"missing", "", defaultHeaders},
		{"null", "headers:\n", defaultHeaders},
		{"empty mapping", "headers: {}\n", defaultHeaders},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, dir := saveFromParent(t, "kind: GraphQL\nurl: \"{{.base}}/gql\"\n"+tc.headers)
			if got := savedHeaders(t, dir); !equalYAML(got, tc.want) {
				t.Errorf("saved headers = %#v, want %#v (notification %q)", got, tc.want, m.notification.CopyText())
			}
		})
	}
}

func TestCreateHulakRequestFile_269_D7_RefusesHeadersOfAnotherShape(t *testing.T) {
	m, dir := saveFromParent(t, "kind: GraphQL\nurl: \"{{.base}}/gql\"\nheaders:\n  - Authorization\n")
	want := "Save failed: " + filepath.Join(dir, "parent.hk.yaml") + ": headers must be a mapping or a template string"
	if got := m.notification.CopyText(); got != want {
		t.Errorf("notification = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "getUser.hk.yaml")); err == nil {
		t.Error("a refused save wrote getUser.hk.yaml")
	}
}

func TestCreateHulakRequestFile_269_D7_TemplatedHeadersFromARealRun(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.Mkdir(filepath.Join(dir, utils.EnvironmentFolder), utils.DirPer); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tok.json"), []byte(`{"hdrs": {"Authorization": "Bearer s3cret-token"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	parentPath := filepath.Join(dir, "parent.hk.yaml")
	parent := "kind: GraphQL\nurl: https://api.example.com/gql\nheaders: '{{getValueOf \"hdrs\" \"tok.json\"}}'\n"
	if err := os.WriteFile(parentPath, []byte(parent), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := graphql.ProcessGraphQLFile(parentPath, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if info.Headers["Authorization"] != "Bearer s3cret-token" {
		t.Fatalf("the run did not resolve the headers template: %#v", info.Headers)
	}
	m := &Model{
		filtered:        []UnifiedOperation{{Name: "getUser", Endpoint: info.URL}},
		schemaFilePaths: map[string]string{info.URL: parentPath},
		apiInfos:        map[string]yamlparser.APIInfo{info.URL: info},
	}
	m.createHulakRequestFile()
	if got := savedHeaders(t, dir); got != `{{getValueOf "hdrs" "tok.json"}}` {
		t.Errorf("saved headers = %#v, want the parent's template", got)
	}
}

func equalYAML(got, want any) bool {
	g, err1 := yaml.Marshal(got)
	w, err2 := yaml.Marshal(want)
	return err1 == nil && err2 == nil && bytes.Equal(g, w)
}

func TestCreateHulakRequestFile_269_D8_RefusesParentWithoutURL(t *testing.T) {
	for name, url := range map[string]string{"missing": "", "not a string": "url: 42\n"} {
		t.Run(name, func(t *testing.T) {
			m, dir := saveFromParent(t, "kind: GraphQL\n"+url+"headers:\n  Authorization: \"Bearer {{.token}}\"\n")
			want := "Save failed: " + filepath.Join(dir, "parent.hk.yaml") + ": url must be a non-empty string"
			if got := m.notification.CopyText(); got != want {
				t.Errorf("notification = %q, want %q", got, want)
			}
			if _, err := os.Stat(filepath.Join(dir, "getUser.hk.yaml")); err == nil {
				t.Error("a refused save wrote getUser.hk.yaml")
			}
		})
	}
}

func TestCreateHulakRequestFile_269_MixedCaseParentKeys(t *testing.T) {
	_, dir := saveFromParent(t, "Kind: GraphQL\nURL: \"{{.base}}/gql\"\nHeaders:\n  Authorization: \"Bearer {{.token}}\"\n")
	saved, err := os.ReadFile(filepath.Join(dir, "getUser.hk.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), `url: "{{.base}}/gql"`) || !strings.Contains(string(saved), "{{.token}}") {
		t.Errorf("saved request must keep the parent's url and headers:\n%s", saved)
	}
}
