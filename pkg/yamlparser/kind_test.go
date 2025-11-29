package yamlparser

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
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

	if _, err := tmpfile.Write([]byte(content)); err != nil {
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
		{"API mixed", `kind: ApI`, true, ConfigType{KindAPI}},

		// Auth kinds
		{"Auth lowercase", `kind: auth`, true, ConfigType{KindAuth}},
		{"Auth uppercase", `kind: AUTH`, true, ConfigType{KindAuth}},
		{"Auth mixed", `kind: AuTh`, true, ConfigType{KindAuth}},

		// GraphQL kinds
		{"GraphQL", `kind: GraphQL`, true, ConfigType{KindGraphQL}},
		{"graphql lower", `kind: graphql`, true, ConfigType{KindGraphQL}},

		// Missing kind (defaults to API)
		{"missing kind defaults to API", `method: POST`, true, ConfigType{Kind: KindAPI}},

		// Invalid kinds
		{"invalid kind", `kind: invalid`, true, ConfigType{Kind: "invalid"}},
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
