package yamlparser

import (
	"fmt"
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

func TestD9_278_PeekRequestKindSurvivesTabValues(t *testing.T) {
	path := createTempYAMLFile(t, "method: POST\nurl: https://e.com/gql\nheaders:\n  X-T: \"k:\\tv\"\nbody:\n  graphql:\n    query: \"query { a }\"\n")
	kind, err := PeekRequestKind(path)
	if err != nil || kind != KindGraphQL {
		t.Errorf("PeekRequestKind = %q, %v, want %q", kind, err, KindGraphQL)
	}
}

func TestD9_278_GraphQLVariablesKeepTypesAndCase(t *testing.T) {
	path := createTempYAMLFile(t, "kind: GraphQL\nmethod: POST\nurl: https://e.com/gql\nbody:\n  graphql:\n    query: \"query { a }\"\n    variables:\n      userId: 5\n      on: true\n      ratio: 1.5\n      none: null\n      list: []\n      obj: {}\n      fromEnv: \"{{.num}}\"\n")
	file, _, err := FinalStructForGraphQL(path, map[string]any{"num": 7})
	if err != nil {
		t.Fatal(err)
	}
	vars, _ := file.Body.Graphql.Variables.(map[string]any)
	want := map[string]any{
		"userId": uint64(5), "on": true, "ratio": 1.5, "none": nil,
		"list": []any{}, "obj": map[string]any{}, "fromEnv": uint64(7),
	}
	if len(vars) != len(want) {
		t.Fatalf("variables = %#v, want %#v", vars, want)
	}
	for k, w := range want {
		got, ok := vars[k]
		if !ok || fmt.Sprintf("%#v", got) != fmt.Sprintf("%#v", w) {
			t.Errorf("variable %s = %#v (present %v), want %#v", k, got, ok, w)
		}
	}
}

func TestD11_278_EveryDecodeAfterSubstitutionLeavesOutTheRequest(t *testing.T) {
	secrets := map[string]any{"api_token": "ctl\x01SECRET-value", "body": "FILEBODY-content"}
	for name, tc := range map[string]struct {
		content string
		decode  func(path string) error
	}{
		"ParseConfig": {
			"kind: [a]\nmethod: GET\nurl: https://e.com\nheaders:\n  X-K: \"{{.api_token}}\"\n  X-B: \"{{.body}}\"\n",
			func(path string) error { _, err := ParseConfig(path, secrets); return err },
		},
		"FinalStructForGraphQL": {
			"kind: GraphQL\nmethod: POST\nurl: https://e.com\nheaders:\n  X-K: \"{{.api_token}}\"\n  X-B: \"{{.body}}\"\n  X-L: [a]\n",
			func(path string) error { _, _, err := FinalStructForGraphQL(path, secrets); return err },
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := tc.decode(createTempYAMLFile(t, tc.content))
			if err == nil {
				t.Fatal("want a decode error")
			}
			if strings.Contains(err.Error(), "SECRET") || strings.Contains(err.Error(), "FILEBODY") {
				t.Errorf("error carries the substituted request:\n%v", err)
			}
		})
	}
}
