package apicalls

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/yamlparser"
)

// captureStdout swaps os.Stdout for a pipe, runs fn, and returns what was
// written. Restores stdout even if fn panics.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()
	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	return buf.String()
}

func TestPrintDryRun_MethodAndURL(t *testing.T) {
	info := &yamlparser.APIInfo{
		Method:    "POST",
		URL:       "https://api.example.com/users",
		URLParams: map[string]string{"limit": "10"},
	}
	out := captureStdout(t, func() {
		if err := PrintDryRun(info, false); err != nil {
			t.Fatalf("PrintDryRun: %v", err)
		}
	})
	if !strings.HasPrefix(out, "POST https://api.example.com/users?limit=10") {
		t.Errorf("expected first line to be method + URL with params, got:\n%s", out)
	}
}

func TestPrintDryRun_HeadersSortedAndRedacted(t *testing.T) {
	info := &yamlparser.APIInfo{
		Method: "GET",
		URL:    "https://api.example.com/x",
		Headers: map[string]string{
			"Authorization": "Bearer secret123",
			"Accept":        "application/json",
			"Content-Type":  "application/json",
			"X-API-Key":     "key-456",
		},
	}
	out := captureStdout(t, func() {
		if err := PrintDryRun(info, false); err != nil {
			t.Fatalf("PrintDryRun: %v", err)
		}
	})

	if !strings.Contains(out, "Authorization: ••••") {
		t.Errorf("Authorization should be masked, got:\n%s", out)
	}
	if !strings.Contains(out, "X-API-Key: ••••") {
		t.Errorf("X-API-Key should be masked, got:\n%s", out)
	}
	if !strings.Contains(out, "Accept: application/json") {
		t.Errorf("Accept should be unmasked, got:\n%s", out)
	}
	if strings.Contains(out, "Bearer secret123") {
		t.Errorf("Bearer token leaked into output:\n%s", out)
	}

	// Headers should be sorted alphabetically for deterministic output.
	acceptIdx := strings.Index(out, "Accept:")
	authIdx := strings.Index(out, "Authorization:")
	ctIdx := strings.Index(out, "Content-Type:")
	xkeyIdx := strings.Index(out, "X-API-Key:")
	if acceptIdx >= authIdx || authIdx >= ctIdx || ctIdx >= xkeyIdx {
		t.Errorf("headers not in alphabetical order:\n%s", out)
	}
}

func TestPrintDryRun_ShowReveals(t *testing.T) {
	info := &yamlparser.APIInfo{
		Method: "GET",
		URL:    "https://api.example.com/x",
		Headers: map[string]string{
			"Authorization": "Bearer secret123",
		},
	}
	out := captureStdout(t, func() {
		if err := PrintDryRun(info, true); err != nil {
			t.Fatalf("PrintDryRun: %v", err)
		}
	})
	if !strings.Contains(out, "Authorization: Bearer secret123") {
		t.Errorf("show=true should reveal Authorization, got:\n%s", out)
	}
	if strings.Contains(out, "••••") {
		t.Errorf("show=true should not mask anything, got:\n%s", out)
	}
}

func TestPrintDryRun_JSONBodyPrettyPrinted(t *testing.T) {
	info := &yamlparser.APIInfo{
		Method: "POST",
		URL:    "https://api.example.com/x",
		Body:   strings.NewReader(`{"name":"alice","age":42}`),
	}
	out := captureStdout(t, func() {
		if err := PrintDryRun(info, false); err != nil {
			t.Fatalf("PrintDryRun: %v", err)
		}
	})
	if !strings.Contains(out, "\"name\": \"alice\"") {
		t.Errorf("JSON body should be pretty-printed with spaces, got:\n%s", out)
	}
	if !strings.Contains(out, "\n  \"age\": 42") {
		t.Errorf("JSON body should be indented, got:\n%s", out)
	}
}

func TestPrintDryRun_NonJSONBodyVerbatim(t *testing.T) {
	body := "name=alice&age=42"
	info := &yamlparser.APIInfo{
		Method: "POST",
		URL:    "https://api.example.com/x",
		Body:   strings.NewReader(body),
	}
	out := captureStdout(t, func() {
		if err := PrintDryRun(info, false); err != nil {
			t.Fatalf("PrintDryRun: %v", err)
		}
	})
	if !strings.Contains(out, body) {
		t.Errorf("non-JSON body should appear verbatim, got:\n%s", out)
	}
}

func TestPrintDryRun_NilBody(t *testing.T) {
	info := &yamlparser.APIInfo{
		Method: "GET",
		URL:    "https://api.example.com/x",
		Body:   nil,
	}
	out := captureStdout(t, func() {
		if err := PrintDryRun(info, false); err != nil {
			t.Fatalf("PrintDryRun: %v", err)
		}
	})
	// No blank line then body — last line should be the method/URL since
	// there are no headers in this test.
	if strings.Contains(out, "\n\n") {
		t.Errorf("nil body should not emit blank line + body block, got:\n%s", out)
	}
}

func TestPrintDryRun_EmptyBody(t *testing.T) {
	info := &yamlparser.APIInfo{
		Method: "GET",
		URL:    "https://api.example.com/x",
		Body:   strings.NewReader(""),
	}
	out := captureStdout(t, func() {
		if err := PrintDryRun(info, false); err != nil {
			t.Fatalf("PrintDryRun: %v", err)
		}
	})
	if strings.Contains(out, "\n\n") {
		t.Errorf("empty body should not emit blank line + body block, got:\n%s", out)
	}
}
