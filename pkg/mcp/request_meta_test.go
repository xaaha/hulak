package mcp

import "testing"

func TestRawHost_259(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{"literal with path and query", "https://api.example.com/v1/users?x=1", "api.example.com"},
		{"query without path", "https://api.example.com?x=1", "api.example.com"},
		{"fragment", "https://api.example.com#top", "api.example.com"},
		{"port", "http://localhost:8080/health", "localhost:8080"},
		{"userinfo", "https://user:pw@api.example.com/a", "api.example.com"},
		{"no scheme", "api.example.com/a", "api.example.com"},
		{"template only", "{{.base_url}}", "{{.base_url}}"},
		{"template prefix", "{{.base_url}}/v1/users", "{{.base_url}}"},
		{"template inside host", "https://{{.sub}}.example.com/a", "{{.sub}}.example.com"},
		{"slash inside an action", `{{getValueOf "url" "auth/login"}}/a`, `{{getValueOf "url" "auth/login"}}`},
		{"at sign inside an action", `{{getValueOf "a@b" "x"}}/a`, `{{getValueOf "a@b" "x"}}`},
		{"templated scheme", "{{.scheme}}://api.example.com/a", "api.example.com"},
		{"scheme-like text inside an action", `{{getValueOf "a://b" "x"}}/a`, `{{getValueOf "a://b" "x"}}`},
		{"surrounding space", "  https://api.example.com/a  ", "api.example.com"},
		{"empty", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := rawHost(tc.url); got != tc.want {
				t.Errorf("rawHost(%q) = %q, want %q", tc.url, got, tc.want)
			}
		})
	}
}
