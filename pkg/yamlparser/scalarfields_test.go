package yamlparser

import (
	"fmt"
	"strings"
	"testing"
)

func TestD1_278_NonTextRequestFieldsReturnErrors(t *testing.T) {
	for _, value := range []string{"8080", "true", "1.5"} {
		t.Run("url "+value, func(t *testing.T) {
			path := createTempYAMLFile(t, "method: GET\nurl: "+value+"\n")
			_, _, err := FinalStructForAPI(path, map[string]any{})
			if want := fmt.Sprintf("missing or invalid URL: %s in file %s", value, path); err == nil || err.Error() != want {
				t.Errorf("err = %v, want %q", err, want)
			}
		})
		t.Run("method "+value, func(t *testing.T) {
			path := createTempYAMLFile(t, "method: "+value+"\nurl: https://e.com\n")
			_, _, err := FinalStructForAPI(path, map[string]any{})
			if want := fmt.Sprintf("invalid HTTP method '%s' in '%s'", value, path); err == nil || err.Error() != want {
				t.Errorf("err = %v, want %q", err, want)
			}
		})
		t.Run("kind "+value, func(t *testing.T) {
			path := createTempYAMLFile(t, "kind: "+value+"\nmethod: GET\nurl: https://e.com\n")
			cfg, err := ParseConfig(path, map[string]any{})
			if err != nil {
				t.Fatal(err)
			}
			if cfg.IsAPI() || cfg.IsGraphql() || cfg.IsAuth() {
				t.Errorf("kind %s must be an unsupported kind, got %q", value, cfg.Kind)
			}
		})
	}
}

func TestD1_278_NonTextAuthFieldsReturnErrors(t *testing.T) {
	want := "error on auth2 request body: invalid 'auth' section. Make sure the Auth2.0 file contains valid auth section with 'type' && access_token_url"
	for name, auth := range map[string]string{
		"type":             "  type: 5\n  access_token_url: https://e.com/t\n",
		"access_token_url": "  type: OAuth2.0\n  access_token_url: 9\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := createTempYAMLFile(t, "kind: Auth\nmethod: POST\nurl: https://e.com\nauth:\n"+auth)
			_, err := FinalStructForOAuth2(path, map[string]any{})
			if err == nil || err.Error() != want {
				t.Errorf("err = %v, want %q", err, want)
			}
		})
	}
}

func TestD1_278_NumericEnvValueAsWholeURL(t *testing.T) {
	path := createTempYAMLFile(t, "method: GET\nurl: \"{{.whole}}\"\n")
	_, _, err := FinalStructForAPI(path, map[string]any{"whole": 8080})
	if want := "missing or invalid URL: 8080 in file " + path; err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
}

func TestD4_278_RepeatedKeyIsAnError(t *testing.T) {
	for name, tc := range map[string]struct{ content, at, first string }{
		"url":    {"method: GET\nurl: https://e.com\nurl: https://f.com\n", "[3:1] ", "[2:1]"},
		"header": {"method: GET\nurl: https://e.com\nheaders:\n  X-A: one\n  X-A: two\n", "[5:3] ", "[4:3]"},
	} {
		t.Run(name, func(t *testing.T) {
			path := createTempYAMLFile(t, tc.content)
			_, _, err := FinalStructForAPI(path, map[string]any{})
			want := "parsing " + path + ": " + tc.at
			if err == nil || !strings.HasPrefix(err.Error(), want) || !strings.Contains(err.Error(), tc.first) {
				t.Errorf("err = %v, want it to start with %q and name %s", err, want, tc.first)
			}
		})
	}
}

func TestD5_278_TabIndentationIsAnError(t *testing.T) {
	path := createTempYAMLFile(t, "method: GET\nurl: https://e.com\nheaders:\n\tX-A: b\n")
	_, _, err := FinalStructForAPI(path, map[string]any{})
	if want := "parsing " + path + ": [4:1] "; err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("err = %v, want it to start with %q", err, want)
	}
}
