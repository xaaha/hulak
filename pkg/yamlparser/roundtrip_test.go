package yamlparser

import (
	"strings"
	"testing"
)

func TestD9_278_SubstitutedValuesSurviveTheRoundTrip(t *testing.T) {
	values := map[string]string{
		"tab":     "a\tb",
		"lead":    "\tlead",
		"colon":   "k:\tv",
		"cr":      "a\rb",
		"newline": "\n",
		"at":      "@me",
		"null":    "null",
		"zip":     "08544",
	}
	content := "method: GET\nurl: https://e.com\nheaders:\n"
	secrets := map[string]any{}
	for name, v := range values {
		content += "  X-" + name + ": \"{{." + name + "}}\"\n"
		secrets[name] = v
	}
	file, _, err := FinalStructForAPI(createTempYAMLFile(t, content), secrets)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range values {
		if got := file.Headers["x-"+name]; got != want {
			t.Errorf("header x-%s = %q, want %q", name, got, want)
		}
	}
}

func TestD9_278_EscapedSourceValueSurvivesTheRoundTrip(t *testing.T) {
	path := createTempYAMLFile(t, "kind: GraphQL\nmethod: POST\nurl: https://e.com/gql\nbody:\n  graphql:\n    query: \"query { a }\"\n    variables:\n      path: \"C:\\temp\"\n")
	file, _, err := FinalStructForGraphQL(path, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	vars, _ := file.Body.Graphql.Variables.(map[string]any)
	if got := vars["path"]; got != "C:\temp" {
		t.Errorf("variable path = %q, want %q", got, "C:\temp")
	}
}

func TestD9_278_DecodeErrorLeavesOutTheSubstitutedRequest(t *testing.T) {
	big := strings.Repeat("x", 200_000)
	path := createTempYAMLFile(t, "method: GET\nurl: https://e.com\nheaders:\n  X-Big: \"{{.big}}\"\n  X-L: [a]\n")
	_, _, err := FinalStructForAPI(path, map[string]any{"big": big})
	if err == nil {
		t.Fatal("want a decode error for the list header")
	}
	if len(err.Error()) > 300 || strings.Contains(err.Error(), "xxxx") {
		t.Errorf("error carries the substituted request (%d bytes)", len(err.Error()))
	}
}
