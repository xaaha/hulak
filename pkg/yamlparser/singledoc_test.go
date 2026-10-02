package yamlparser

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml/parser"
)

const strayLineRequest = "method: POST\n// stray\nurl: \"https://e.com\"\n"

func TestD21FinalStructForAPIRejectsStrayLine(t *testing.T) {
	path := createTempYAMLFile(t, strayLineRequest)

	_, ok, err := FinalStructForAPI(path, map[string]any{})
	if err == nil {
		t.Fatalf("expected an error for a split request file, got ok=%v", ok)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error should name the file %q, got %q", path, err)
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error should name line 2, got %q", err)
	}
}

func TestD21PeekConfigRejectsStrayLine(t *testing.T) {
	path := createTempYAMLFile(t, "kind: API\n"+strayLineRequest)

	if _, err := PeekConfig(path); err == nil {
		t.Fatal("expected an error for a split request file")
	} else if !strings.Contains(err.Error(), "line 3") {
		t.Errorf("error should name line 3, got %q", err)
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
