package yamlparser

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// createTempYAMLFile is the helper function
func createTempYAMLFile(t *testing.T, content string) string {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("could not create temporary file: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Remove(tmpfile.Name())
	})

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("could not write to temporary file: %v", err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatalf("could not close temporary file: %v", err)
	}

	return tmpfile.Name()
}

func TestKindNormalization(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		want Kind
	}{
		{"empty => API", "", KindAPI},
		{"api lowercase => API", "api", KindAPI},
		{"Api mixed => API", "Api", KindAPI},
		{"AUTH upper => Auth", "AUTH", KindAuth},
		{"auth lower => Auth", "auth", KindAuth},
		{"invalid returns lowercase", "Invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.kind.normalize()
			if got != tt.want {
				t.Errorf("normalize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKindHelpers(t *testing.T) {
	tests := []struct {
		name     string
		kind     Kind
		wantAuth bool
		wantAPI  bool
		wantGQL  bool
	}{
		{"empty => API", "", false, true, false},
		{"API", "API", false, true, false},
		{"api lower", "api", false, true, false},
		{"Auth", "Auth", true, false, false},
		{"auth lower", "auth", true, false, false},
		{"GraphQL", "GraphQL", false, false, true},
		{"graphql lower", "graphql", false, false, true},
		{"invalid => none", "invalid", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ConfigType{Kind: tt.kind}
			if got := conf.IsAuth(); got != tt.wantAuth {
				t.Errorf("IsAuth() = %v, want %v", got, tt.wantAuth)
			}
			if got := conf.IsAPI(); got != tt.wantAPI {
				t.Errorf("IsAPI() = %v, want %v", got, tt.wantAPI)
			}
			if got := conf.IsGraphql(); got != tt.wantGQL {
				t.Errorf("IsGraphql() = %v, want %v", got, tt.wantGQL)
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		createFile  bool
		want        ConfigType
	}{
		// API kinds
		{"API lowercase", `kind: api`, true, ConfigType{Kind: KindAPI}},
		{"API uppercase", `kind: API`, true, ConfigType{Kind: KindAPI}},
		{"API mixed", `kind: ApI`, true, ConfigType{Kind: KindAPI}},

		// Auth kinds
		{"Auth lowercase", `kind: auth`, true, ConfigType{Kind: KindAuth}},
		{"Auth uppercase", `kind: AUTH`, true, ConfigType{Kind: KindAuth}},
		{"Auth mixed", `kind: AuTh`, true, ConfigType{Kind: KindAuth}},

		// GraphQL kinds
		{"GraphQL", `kind: GraphQL`, true, ConfigType{Kind: KindGraphQL}},
		{"graphql lower", `kind: graphql`, true, ConfigType{Kind: KindGraphQL}},

		// Missing kind (defaults to API)
		{"missing kind defaults to API", `method: POST`, true, ConfigType{Kind: KindAPI}},

		// Invalid kinds
		{"invalid kind", `kind: invalid`, true, ConfigType{Kind: "invalid"}},

		// Timeout pass-through (validation tested in TestParsedTimeout).
		{
			"timeout passes through",
			"kind: API\ntimeout: 5m",
			true,
			ConfigType{Kind: KindAPI, Timeout: "5m"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.createFile {
				path = createTempYAMLFile(t, tt.yamlContent)
			} else {
				path = filepath.Join(os.TempDir(), "does-not-exist.yaml")
			}

			got, err := ParseConfig(path, nil)
			if err != nil {
				t.Errorf("ParseConfig returned unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(*got, tt.want) {
				t.Errorf("ParseConfig() = %#v, want %#v", *got, tt.want)
			}
		})
	}
}

// TestParseConfig_InvalidTimeout asserts that a bad `timeout:` value fails
// the file with a clear, file-scoped error rather than silently falling back.
func TestParseConfig_InvalidTimeout(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		errMatch string
	}{
		{"missing unit", "kind: API\ntimeout: 60", "invalid timeout"},
		{"garbage", "kind: API\ntimeout: not-a-duration", "invalid timeout"},
		{"zero", "kind: API\ntimeout: 0s", "must be positive"},
		{"negative", "kind: API\ntimeout: -5m", "must be positive"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := createTempYAMLFile(t, tc.yaml)
			_, err := ParseConfig(path, nil)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.errMatch) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.errMatch)
			}
		})
	}
}

// TestParsedTimeout covers the value-level parsing rules: unset returns 0,
// valid durations parse, malformed/non-positive errors clearly.
func TestParsedTimeout(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    time.Duration
		wantErr bool
	}{
		{"empty unset", "", 0, false},
		{"5m", "5m", 5 * time.Minute, false},
		{"90s", "90s", 90 * time.Second, false},
		{"2m30s", "2m30s", 2*time.Minute + 30*time.Second, false},
		{"missing unit", "60", 0, true},
		{"garbage", "soon", 0, true},
		{"zero", "0s", 0, true},
		{"negative", "-1s", 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := &ConfigType{Timeout: tc.value}
			got, err := c.ParsedTimeout()
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPeekKind(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	cases := []struct {
		name string
		body string
		want Kind
	}{
		{"graphql.hk.yaml", "kind: GraphQL\nurl: \"{{.baseUrl}}\"\nbody:\n  graphql:\n    query: '{{getFile \"q.gql\"}}'\n", KindGraphQL},
		{"api.hk.yaml", "kind: API\nurl: http://x\n", KindAPI},
		{"lower.hk.yaml", "kind: graphql\nurl: http://x\n", KindGraphQL},
		{"nokind.hk.yaml", "url: http://x\n", KindAPI}, // defaults to API
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PeekKind(write(tc.name, tc.body))
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("PeekKind = %q, want %q", got, tc.want)
			}
		})
	}

	t.Run("missing file errors", func(t *testing.T) {
		if _, err := PeekKind(filepath.Join(dir, "nope.yaml")); err == nil {
			t.Error("expected error for missing file")
		}
	})
}
