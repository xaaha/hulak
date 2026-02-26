package yamlparser

import (
	"strings"
	"testing"
)

func TestAPICallFileIsValidForGraphQL(t *testing.T) {
	tests := []struct {
		name        string
		file        *APICallFile
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
			file:        &APICallFile{},
			wantValid:   false,
			wantErr:     true,
			errContains: "URL",
		},
		{
			name: "invalid HTTP method",
			file: &APICallFile{
				URL:    "https://example.com/graphql",
				Method: "INVALID",
			},
			wantValid:   false,
			wantErr:     true,
			errContains: "invalid HTTP method",
		},
		{
			name: "valid with defaults applied - no body required",
			file: &APICallFile{
				URL: "https://example.com/graphql",
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "valid with explicit method",
			file: &APICallFile{
				URL:    "https://example.com/graphql",
				Method: "post",
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "valid with GET method",
			file: &APICallFile{
				URL:    "https://example.com/graphql",
				Method: "GET",
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "valid with existing headers",
			file: &APICallFile{
				URL:     "https://example.com/graphql",
				Headers: map[string]string{"authorization": "Bearer token"},
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "valid with existing content-type - should not override",
			file: &APICallFile{
				URL:     "https://example.com/graphql",
				Headers: map[string]string{"content-type": "application/graphql"},
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "valid with body - body is allowed but not required",
			file: &APICallFile{
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
			file: &APICallFile{
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

func TestAPICallFileIsValidForGraphQLAppliesDefaults(t *testing.T) {
	t.Run("applies default POST method", func(t *testing.T) {
		file := &APICallFile{
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
		file := &APICallFile{
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
		file := &APICallFile{
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
		file := &APICallFile{
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

func TestAPICallFilePrepareGraphQLStruct(t *testing.T) {
	tests := []struct {
		name        string
		file        *APICallFile
		checkResult func(t *testing.T, result APIInfo)
	}{
		{
			name: "basic GraphQL request",
			file: &APICallFile{
				URL:     "https://example.com/graphql",
				Method:  POST,
				Headers: map[string]string{"content-type": "application/json"},
			},
			checkResult: func(t *testing.T, result APIInfo) {
				if result.Method != "POST" {
					t.Errorf("expected method POST, got %s", result.Method)
				}
				if result.URL != "https://example.com/graphql" {
					t.Errorf("expected URL https://example.com/graphql, got %s", result.URL)
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
			file: &APICallFile{
				URL:       "https://example.com/graphql",
				Method:    POST,
				Headers:   map[string]string{"content-type": "application/json"},
				URLParams: map[string]string{"env": "prod"},
			},
			checkResult: func(t *testing.T, result APIInfo) {
				if result.URLParams["env"] != "prod" {
					t.Errorf("expected URLParams to contain env=prod, got %v", result.URLParams)
				}
			},
		},
		{
			name: "GraphQL with auth header",
			file: &APICallFile{
				URL:    "https://example.com/graphql",
				Method: POST,
				Headers: map[string]string{
					"content-type":  "application/json",
					"authorization": "Bearer token123",
				},
			},
			checkResult: func(t *testing.T, result APIInfo) {
				if result.Headers["authorization"] != "Bearer token123" {
					t.Errorf("expected authorization header, got %v", result.Headers)
				}
			},
		},
		{
			name: "body is always nil in GraphQL struct",
			file: &APICallFile{
				URL:    "https://example.com/graphql",
				Method: POST,
				Body: &Body{
					Graphql: &GraphQl{
						Query: "{ users { id } }",
					},
				},
			},
			checkResult: func(t *testing.T, result APIInfo) {
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
