package apicalls

import (
	"bytes"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils/testutil"
	"github.com/xaaha/hulak/pkg/yamlparser"
)

func TestPrintDryRun_MethodAndURL(t *testing.T) {
	info := &yamlparser.APIInfo{
		Method:    "POST",
		URL:       "https://api.example.com/users",
		URLParams: map[string]string{"limit": "10"},
	}
	out := testutil.CaptureStdout(t, func() {
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
	out := testutil.CaptureStdout(t, func() {
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
	out := testutil.CaptureStdout(t, func() {
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
	out := testutil.CaptureStdout(t, func() {
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
	out := testutil.CaptureStdout(t, func() {
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
	out := testutil.CaptureStdout(t, func() {
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

func TestPrintDryRun_URLEncodedBodyPretty(t *testing.T) {
	info := &yamlparser.APIInfo{
		Method:  "POST",
		URL:     "https://api.example.com/x",
		Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
		Body:    strings.NewReader("user=Jane+Doe&age=42"),
	}
	out := testutil.CaptureStdout(t, func() {
		if err := PrintDryRun(info, false); err != nil {
			t.Fatalf("PrintDryRun: %v", err)
		}
	})
	if !strings.Contains(out, "age: 42") {
		t.Errorf("urlencoded body should render key: value, got:\n%s", out)
	}
	if !strings.Contains(out, "user: Jane Doe") {
		t.Errorf("urlencoded body should URL-decode values, got:\n%s", out)
	}
	if strings.Contains(out, "user=Jane+Doe") {
		t.Errorf("raw urlencoded bytes should not leak when pretty-printed, got:\n%s", out)
	}
}

func TestPrintDryRun_MultipartBodyPretty(t *testing.T) {
	// Build a real multipart body via mime/multipart so the boundary in the
	// header matches the bytes in the body.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("product", "hulak"); err != nil {
		t.Fatal(err)
	}
	if err := mw.WriteField("user", "Jane Doe"); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	info := &yamlparser.APIInfo{
		Method:  "POST",
		URL:     "https://api.example.com/x",
		Headers: map[string]string{"Content-Type": mw.FormDataContentType()},
		Body:    &buf,
	}
	out := testutil.CaptureStdout(t, func() {
		if err := PrintDryRun(info, false); err != nil {
			t.Fatalf("PrintDryRun: %v", err)
		}
	})
	if !strings.Contains(out, "product: hulak") {
		t.Errorf("multipart body should render key: value, got:\n%s", out)
	}
	if !strings.Contains(out, "user: Jane Doe") {
		t.Errorf("multipart body should render second field, got:\n%s", out)
	}
	if strings.Contains(out, "Content-Disposition") {
		t.Errorf("raw multipart envelope should not leak when pretty-printed, got:\n%s", out)
	}
}

// TestPrintDryRun_MultipartMixedTextAndFile exercises a multipart body that
// contains both a text field and a file part in the same envelope. File
// parts must render as a "<file: name, N bytes>" summary, not raw bytes.
func TestPrintDryRun_MultipartMixedTextAndFile(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("title", "report"); err != nil {
		t.Fatal(err)
	}
	fileWriter, err := mw.CreateFormFile("attachment", "data.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fileWriter.Write([]byte{0x00, 0x01, 0x02, 0x03, 0x04}); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	info := &yamlparser.APIInfo{
		Method:  "POST",
		URL:     "https://api.example.com/x",
		Headers: map[string]string{"Content-Type": mw.FormDataContentType()},
		Body:    &buf,
	}
	out := testutil.CaptureStdout(t, func() {
		if err := PrintDryRun(info, false); err != nil {
			t.Fatalf("PrintDryRun: %v", err)
		}
	})
	if !strings.Contains(out, "title: report") {
		t.Errorf("text field should render verbatim, got:\n%s", out)
	}
	if !strings.Contains(out, "attachment: <file: data.bin, 5 bytes>") {
		t.Errorf("file part should render as summary, got:\n%s", out)
	}
	if strings.Contains(out, "\x00\x01\x02") {
		t.Errorf("raw binary bytes leaked into output:\n%s", out)
	}
}

func TestPrintDryRun_EmptyBody(t *testing.T) {
	info := &yamlparser.APIInfo{
		Method: "GET",
		URL:    "https://api.example.com/x",
		Body:   strings.NewReader(""),
	}
	out := testutil.CaptureStdout(t, func() {
		if err := PrintDryRun(info, false); err != nil {
			t.Fatalf("PrintDryRun: %v", err)
		}
	})
	if strings.Contains(out, "\n\n") {
		t.Errorf("empty body should not emit blank line + body block, got:\n%s", out)
	}
}
