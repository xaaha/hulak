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

func TestPeekFileInfo_SplitFileReportsSplitAndNoEnv(t *testing.T) {
	path := writeSplit(t, t.TempDir())
	info, err := peekFileInfo(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.splitErr == nil || info.splitErr.Error() != testutil.SingleDocError(path, 3) {
		t.Errorf("splitErr = %v, want %q", info.splitErr, testutil.SingleDocError(path, 3))
	}
	if info.needsEnv {
		t.Error("a split file must not ask for an environment")
	}
	if info.kind != "graphql" || info.url != "{{.base}}/gql" {
		t.Errorf("first document: kind %q url %q", info.kind, info.url)
	}
}

func TestValidateGraphQLFile_RejectsSplitFile(t *testing.T) {
	path := writeSplit(t, t.TempDir())
	_, _, err := ValidateGraphQLFile(path)
	if err == nil || err.Error() != testutil.SingleDocError(path, 3) {
		t.Errorf("err = %v, want %q", err, testutil.SingleDocError(path, 3))
	}
}

func TestFindGraphQLFiles_KeepsSplitFileWithoutAskingForEnv(t *testing.T) {
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
