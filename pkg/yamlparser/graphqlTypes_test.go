package yamlparser

import (
	"strings"
	"testing"
)

func TestApiCallFileIsValidForGraphQL(t *testing.T) {
	tests := []struct {
		name        string
		file        *ApiCallFile
		wantValid   bool
		wantErr     bool
		errContains string
	}{
		{
			name:        "nil file",
			file:        nil,
			wantValid:   false,
			wantErr:     true,
			errContains: "not valid",
		},
		{
			name:        "missing URL",
			file:        &ApiCallFile{},
			wantValid:   false,
			wantErr:     true,
			errContains: "URL",
		},
		{
			name: "invalid HTTP method",
			file: &ApiCallFile{
				URL:    "https://example.com/graphql",
				Method: "INVALID",
			},
			wantValid:   false,
			wantErr:     true,
			errContains: "invalid HTTP method",
		},
		{
			name: "valid with defaults applied - no body required",
			file: &ApiCallFile{
				URL: "https://example.com/graphql",
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "valid with explicit method",
			file: &ApiCallFile{
				URL:    "https://example.com/graphql",
				Method: "post",
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "valid with GET method",
			file: &ApiCallFile{
				URL:    "https://example.com/graphql",
				Method: "GET",
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "valid with existing headers",
			file: &ApiCallFile{
				URL:     "https://example.com/graphql",
				Headers: map[string]string{"authorization": "Bearer token"},
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "valid with existing content-type - should not override",
			file: &ApiCallFile{
				URL:     "https://example.com/graphql",
				Headers: map[string]string{"content-type": "application/graphql"},
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "valid with body - body is allowed but not required",
			file: &ApiCallFile{
				URL: "https://example.com/graphql",
				Body: &Body{
					Graphql: &GraphQl{
						Query: "{ users { id } }",
					},
				},
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "valid with urlparams",
			file: &ApiCallFile{
				URL:       "https://example.com/graphql",
				URLParams: map[string]string{"env": "prod"},
			},
			wantValid: true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := tt.file.IsValidForGraphQL("test.yaml")

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
					return
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if valid != tt.wantValid {
				t.Errorf("expected valid=%v, got valid=%v", tt.wantValid, valid)
			}
		})
	}
}

func TestApiCallFileIsValidForGraphQLAppliesDefaults(t *testing.T) {
	t.Run("applies default POST method", func(t *testing.T) {
		file := &ApiCallFile{
			URL: "https://example.com/graphql",
		}

		valid, err := file.IsValidForGraphQL("test.yaml")
		if !valid || err != nil {
			t.Fatalf("expected valid file, got valid=%v, err=%v", valid, err)
		}

		if file.Method != POST {
			t.Errorf("expected method to be POST, got %s", file.Method)
		}
	})

	t.Run("applies default Content-Type header", func(t *testing.T) {
		file := &ApiCallFile{
			URL: "https://example.com/graphql",
		}

		valid, err := file.IsValidForGraphQL("test.yaml")
		if !valid || err != nil {
			t.Fatalf("expected valid file, got valid=%v, err=%v", valid, err)
		}

		contentType, exists := file.Headers["content-type"]
		if !exists {
			t.Error("expected content-type header to be set")
		}
		if contentType != "application/json" {
			t.Errorf("expected content-type to be 'application/json', got %q", contentType)
		}
	})

	t.Run("uppercases method", func(t *testing.T) {
		file := &ApiCallFile{
			URL:    "https://example.com/graphql",
			Method: "post",
		}

		valid, err := file.IsValidForGraphQL("test.yaml")
		if !valid || err != nil {
			t.Fatalf("expected valid file, got valid=%v, err=%v", valid, err)
		}

		if file.Method != POST {
			t.Errorf("expected method to be uppercase POST, got %s", file.Method)
		}
	})

	t.Run("initializes nil headers map", func(t *testing.T) {
		file := &ApiCallFile{
			URL: "https://example.com/graphql",
		}

		valid, err := file.IsValidForGraphQL("test.yaml")
		if !valid || err != nil {
			t.Fatalf("expected valid file, got valid=%v, err=%v", valid, err)
		}

		if file.Headers == nil {
			t.Error("expected headers to be initialized")
		}
	})
}

func TestApiCallFilePrepareGraphQLStruct(t *testing.T) {
	tests := []struct {
		name        string
		file        *ApiCallFile
		checkResult func(t *testing.T, result ApiInfo)
	}{
		{
			name: "basic GraphQL request",
			file: &ApiCallFile{
				URL:     "https://example.com/graphql",
				Method:  POST,
				Headers: map[string]string{"content-type": "application/json"},
			},
			checkResult: func(t *testing.T, result ApiInfo) {
				if result.Method != "POST" {
					t.Errorf("expected method POST, got %s", result.Method)
				}
				if result.Url != "https://example.com/graphql" {
					t.Errorf("expected URL https://example.com/graphql, got %s", result.Url)
				}
				if result.Headers["content-type"] != "application/json" {
					t.Errorf("expected content-type header, got %v", result.Headers)
				}
				if result.Body != nil {
					t.Error("expected body to be nil for GraphQL struct")
				}
			},
		},
		{
			name: "GraphQL with URLParams",
			file: &ApiCallFile{
				URL:       "https://example.com/graphql",
				Method:    POST,
				Headers:   map[string]string{"content-type": "application/json"},
				URLParams: map[string]string{"env": "prod"},
			},
			checkResult: func(t *testing.T, result ApiInfo) {
				if result.UrlParams["env"] != "prod" {
					t.Errorf("expected URLParams to contain env=prod, got %v", result.UrlParams)
				}
			},
		},
		{
			name: "GraphQL with auth header",
			file: &ApiCallFile{
				URL:    "https://example.com/graphql",
				Method: POST,
				Headers: map[string]string{
					"content-type":  "application/json",
					"authorization": "Bearer token123",
				},
			},
			checkResult: func(t *testing.T, result ApiInfo) {
				if result.Headers["authorization"] != "Bearer token123" {
					t.Errorf("expected authorization header, got %v", result.Headers)
				}
			},
		},
		{
			name: "body is always nil in GraphQL struct",
			file: &ApiCallFile{
				URL:    "https://example.com/graphql",
				Method: POST,
				Body: &Body{
					Graphql: &GraphQl{
						Query: "{ users { id } }",
					},
				},
			},
			checkResult: func(t *testing.T, result ApiInfo) {
				// Body should be nil because GraphQL query is provided separately
				if result.Body != nil {
					t.Error("expected body to be nil for GraphQL struct")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.file.PrepareGraphQLStruct()

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}
