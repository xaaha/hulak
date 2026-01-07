package curl

import (
	"strings"
	"testing"
)

func TestParseCurlCommand(t *testing.T) {
	tests := []struct {
		name        string
		curlCmd     string
		wantMethod  string
		wantURL     string
		wantErr     bool
		checkBody   bool
		wantBody    string
		checkParams bool
		wantParams  map[string]string
	}{
		{
			name:       "Simple GET request",
			curlCmd:    "curl https://api.example.com/users",
			wantMethod: "GET",
			wantURL:    "https://api.example.com/users",
			wantErr:    false,
		},
		{
			name:       "GET with curl keyword",
			curlCmd:    "curl https://jsonplaceholder.typicode.com/todos/1",
			wantMethod: "GET",
			wantURL:    "https://jsonplaceholder.typicode.com/todos/1",
			wantErr:    false,
		},
		{
			name:       "POST with method flag",
			curlCmd:    "curl -X POST https://api.example.com/posts",
			wantMethod: "POST",
			wantURL:    "https://api.example.com/posts",
			wantErr:    false,
		},
		{
			name:        "GET with URL parameters",
			curlCmd:     "curl 'https://api.example.com/search?q=test&page=1'",
			wantMethod:  "GET",
			wantURL:     "https://api.example.com/search",
			checkParams: true,
			wantParams:  map[string]string{"q": "test", "page": "1"},
			wantErr:     false,
		},
		{
			name:       "POST with JSON body",
			curlCmd:    `curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d '{"name":"John","age":30}'`,
			wantMethod: "POST",
			wantURL:    "https://api.example.com/users",
			checkBody:  true,
			wantBody:   `{"name":"John","age":30}`,
			wantErr:    false,
		},
		{
			name:       "Multi-line curl with backslashes",
			curlCmd:    "curl -X POST \\\nhttps://api.example.com/data \\\n-H \"Authorization: Bearer token\"",
			wantMethod: "POST",
			wantURL:    "https://api.example.com/data",
			wantErr:    false,
		},
		{
			name:    "Empty curl command",
			curlCmd: "",
			wantErr: true,
		},
		{
			name:    "No URL in command",
			curlCmd: "curl -X GET",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseCurlCommand(tc.curlCmd)

			if (err != nil) != tc.wantErr {
				t.Errorf("ParseCurlCommand() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if tc.wantErr {
				return // If we expected an error, we're done
			}

			if result.Method != tc.wantMethod {
				t.Errorf("Method = %v, want %v", result.Method, tc.wantMethod)
			}

			if result.URL != tc.wantURL {
				t.Errorf("URL = %v, want %v", result.URL, tc.wantURL)
			}

			if tc.checkBody && result.Body != tc.wantBody {
				t.Errorf("Body = %v, want %v", result.Body, tc.wantBody)
			}

			if tc.checkParams {
				for key, expectedVal := range tc.wantParams {
					if actualVal, ok := result.URLParams[key]; !ok || actualVal != expectedVal {
						t.Errorf("URLParams[%s] = %v, want %v", key, actualVal, expectedVal)
					}
				}
			}
		})
	}
}

func TestExtractHeaders(t *testing.T) {
	tests := []struct {
		name        string
		curlCmd     string
		wantHeaders map[string]string
	}{
		{
			name:    "Single header",
			curlCmd: `curl -H "Content-Type: application/json" https://api.example.com`,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:    "Multiple headers",
			curlCmd: `curl -H "Content-Type: application/json" -H "Authorization: Bearer token123" https://api.example.com`,
			wantHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token123",
			},
		},
		{
			name:        "No headers",
			curlCmd:     "curl https://api.example.com",
			wantHeaders: map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			headers := extractHeaders(tc.curlCmd)

			if len(headers) != len(tc.wantHeaders) {
				t.Errorf("Got %d headers, want %d", len(headers), len(tc.wantHeaders))
			}

			for key, expectedVal := range tc.wantHeaders {
				if actualVal, ok := headers[key]; !ok || actualVal != expectedVal {
					t.Errorf("Headers[%s] = %v, want %v", key, actualVal, expectedVal)
				}
			}
		})
	}
}

func TestExtractMethod(t *testing.T) {
	tests := []struct {
		name       string
		curlCmd    string
		wantMethod string
	}{
		{
			name:       "Method with -X flag",
			curlCmd:    "curl -X POST https://api.example.com",
			wantMethod: "POST",
		},
		{
			name:       "Method with --request flag",
			curlCmd:    "curl --request PUT https://api.example.com",
			wantMethod: "PUT",
		},
		{
			name:       "No method specified",
			curlCmd:    "curl https://api.example.com",
			wantMethod: "",
		},
		{
			name:       "Method lowercase",
			curlCmd:    "curl -X post https://api.example.com",
			wantMethod: "POST",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			method := extractMethod(tc.curlCmd)
			if method != tc.wantMethod {
				t.Errorf("extractMethod() = %v, want %v", method, tc.wantMethod)
			}
		})
	}
}

func TestExtractFormData(t *testing.T) {
	tests := []struct {
		name         string
		curlCmd      string
		wantFormData map[string]string
	}{
		{
			name:    "Single form field",
			curlCmd: `curl -F "username=john" https://api.example.com`,
			wantFormData: map[string]string{
				"username": "john",
			},
		},
		{
			name:    "Multiple form fields",
			curlCmd: `curl -F "username=john" -F "password=secret" https://api.example.com`,
			wantFormData: map[string]string{
				"username": "john",
				"password": "secret",
			},
		},
		{
			name:         "No form data",
			curlCmd:      "curl https://api.example.com",
			wantFormData: map[string]string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			formData := extractFormData(tc.curlCmd)

			if len(formData) != len(tc.wantFormData) {
				t.Errorf("Got %d form fields, want %d", len(formData), len(tc.wantFormData))
			}

			for key, expectedVal := range tc.wantFormData {
				if actualVal, ok := formData[key]; !ok || actualVal != expectedVal {
					t.Errorf("FormData[%s] = %v, want %v", key, actualVal, expectedVal)
				}
			}
		})
	}
}

func TestIsGraphQLBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "Valid GraphQL body",
			body: `{"query":"query Hello { hello }","variables":{}}`,
			want: true,
		},
		{
			name: "GraphQL with single quotes",
			body: `{'query':'query Hello { hello }'}`,
			want: true,
		},
		{
			name: "Not GraphQL - regular JSON",
			body: `{"name":"John","age":30}`,
			want: false,
		},
		{
			name: "Not GraphQL - plain text",
			body: "username=john&password=secret",
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isGraphQLBody(tc.body)
			if result != tc.want {
				t.Errorf("isGraphQLBody() = %v, want %v", result, tc.want)
			}
		})
	}
}

func TestCleanCurlString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Remove curl prefix",
			input: "curl https://example.com",
			want:  "https://example.com",
		},
		{
			name:  "Handle multi-line with backslashes",
			input: "curl \\\nhttps://example.com \\\n-H 'test'",
			want:  "https://example.com -H 'test'",
		},
		{
			name:  "Normalize whitespace",
			input: "curl    https://example.com     -X    POST",
			want:  "https://example.com -X POST",
		},
		{
			name:  "Already clean",
			input: "https://example.com -X POST",
			want:  "https://example.com -X POST",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := cleanCurlString(tc.input)
			if result != tc.want {
				t.Errorf("cleanCurlString() = %q, want %q", result, tc.want)
			}
		})
	}
}

func TestBasicAuth(t *testing.T) {
	curlCmd := "curl -u user:password https://api.example.com"
	result, err := ParseCurlCommand(curlCmd)

	if err != nil {
		t.Fatalf("ParseCurlCommand() error = %v", err)
	}

	auth, ok := result.Headers["Authorization"]
	if !ok {
		t.Fatal("Authorization header not found")
	}

	if !strings.HasPrefix(auth, "Basic ") {
		t.Errorf("Authorization header should start with 'Basic ', got %v", auth)
	}
}

func TestCookies(t *testing.T) {
	curlCmd := `curl --cookie "session=abc123" https://api.example.com`
	result, err := ParseCurlCommand(curlCmd)

	if err != nil {
		t.Fatalf("ParseCurlCommand() error = %v", err)
	}

	cookie, ok := result.Headers["Cookie"]
	if !ok {
		t.Fatal("Cookie header not found")
	}

	if cookie != "session=abc123" {
		t.Errorf("Cookie = %v, want session=abc123", cookie)
	}
}
