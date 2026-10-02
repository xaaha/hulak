package yamlparser

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml/parser"
	"github.com/xaaha/hulak/pkg/utils/testutil"
)

const strayLineRequest = "method: POST\n// stray\nurl: \"https://e.com\"\n"

func TestD21FinalStructForAPIRejectsStrayLine(t *testing.T) {
	path := createTempYAMLFile(t, strayLineRequest)

	_, ok, err := FinalStructForAPI(path, map[string]any{})
	if err == nil {
		t.Fatalf("expected an error for a split request file, got ok=%v", ok)
	}
	if want := testutil.SingleDocError(path, 2); err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

func TestD21PeekConfigRejectsStrayLine(t *testing.T) {
	path := createTempYAMLFile(t, "kind: API\n"+strayLineRequest)

	_, err := PeekConfig(path)
	if err == nil {
		t.Fatal("expected an error for a split request file")
	}
	if want := testutil.SingleDocError(path, 3); err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

func TestD21ReEncodedBufferIsSingleDocument(t *testing.T) {
	path := createTempYAMLFile(t, strings.Join([]string{
		"kind: API",
		"method: POST",
		"url: https://e.com",
		"headers:",
		"  Content-Type: application/json",
		"body:",
		"  graphql:",
		"    query: |",
		"      ---",
		"      query X { y }",
		"",
	}, "\n"))

	buf, err := checkYamlFile(path, map[string]any{})
	if err != nil {
		t.Fatalf("checkYamlFile: %v", err)
	}

	file, err := parser.ParseBytes(buf.Bytes(), 0)
	if err != nil {
		t.Fatalf("parsing re-encoded buffer: %v", err)
	}
	if len(file.Docs) != 1 {
		t.Errorf("re-encoded buffer has %d documents, want 1: %s", len(file.Docs), buf.String())
	}
}

func TestD22SeparatorsAroundOneDocumentStillLoad(t *testing.T) {
	path := createTempYAMLFile(t, "---\nmethod: GET\nurl: https://e.com\n---\n")

	file, ok, err := FinalStructForAPI(path, map[string]any{})
	if err != nil || !ok {
		t.Fatalf("want a successful parse, got ok=%v err=%v", ok, err)
	}
	if file.URL != "https://e.com" {
		t.Errorf("url = %q, want %q", file.URL, "https://e.com")
	}
}
