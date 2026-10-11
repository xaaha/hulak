package graphql

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xaaha/hulak/pkg/utils/testutil"
)

const splitGraphQL = "kind: GraphQL\nurl: \"{{.base}}/gql\"\n---\nheaders:\n  X-A: b\n"

func writeSplit(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "split.hk.yaml")
	if err := os.WriteFile(path, []byte(splitGraphQL), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPeekFileInfo_269_SplitFileReportsSplitAndNoEnv(t *testing.T) {
	path := writeSplit(t, t.TempDir())
	info, err := peekFileInfo(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.parseErr == nil || info.parseErr.Error() != testutil.SingleDocError(path, 3) {
		t.Errorf("parseErr = %v, want %q", info.parseErr, testutil.SingleDocError(path, 3))
	}
	if info.needsEnv {
		t.Error("a split file must not ask for an environment")
	}
	if info.kind != "graphql" || info.url != "{{.base}}/gql" {
		t.Errorf("first document: kind %q url %q", info.kind, info.url)
	}
}

func TestValidateGraphQLFile_269_RejectsSplitFile(t *testing.T) {
	path := writeSplit(t, t.TempDir())
	_, _, err := ValidateGraphQLFile(path)
	if err == nil || err.Error() != testutil.SingleDocError(path, 3) {
		t.Errorf("err = %v, want %q", err, testutil.SingleDocError(path, 3))
	}
}

func TestFindGraphQLFiles_269_KeepsSplitFileWithoutAskingForEnv(t *testing.T) {
	dir := t.TempDir()
	path := writeSplit(t, dir)
	files, needsEnv, err := FindGraphQLFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if files["{{.base}}/gql"] != path {
		t.Errorf("split file dropped from discovery: %v", files)
	}
	if needsEnv || NeedsEnvResolution(files) {
		t.Error("a split file must not ask for an environment")
	}
	results := ProcessFilesConcurrent([]string{path}, nil)
	if len(results) != 1 || results[0].Error == nil || results[0].Error.Error() != testutil.SingleDocError(path, 3) {
		t.Errorf("processing must report the split file: %+v", results)
	}
}

func TestNeedsEnvResolution_269_SplitFileDoesNotHideOthers(t *testing.T) {
	dir := t.TempDir()
	split := writeSplit(t, dir)
	templated := filepath.Join(dir, "templated.hk.yaml")
	if err := os.WriteFile(templated, []byte("kind: GraphQL\nurl: '{{getValueOf \"url\" \"config\"}}'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mixed := map[string]string{"{{.base}}/gql": split, `{{getValueOf "url" "config"}}`: templated}
	for range 64 {
		if !NeedsEnvResolution(mixed) {
			t.Fatal("a templated single-document file must still ask for an environment next to a split file")
		}
	}
	if !NeedsEnvResolution(map[string]string{`{{getValueOf "url" "config"}}`: templated}) {
		t.Error("a templated url in a single-document file must ask for an environment")
	}
}

func TestValidateGraphQLFile_269_RejectsSplitFileWithPlainURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain.hk.yaml")
	if err := os.WriteFile(path, []byte("kind: GraphQL\nurl: https://e.com/gql\n---\nheaders:\n  X-A: b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := ValidateGraphQLFile(path)
	if err == nil || err.Error() != testutil.SingleDocError(path, 3) {
		t.Errorf("err = %v, want %q", err, testutil.SingleDocError(path, 3))
	}
}

func TestValidateGraphQLFile_269_RejectsSplitBeforeURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "early.hk.yaml")
	if err := os.WriteFile(path, []byte("kind: GraphQL\n---\nurl: \"{{.base}}/gql\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := ValidateGraphQLFile(path)
	if err == nil || err.Error() != testutil.SingleDocError(path, 2) {
		t.Errorf("err = %v, want %q", err, testutil.SingleDocError(path, 2))
	}
}
