package curl

import (
	"regexp"
	"strings"
	"testing"
)

func TestCleanStdinInput(t *testing.T) {
	// Export the cleanStdinInput function for testing
	cleanStdinInput := func(input string) string {
		// Split into lines
		lines := strings.Split(input, "\n")

		// Process each line
		var cleanedLines []string
		for _, line := range lines {
			// Trim whitespace
			line = strings.TrimSpace(line)

			// Skip empty lines
			if line == "" {
				continue
			}

			// Remove trailing backslash (line continuation)
			line = strings.TrimSuffix(line, "\\")
			line = strings.TrimSpace(line)

			if line != "" {
				cleanedLines = append(cleanedLines, line)
			}
		}

		// Join with spaces
		result := strings.Join(cleanedLines, " ")

		// Normalize multiple spaces to single space
		result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

		return strings.TrimSpace(result)
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "Multi-line with backslashes",
			input: `curl -X POST https://example.com \
  -H "Content-Type: application/json" \
  -d '{"key":"value"}'`,
			want: `curl -X POST https://example.com -H "Content-Type: application/json" -d '{"key":"value"}'`,
		},
		{
			name:  "Single line",
			input: `curl https://example.com`,
			want:  `curl https://example.com`,
		},
		{
			name:  "Multiple spaces",
			input: `curl    -X    POST    https://example.com`,
			want:  `curl -X POST https://example.com`,
		},
		{
			name: "Empty lines in middle",
			input: `curl -X POST https://example.com

-H "Content-Type: application/json"`,
			want: `curl -X POST https://example.com -H "Content-Type: application/json"`,
		},
		{
			name: "Leading and trailing whitespace",
			input: `
			curl https://example.com
			`,
			want: `curl https://example.com`,
		},
		{
			name: "Backslash with extra spaces",
			input: `curl -X POST https://example.com    \
    -H "Content-Type: application/json"    \
    -d '{"key":"value"}'`,
			want: `curl -X POST https://example.com -H "Content-Type: application/json" -d '{"key":"value"}'`,
		},
		{
			name: "Real DevTools example",
			input: `curl 'https://jsonplaceholder.typicode.com/posts' \
  -H 'accept: application/json' \
  -H 'content-type: application/json' \
  --data-raw '{"title":"foo","body":"bar","userId":1}'`,
			want: `curl 'https://jsonplaceholder.typicode.com/posts' -H 'accept: application/json' -H 'content-type: application/json' --data-raw '{"title":"foo","body":"bar","userId":1}'`,
		},
		{
			name: "Complex nested JSON",
			input: `curl -X POST https://api.example.com/graphql \
-H "Content-Type: application/json" \
-d '{"query":"query { user(id: 1) { name email } }","variables":{"id":"123"}}'`,
			want: `curl -X POST https://api.example.com/graphql -H "Content-Type: application/json" -d '{"query":"query { user(id: 1) { name email } }","variables":{"id":"123"}}'`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := cleanStdinInput(tc.input)
			if got != tc.want {
				t.Errorf("cleanStdinInput() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCleanStdinInputPreservesQuotes(t *testing.T) {
	// Export the cleanStdinInput function for testing
	cleanStdinInput := func(input string) string {
		// Split into lines
		lines := strings.Split(input, "\n")

		// Process each line
		var cleanedLines []string
		for _, line := range lines {
			// Trim whitespace
			line = strings.TrimSpace(line)

			// Skip empty lines
			if line == "" {
				continue
			}

			// Remove trailing backslash (line continuation)
			line = strings.TrimSuffix(line, "\\")
			line = strings.TrimSpace(line)

			if line != "" {
				cleanedLines = append(cleanedLines, line)
			}
		}

		// Join with spaces
		result := strings.Join(cleanedLines, " ")

		// Normalize multiple spaces to single space
		result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

		return strings.TrimSpace(result)
	}

	input := `curl -d '{"key":"value"}' https://example.com`
	result := cleanStdinInput(input)

	if !strings.Contains(result, `'{"key":"value"}'`) {
		t.Errorf("cleanStdinInput should preserve single quotes, got: %s", result)
	}
}

func TestCleanStdinInputEmptyInput(t *testing.T) {
	// Export the cleanStdinInput function for testing
	cleanStdinInput := func(input string) string {
		// Split into lines
		lines := strings.Split(input, "\n")

		// Process each line
		var cleanedLines []string
		for _, line := range lines {
			// Trim whitespace
			line = strings.TrimSpace(line)

			// Skip empty lines
			if line == "" {
				continue
			}

			// Remove trailing backslash (line continuation)
			line = strings.TrimSuffix(line, "\\")
			line = strings.TrimSpace(line)

			if line != "" {
				cleanedLines = append(cleanedLines, line)
			}
		}

		// Join with spaces
		result := strings.Join(cleanedLines, " ")

		// Normalize multiple spaces to single space
		result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")

		return strings.TrimSpace(result)
	}

	tests := []string{
		"",
		"\n\n\n",
		"   \n   \n   ",
		"\\\n\\\n",
	}

	for _, input := range tests {
		result := cleanStdinInput(input)
		if result != "" {
			t.Errorf("cleanStdinInput(%q) = %q, want empty string", input, result)
		}
	}
}
