package yamlparser

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestConfigType_IsValid(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		want bool
	}{
		{
			name: "empty kind defaults to API",
			kind: "",
			want: true,
		},
		{
			name: "valid API uppercase",
			kind: "API",
			want: true,
		},
		{
			name: "valid API lowercase",
			kind: "api",
			want: true,
		},
		{
			name: "valid API mixed case",
			kind: "Api",
			want: true,
		},
		{
			name: "valid Auth uppercase",
			kind: "AUTH",
			want: true,
		},
		{
			name: "valid Auth lowercase",
			kind: "auth",
			want: true,
		},
		{
			name: "invalid kind",
			kind: "invalid",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ConfigType{Kind: tt.kind}
			if got := conf.IsValid(); got != tt.want {
				t.Errorf("ConfigType.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigType_GetKind(t *testing.T) {
	tests := []struct {
		name string
		kind Kind
		want Kind
	}{
		{
			name: "empty kind returns API",
			kind: "",
			want: KindAPI,
		},
		{
			name: "API different case returns canonical form",
			kind: "api",
			want: KindAPI,
		},
		{
			name: "Auth different case returns canonical form",
			kind: "auth",
			want: KindAuth,
		},
		{
			name: "invalid kind returns lowercase version",
			kind: "Invalid",
			want: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ConfigType{Kind: tt.kind}
			if got := conf.GetKind(); got != tt.want {
				t.Errorf("ConfigType.GetKind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateKinds(t *testing.T) {
	tests := []struct {
		name        string
		kinds       []Kind
		wantInvalid []string
		wantValid   bool
	}{
		{
			name:        "all valid kinds",
			kinds:       []Kind{"API", "Auth"},
			wantInvalid: nil,
			wantValid:   true,
		},
		{
			name:        "mixed case valid kinds",
			kinds:       []Kind{"api", "AUTH"},
			wantInvalid: nil,
			wantValid:   true,
		},
		{
			name:        "some invalid kinds",
			kinds:       []Kind{"API", "invalid", "Auth", "unknown"},
			wantInvalid: []string{"invalid", "unknown"},
			wantValid:   false,
		},
		{
			name:        "empty list",
			kinds:       []Kind{},
			wantInvalid: nil,
			wantValid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotInvalid, gotValid := ValidateKinds(tt.kinds)
			if !reflect.DeepEqual(gotInvalid, tt.wantInvalid) {
				t.Errorf("ValidateKinds() invalid = %v, want %v", gotInvalid, tt.wantInvalid)
			}
			if gotValid != tt.wantValid {
				t.Errorf("ValidateKinds() valid = %v, want %v", gotValid, tt.wantValid)
			}
		})
	}
}

func TestConfigType_IsAuth_IsAPI(t *testing.T) {
	tests := []struct {
		name     string
		kind     Kind
		wantAuth bool
		wantAPI  bool
	}{
		{
			name:     "empty kind is API",
			kind:     "",
			wantAuth: false,
			wantAPI:  true,
		},
		{
			name:     "API uppercase",
			kind:     "API",
			wantAuth: false,
			wantAPI:  true,
		},
		{
			name:     "Auth uppercase",
			kind:     "AUTH",
			wantAuth: true,
			wantAPI:  false,
		},
		{
			name:     "invalid kind is neither",
			kind:     "invalid",
			wantAuth: false,
			wantAPI:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &ConfigType{Kind: tt.kind}
			if got := conf.IsAuth(); got != tt.wantAuth {
				t.Errorf("ConfigType.IsAuth() = %v, want %v", got, tt.wantAuth)
			}
			if got := conf.IsAPI(); got != tt.wantAPI {
				t.Errorf("ConfigType.IsAPI() = %v, want %v", got, tt.wantAPI)
			}
		})
	}
}

// helper function to create temporary YAML file with content
func createTempYAMLFile(t *testing.T, content string) string {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "test-*.yaml")
	if err != nil {
		t.Fatalf("could not create temporary file: %v", err)
	}

	t.Cleanup(func() {
		os.Remove(tmpfile.Name())
	})

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatalf("could not write to temporary file: %v", err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatalf("could not close temporary file: %v", err)
	}

	return tmpfile.Name()
}

func TestConfigParsing(t *testing.T) {
	tests := []struct {
		name             string
		yamlContent      string
		createFile       bool
		secretsMap       map[string]interface{}
		want             ConfigType
		wantPanic        bool
		expectedPanicMsg string
	}{
		// Case insensitive API tests
		{
			name:        "valid API lowercase",
			yamlContent: `kind: api`,
			createFile:  true,
			want:        ConfigType{Kind: KindAPI},
		},
		{
			name:        "valid API uppercase",
			yamlContent: `kind: API`,
			createFile:  true,
			want:        ConfigType{Kind: KindAPI},
		},
		{
			name:        "valid API when kind is missing but file is not empty",
			yamlContent: `method: Post`,
			createFile:  true,
			want:        ConfigType{Kind: KindAPI},
		},
		{
			name:        "valid API mixed case",
			yamlContent: `kind: ApI`,
			createFile:  true,
			want:        ConfigType{Kind: KindAPI},
		},

		// Case insensitive Auth tests
		{
			name:        "valid Auth lowercase",
			yamlContent: `kind: auth`,
			createFile:  true,
			want:        ConfigType{Kind: KindAuth},
		},
		{
			name:        "valid Auth uppercase",
			yamlContent: `kind: AUTH`,
			createFile:  true,
			want:        ConfigType{Kind: KindAuth},
		},
		{
			name:        "valid Auth mixed case",
			yamlContent: `kind: AuTh`,
			createFile:  true,
			want:        ConfigType{Kind: KindAuth},
		},

		// Default and special cases
		{
			name:        "missing kind field defaults to API",
			yamlContent: `other_field: value`,
			createFile:  true,
			want:        ConfigType{Kind: KindAPI},
		},
		{
			name:        "invalid kind value",
			yamlContent: `kind: invalid`,
			createFile:  true,
			want:        ConfigType{Kind: "invalid"},
		},
		// Error cases like invalid syntax is refused from libraray goccy yaml parserA
		// Non exiistant file is throws out...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tmpPath string
			if tt.createFile {
				tmpPath = createTempYAMLFile(t, tt.yamlContent)
			} else {
				tmpPath = filepath.Join(os.TempDir(), "non-existent-file.yaml")
			}

			defer func() {
				r := recover()
				if tt.wantPanic {
					if r == nil {
						t.Error("Expected panic did not occur")
						return
					}
					panicMsg, ok := r.(string)
					if !ok {
						t.Errorf("Expected panic message to be string, got %T", r)
						return
					}
					if !strings.Contains(panicMsg, tt.expectedPanicMsg) {
						t.Errorf(
							"Expected panic message to contain %q, got %q",
							tt.expectedPanicMsg,
							panicMsg,
						)
					}
					return
				}
				if r != nil {
					t.Errorf("Unexpected panic: %v", r)
					return
				}
			}()

			// Test both ParseConfig and MustParseConfig
			got := MustParseConfig(tmpPath, tt.secretsMap)

			// Normalize the kind for comparison
			got.Kind = got.Kind.normalize()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Config parsing = %v, want %v", got, tt.want)
			}

			// Also verify ParseConfig returns the same result
			gotFromParse, err := ParseConfig(tmpPath, tt.secretsMap)
			if err != nil {
				t.Errorf("ParseConfig() unexpected error: %v", err)
				return
			}
			if gotFromParse != nil {
				gotFromParse.Kind = gotFromParse.Kind.normalize()
				if !reflect.DeepEqual(*gotFromParse, tt.want) {
					t.Errorf("ParseConfig() = %v, want %v", *gotFromParse, tt.want)
				}
			}
		})
	}
}
