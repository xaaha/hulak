package mcp

import (
	"slices"
	"testing"
)

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
		{"at sign in the password", "https://user:p@ss@api.example.com/a", "api.example.com"},
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

func TestRequestAuth_259(t *testing.T) {
	tests := []struct {
		name    string
		kind    string
		headers map[string]any
		want    string
	}{
		{"oauth2 kind", "Auth", nil, "oauth2"},
		{"no headers", "API", nil, ""},
		{"no credential header", "API", map[string]any{"content-type": "application/json"}, ""},
		{"bearer literal", "API", map[string]any{"authorization": "Bearer abc123"}, "bearer"},
		{"bearer from env", "API", map[string]any{"authorization": "Bearer {{.token}}"}, "bearer"},
		{"bearer from getValueOf", "API",
			map[string]any{"authorization": `Bearer {{getValueOf "access_token" "get_m2m_token"}}`}, "bearer from get_m2m_token"},
		{"getValueOf in another spelling with a path", "GraphQL",
			map[string]any{"authorization": "Bearer {{ get_value_of `access_token` `auth/login.hk.yaml` }}"}, "bearer from auth/login.hk.yaml"},
		{"basic literal", "API", map[string]any{"authorization": "Basic dXNlcjpwdw=="}, "basic"},
		{"basicAuth action", "API", map[string]any{"authorization": "{{basicAuth .user .pass}}"}, "basic"},
		{"whole value from env", "API", map[string]any{"authorization": "{{.auth_header}}"}, "header authorization"},
		{"api key header", "GraphQL", map[string]any{"x-api-key": "{{.appsync_api_key}}"}, "header x-api-key"},
		{"cookie from getValueOf", "API",
			map[string]any{"cookie": `{{getValueOf "session" "login"}}`}, "header cookie from login"},
		{"authorization wins over others", "API",
			map[string]any{"x-api-key": "k", "authorization": "Bearer t"}, "bearer"},
		{"first credential header by name", "API",
			map[string]any{"x-auth-token": "a", "cookie": "b"}, "header cookie"},
		{"mixed case header name", "API", map[string]any{"Authorization": "bearer t"}, "bearer"},
		{"non-string value", "API", map[string]any{"authorization": 42}, "header authorization"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			doc := map[string]any{}
			if tc.headers != nil {
				doc["headers"] = tc.headers
			}
			if got := requestAuth(tc.kind, doc); got != tc.want {
				t.Errorf("requestAuth(%s, %v) = %q, want %q", tc.kind, tc.headers, got, tc.want)
			}
		})
	}
}

func TestSplitActionArgs_259(t *testing.T) {
	tests := []struct {
		body string
		want []string
	}{
		{`getValueOf "a" "b"`, []string{"getValueOf", "a", "b"}},
		{"getValueOf  `a`   'b c'", []string{"getValueOf", "a", "b c"}},
		{`getValueOf "a \"q\"" "b"`, []string{"getValueOf", `a \"q\"`, "b"}},
		{`getValueOf "unterminated`, []string{"getValueOf", "unterminated"}},
		{"", nil},
	}
	for _, tc := range tests {
		t.Run(tc.body, func(t *testing.T) {
			if got := splitActionArgs(tc.body); !slices.Equal(got, tc.want) {
				t.Errorf("splitActionArgs(%q) = %q, want %q", tc.body, got, tc.want)
			}
		})
	}
}

func TestResolvedHost_259(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		want   string
		wantOK bool
	}{
		{"plain", "https://api.example.com/v1?x=1", "api.example.com", true},
		{"port", "http://localhost:8080/health", "localhost:8080", true},
		{"userinfo", "https://svc:pw@api.example.com/v1", "api.example.com", true},
		{"at sign in the password", "https://svc:p@ss@api.example.com", "api.example.com", true},
		{"no scheme", "api.example.com/v1", "api.example.com", true},
		{"slash in the password", "https://svc:12/secret@api.example.com", "", false},
		{"invalid port from a slash in the password", "https://svc:ab/secret@api.example.com", "", false},
		{"query mark in the password", "https://svc:pa?secret@api.example.com", "", false},
		{"fragment mark in the password", "https://svc:pa#secret@api.example.com", "", false},
		{"template text in the password", "https://admin:{{pw@api.example.com/v1?token=abc", "", false},
		{"at sign in the path", "https://api.example.com/users/a@b.com", "", false},
		{"empty", "", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := resolvedHost(tc.text)
			if got != tc.want || ok != tc.wantOK {
				t.Errorf("resolvedHost(%q) = %q, %v, want %q, %v", tc.text, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
