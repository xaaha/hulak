package apicalls

import "testing"

func TestFullUrl(t *testing.T) {
	tests := []struct {
		name     string
		baseUrl  string
		expected string
		params   []HeaderOrUrlParam
	}{
		// Test with no parameters
		{
			name:     "No parameters",
			baseUrl:  "https://api.example.com/resource",
			params:   []HeaderOrUrlParam{},
			expected: "https://api.example.com/resource",
		},
		// Test with a single parameter
		{
			name:    "Single parameter",
			baseUrl: "https://api.example.com/resource",
			params: []HeaderOrUrlParam{
				{"search", "golang"},
			},
			expected: "https://api.example.com/resource?search=golang",
		},
		// Test with multiple parameters
		{
			name:    "Multiple parameters",
			baseUrl: "https://api.example.com/resource",
			params: []HeaderOrUrlParam{
				{"search", "golang"},
				{"limit", "10"},
				{"sort", "desc"},
			},
			expected: "https://api.example.com/resource?limit=10&search=golang&sort=desc",
		},
		// Test with special characters in parameters
		{
			name:    "Special characters in parameters",
			baseUrl: "https://api.example.com/resource",
			params: []HeaderOrUrlParam{
				{"search", "go programming"},
				{"filter", "name&value"},
			},
			expected: "https://api.example.com/resource?filter=name%26value&search=go+programming",
		},
		// Test with empty parameter values
		{
			name:    "Empty parameter values",
			baseUrl: "https://api.example.com/resource",
			params: []HeaderOrUrlParam{
				{"search", ""},
				{"limit", "10"},
			},
			expected: "https://api.example.com/resource?limit=10&search=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullUrl := FullUrl(tt.baseUrl, tt.params...)
			if fullUrl != tt.expected {
				t.Errorf("FullUrl() = %v, want %v", fullUrl, tt.expected)
			}
		})
	}
}
