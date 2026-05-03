package apicalls

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEvalAndWriteRes(t *testing.T) {
	tempDir := t.TempDir() // Create a temporary directory for tests

	tests := []struct {
		name         string
		resBody      string
		contentType  string
		expectedFile string
		expectedExt  string
	}{
		{
			name:         "JSON file creation",
			resBody:      `{"key": "value"}`,
			contentType:  "application/json",
			expectedFile: "test.json",
			expectedExt:  ".json",
		},
		{
			name:         "XML file creation",
			resBody:      `<root><key>value</key></root>`,
			contentType:  "application/xml",
			expectedFile: "test.xml",
			expectedExt:  ".xml",
		},
		{
			name:         "HTML file creation",
			resBody:      `<!DOCTYPE html><html><body>Hello</body></html>`,
			contentType:  "text/html",
			expectedFile: "test.html",
			expectedExt:  ".html",
		},
		{
			name:         "Plain text file creation",
			resBody:      "This is plain text",
			contentType:  "text/plain",
			expectedFile: "test.txt",
			expectedExt:  ".txt",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "test")

			if err := evalAndWriteRes(tc.resBody, tc.contentType, filePath); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expectedFileName := "test_response" + tc.expectedExt
			expectedPath := filepath.Join(tempDir, expectedFileName)
			if _, err := os.Stat(expectedPath); err != nil {
				t.Errorf("Expected file %s to be created, but it was not", expectedPath)
			}

			_ = os.Remove(expectedPath)
		})
	}

	t.Run("Invalid inputs should not create files", func(t *testing.T) {
		err := evalAndWriteRes("", "", "")
		if err == nil {
			t.Fatal("Expected Error but did not get it")
		}
	})
}

// TestEvalAndWriteRes_ReadOnlyDir is a regression for #205 — write failures
// used to be silently swallowed, leaving the user with a "success" outcome
// and no response file on disk. Skipped on Windows because chmod 0o555 on a
// directory does not block writes there the way it does on POSIX.
func TestEvalAndWriteRes_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only dir semantics differ on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses dir mode bits")
	}

	tempDir := t.TempDir()
	if err := os.Chmod(tempDir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(tempDir, 0o755) })

	err := evalAndWriteRes(`{"ok":true}`, "application/json", filepath.Join(tempDir, "req"))
	if err == nil {
		t.Fatal("expected error writing to read-only dir, got nil")
	}
	if !strings.Contains(err.Error(), "saving response") {
		t.Errorf("expected error to wrap with 'saving response', got %v", err)
	}
}

// TestExtensionFor covers Content-Type → extension resolution and the body-sniff
// fallback for missing/generic headers. Regression for #208 — the previous
// implementation classified by body sniffing alone and produced wrong
// extensions for truncated JSON, JSON containing the literal `</html>`, and
// vendored JSON subtypes.
func TestExtensionFor(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		want        string
	}{
		// --- Header-based resolution ---
		{
			name:        "application/json",
			contentType: "application/json",
			body:        `{"ok":true}`,
			want:        ".json",
		},
		{
			name:        "json with charset parameter",
			contentType: "application/json; charset=utf-8",
			body:        `{"ok":true}`,
			want:        ".json",
		},
		{
			name:        "vendored json subtype",
			contentType: "application/vnd.api+json",
			body:        `{"ok":true}`,
			want:        ".json",
		},
		{
			name:        "text/json",
			contentType: "text/json",
			body:        `{"ok":true}`,
			want:        ".json",
		},
		{
			name:        "text/html",
			contentType: "text/html; charset=utf-8",
			body:        "<!DOCTYPE html><html></html>",
			want:        ".html",
		},
		{
			name:        "application/xml",
			contentType: "application/xml",
			body:        "<r/>",
			want:        ".xml",
		},
		{
			name:        "vendored xml subtype",
			contentType: "image/svg+xml",
			body:        "<svg/>",
			want:        ".xml",
		},
		{
			name:        "text/plain",
			contentType: "text/plain",
			body:        "hello",
			want:        ".txt",
		},
		{
			name:        "text/csv",
			contentType: "text/csv",
			body:        "a,b\n1,2",
			want:        ".csv",
		},
		{
			name:        "application/pdf",
			contentType: "application/pdf",
			body:        "%PDF-1.4...",
			want:        ".pdf",
		},

		// --- Generic / missing header → body sniff ---
		{
			name:        "missing content-type, JSON body",
			contentType: "",
			body:        `{"ok":true}`,
			want:        ".json",
		},
		{
			name:        "octet-stream falls back to sniff",
			contentType: "application/octet-stream",
			body:        `{"ok":true}`,
			want:        ".json",
		},
		{
			name:        "*/* falls back to sniff",
			contentType: "*/*",
			body:        `{"ok":true}`,
			want:        ".json",
		},
		{
			name:        "unknown content-type, plain body",
			contentType: "application/x-some-unknown",
			body:        "hello world",
			want:        ".txt",
		},

		// --- #208 regression cases ---
		{
			name:        "json error mentioning </html> stays json",
			contentType: "application/json",
			body:        `{"error":"got </html> in input"}`,
			want:        ".json",
		},
		{
			name:        "no header, json body containing </html>",
			contentType: "",
			body:        `{"error":"got </html> in input"}`,
			want:        ".json",
		},
		{
			name:        "truncated json with no header is text",
			contentType: "",
			body:        `{"key":"unclosed`,
			want:        ".txt",
		},
		{
			name:        "plain text containing <html> is text",
			contentType: "",
			body:        "log line: <html> appeared in input",
			want:        ".txt",
		},
		{
			name:        "html starting with doctype",
			contentType: "",
			body:        "<!DOCTYPE html><html><body>x</body></html>",
			want:        ".html",
		},
		{
			name:        "html with leading whitespace",
			contentType: "",
			body:        "\n  <html><body>x</body></html>",
			want:        ".html",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extensionFor(tc.contentType, tc.body); got != tc.want {
				t.Errorf("extensionFor(%q, %q) = %q, want %q",
					tc.contentType, tc.body, got, tc.want)
			}
		})
	}
}

// TestIsJSON_StrictValidation verifies the tightened IsJSON catches truncated
// payloads (#208). The previous json.Unmarshal-into-RawMessage check accepted
// some malformed inputs.
func TestIsJSON_StrictValidation(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"valid object", `{"a":1}`, true},
		{"valid array", `[1,2,3]`, true},
		{"valid string", `"hello"`, true},
		{"truncated object", `{"key":"unclosed`, false},
		{"truncated array", `[1,2,`, false},
		{"trailing garbage", `{"a":1}garbage`, false},
		{"empty", ``, false},
		{"plain text", `hello`, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsJSON(tc.in); got != tc.want {
				t.Errorf("IsJSON(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestIsHTML_StrictPrefix verifies IsHTML no longer matches arbitrary
// substrings — only inputs that actually start with HTML structural markers
// (regression for #208).
func TestIsHTML_StrictPrefix(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"doctype html", "<!DOCTYPE html><html></html>", true},
		{"html tag", "<html><body>x</body></html>", true},
		{"html with leading whitespace", "  \n<html></html>", true},
		{"uppercase HTML tag", "<HTML></HTML>", true},
		{"json containing </html>", `{"e":"</html>"}`, false},
		{"plain text mentioning <html>", "see <html> in logs", false},
		{"empty string", "", false},
		{"xml document", "<?xml version=\"1.0\"?><root/>", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsHTML(tc.in); got != tc.want {
				t.Errorf("IsHTML(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
