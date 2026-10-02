package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestIsSensitiveHeader(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Authorization", true},
		{"authorization", true},
		{"AUTHORIZATION", true},
		{"AuThOrIzAtIoN", true},
		{"Proxy-Authorization", true},
		{"Cookie", true},
		{"Set-Cookie", true},
		{"X-API-Key", true},
		{"x-api-key", true},
		{"X-Auth-Token", true},
		{"X-Csrf-Token", true},
		{"X-Amz-Security-Token", true},
		{"Www-Authenticate", true},
		{"Proxy-Authenticate", true},
		{"Content-Type", false},
		{"Accept", false},
		{"User-Agent", false},
		{"X-Custom-Header", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSensitiveHeader(tc.name); got != tc.want {
				t.Errorf("IsSensitiveHeader(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestRedactHeaders_MasksSensitive(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer secret123",
		"Cookie":        "session=abc",
		"Content-Type":  "application/json",
		"X-API-Key":     "key-456",
		"Accept":        "*/*",
	}
	got := RedactHeaders(headers, false)

	wantMasked := []string{"Authorization", "Cookie", "X-API-Key"}
	for _, k := range wantMasked {
		if got[k] != MaskedValue {
			t.Errorf("expected %q to be masked, got %q", k, got[k])
		}
	}

	if got["Content-Type"] != "application/json" {
		t.Errorf("Content-Type should be unchanged, got %q", got["Content-Type"])
	}
	if got["Accept"] != "*/*" {
		t.Errorf("Accept should be unchanged, got %q", got["Accept"])
	}
}

func TestRedactHeaders_ShowReturnsAllAsIs(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer secret123",
		"Cookie":        "session=abc",
		"Content-Type":  "application/json",
	}
	got := RedactHeaders(headers, true)

	for k, v := range headers {
		if got[k] != v {
			t.Errorf("show=true: header %q should equal %q, got %q", k, v, got[k])
		}
	}
}

func TestRedactHeaders_CaseInsensitiveMatch(t *testing.T) {
	headers := map[string]string{
		"authorization": "Bearer x",
		"COOKIE":        "k=v",
	}
	got := RedactHeaders(headers, false)
	if got["authorization"] != MaskedValue {
		t.Errorf("lowercase authorization should be masked, got %q", got["authorization"])
	}
	if got["COOKIE"] != MaskedValue {
		t.Errorf("uppercase COOKIE should be masked, got %q", got["COOKIE"])
	}
}

func TestRedactHeaders_DoesNotMutateOriginal(t *testing.T) {
	headers := map[string]string{
		"Authorization": "Bearer secret",
	}
	_ = RedactHeaders(headers, false)
	if headers["Authorization"] != "Bearer secret" {
		t.Errorf("RedactHeaders mutated input map: %q", headers["Authorization"])
	}
}

func TestRedactHeaders_EmptyMap(t *testing.T) {
	got := RedactHeaders(map[string]string{}, false)
	if got == nil {
		t.Error("expected non-nil empty map, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %d entries", len(got))
	}
}

func TestRedactHeaders_NilMap(t *testing.T) {
	got := RedactHeaders(nil, false)
	if got == nil {
		t.Error("expected non-nil empty map, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %d entries", len(got))
	}
}

func TestD1_5_MaskShape(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  string
	}{
		{"s3cr3t-value-0123456789abcd", `^body=••••\(27 chars, #[0-9a-f]{4}\)$`},
		{"токен-значение", `^body=••••\(27 chars, #[0-9a-f]{4}\)$`},
	} {
		t.Run(tc.value, func(t *testing.T) {
			r := NewValueRedactor(map[string]any{"client_secret": tc.value}, true, nil)

			got := r.Redact("body=" + tc.value)
			want := fmt.Sprintf(
				"body=%s(%d chars, #%s)", MaskedValue, len(tc.value), fingerprint(tc.value),
			)
			if got != want {
				t.Errorf("Redact() = %q, want %q", got, want)
			}
			if !regexp.MustCompile(tc.want).MatchString(got) {
				t.Errorf("mask does not match the agreed shape: %q", got)
			}
		})
	}
}

// fingerprintProbeEnv switches the test binary into its subprocess role.
const fingerprintProbeEnv = "HULAK_FINGERPRINT_PROBE"

// Several values, so an accidental collision on one 4-hex fingerprint cannot
// make TestD1_5_SaltIsRandomPerProcess flake.
var fingerprintProbeValues = []string{
	"same-secret-value-across-processes",
	"another-secret-value-entirely",
	"a-third-secret-value-for-entropy",
}

// TestFingerprintProbe is the subprocess half of
// TestD1_5_SaltIsRandomPerProcess. It does nothing in a normal run.
func TestFingerprintProbe(t *testing.T) {
	if os.Getenv(fingerprintProbeEnv) == "" {
		t.Skip("runs only as the fingerprint subprocess")
	}
	for _, value := range fingerprintProbeValues {
		fmt.Printf("FINGERPRINT=%s\n", fingerprint(value))
	}
}

func TestD1_5_SaltIsRandomPerProcess(t *testing.T) {
	if len(fingerprintSalt) != 32 {
		t.Errorf("salt is %d bytes, want 32", len(fingerprintSalt))
	}
	if !slices.ContainsFunc(fingerprintSalt, func(b byte) bool { return b != 0 }) {
		t.Error("salt is all zero, so it is not from crypto/rand")
	}

	probe := func() string {
		//nolint:gosec // G204 re-executing this test binary is the only way to read a second process's salt
		cmd := exec.Command(os.Args[0], "-test.run=^TestFingerprintProbe$")
		cmd.Env = append(os.Environ(), fingerprintProbeEnv+"=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("fingerprint subprocess: %v\n%s", err, out)
		}
		found := regexp.MustCompile(`FINGERPRINT=([0-9a-f]{4})`).FindAllSubmatch(out, -1)
		if len(found) != len(fingerprintProbeValues) {
			t.Fatalf("subprocess printed %d fingerprints:\n%s", len(found), out)
		}
		var all []string
		for _, match := range found {
			all = append(all, string(match[1]))
		}
		return strings.Join(all, "-")
	}
	if first, second := probe(), probe(); first == second {
		t.Errorf("two processes shared the fingerprints %q, so the salt is compiled in", first)
	}
}

func TestD1_5_MinimumLengthIsEight(t *testing.T) {
	r := NewValueRedactor(map[string]any{
		"short_token":  "1234567",
		"exact_token":  "12345678",
		"longer_token": "123456789",
	}, true, nil)

	if got := r.Redact("v=1234567"); got != "v=1234567" {
		t.Errorf("7-char value must be left alone, got %q", got)
	}
	for _, v := range []string{"12345678", "123456789"} {
		if got := r.Redact("v=" + v); strings.Contains(got, v) {
			t.Errorf("value of length %d must be masked, got %q", len(v), got)
		}
	}
}

func TestD1_5_FingerprintStableAndSalted(t *testing.T) {
	const value = "same-secret-value"

	a := NewValueRedactor(map[string]any{"token": value}, true, nil).Redact(value)
	b := NewValueRedactor(map[string]any{"other_token": value}, true, nil).Redact(value)
	if a != b {
		t.Errorf("same value must fingerprint the same within a run: %q vs %q", a, b)
	}

	other := NewValueRedactor(map[string]any{"token": "different-secret!"}, true, nil).
		Redact("different-secret!")
	if a == other {
		t.Errorf("different values must not share a mask: %q", a)
	}

	unsalted := sha256.Sum256([]byte(value))
	if strings.Contains(a, hex.EncodeToString(unsalted[:])[:4]) {
		t.Errorf("fingerprint must be salted, not a bare sha256 of the value: %q", a)
	}
}

func TestD1_5_LongestValueFirst(t *testing.T) {
	const short = "abcdefgh"
	const long = short + "-1234567890"
	r := NewValueRedactor(map[string]any{"key": short, "token": long}, true, nil)

	got := r.Redact(long)
	want := fmt.Sprintf("%s(%d chars, #%s)", MaskedValue, len(long), fingerprint(long))
	if got != want {
		t.Errorf("a secret containing another secret must be masked whole: got %q, want %q", got, want)
	}
}

func TestD1_5_NilRedactorLeavesTextAlone(t *testing.T) {
	var r *ValueRedactor
	if got := r.Redact("plain text"); got != "plain text" {
		t.Errorf("nil redactor must pass text through, got %q", got)
	}
}

// leakyValues carry characters a rendered request percent- or JSON-escapes.
var leakyValues = []string{
	"super-secret-client-value",
	"Zm9vYmFy/c2VjcmV0+dmFsdWU=",
	`pa$$w"rd-1234567890`,
	`with space and \ backslash`,
	"angle<brackets>and&ersand-value",
	"db%2Fpass-and%20more",
}

func TestLeak1_EncodedFormsAreMasked(t *testing.T) {
	for _, value := range leakyValues {
		t.Run(value, func(t *testing.T) {
			r := NewValueRedactor(map[string]any{"client_secret": value}, true, nil)
			jsonForm, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			for form, text := range map[string]string{
				"raw":         value,
				"queryEscape": url.QueryEscape(value),
				"pathEscape":  url.PathEscape(value),
				"jsonEscape":  string(jsonForm[1 : len(jsonForm)-1]),
			} {
				got := r.Redact("client_secret=" + text)
				if strings.Contains(got, text) {
					t.Errorf("%s form left in clear: %q", form, got)
				}
				if !strings.Contains(got, maskFor(value)) {
					t.Errorf("%s form not replaced by the value mask: %q", form, got)
				}
			}
		})
	}
}

// pathLeakyValues render differently under net/url's path encoding than under
// PathEscape: they combine one of / ; , with a character both escape.
var pathLeakyValues = []string{
	"pa55 word/with slash",
	"токен/значение",
	"alpha,beta gamma,delta",
	`semi;colon and "quote"`,
}

func TestLeak4_URLPathFormIsMasked(t *testing.T) {
	for _, value := range pathLeakyValues {
		t.Run(value, func(t *testing.T) {
			rendered := (&url.URL{Path: value}).EscapedPath()
			if rendered == url.PathEscape(value) {
				t.Fatalf("probe is not distinct from PathEscape: %q", rendered)
			}
			got := NewValueRedactor(map[string]any{"client_secret": value}, true, nil).
				Redact("https://api.example.com/v1/" + rendered + "/profile")
			if strings.Contains(got, rendered) {
				t.Errorf("URL-path form left in clear: %q", got)
			}
			if !strings.Contains(got, maskFor(value)) {
				t.Errorf("URL-path form not replaced by the value mask: %q", got)
			}
		})
	}
}

func TestLeak2_NonStringVaultValuesAreMasked(t *testing.T) {
	for name, raw := range map[string]any{
		"json_number_int":   json.Number("987654321098765"),
		"json_number_float": json.Number("1234.5678"),
		"int":               int(987654321098765),
		"int64":             int64(987654321098765),
		"float64":           float64(1234.5678),
	} {
		t.Run(name, func(t *testing.T) {
			rendered := fmt.Sprintf("%v", raw)
			if n, ok := raw.(json.Number); ok {
				rendered = n.String()
			}
			if len(rendered) < minMaskedValueLen {
				t.Fatalf("fixture %q is under the masking floor, so it proves nothing", rendered)
			}
			got := NewValueRedactor(map[string]any{"account_secret": raw}, true, nil).
				Redact("secret=" + rendered)
			if strings.Contains(got, rendered) {
				t.Errorf("non-string vault value left in clear: %q", got)
			}
		})
	}
}

func TestLeak2_CompositeValuesAreSkipped(t *testing.T) {
	r := NewValueRedactor(map[string]any{
		"nested_secret": map[string]any{"inner": "a-nested-secret-value"},
		"list_secret":   []any{"a-listed-secret-value"},
	}, true, nil)
	if len(r.secrets) != 0 {
		t.Errorf("composite values must not be registered, got %d", len(r.secrets))
	}
}

func TestD1_4a_FooterScopedToReferencedVariables(t *testing.T) {
	values := map[string]any{
		"client_secret": "",
		"tenant_id":     "",
		"unused_token":  "",
		"api_token":     "a-value-that-resolved",
	}
	referenced := []string{"api_token", "client_secret", "tenant_id"}

	for name, allSecret := range map[string]bool{"vault": true, "plain env": false} {
		t.Run(name, func(t *testing.T) {
			got := NewValueRedactor(values, allSecret, referenced).UnresolvedLine()
			if got != "// unresolved: client_secret, tenant_id" {
				t.Errorf("UnresolvedLine() = %q", got)
			}
		})
	}

	t.Run("no reference, no footer", func(t *testing.T) {
		if got := NewValueRedactor(values, true, nil).UnresolvedLine(); got != "" {
			t.Errorf("a request referencing nothing has nothing unresolved, got %q", got)
		}
	})
}

func TestD1_1_SecretKeyHintsClassifyKeyNames(t *testing.T) {
	tests := map[string]bool{
		"client_secret":   true,
		"access_token":    true,
		"db_password":     true,
		"api_key":         true,
		"aws_credential":  true,
		"auth_header":     true,
		"jwt":             true,
		"my_secret_v2":    true,
		"SECRET":          true,
		"Access_Token":    true,
		"base_url":        false,
		"tenant_id":       false,
		"method":          false,
		"client_identity": false,
		"":                false,
	}
	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			if got := isSecretKeyName(name); got != want {
				t.Errorf("isSecretKeyName(%q) = %v, want %v", name, got, want)
			}
		})
	}
}

func TestD1_1_EverySecretKeyHintMatches(t *testing.T) {
	for _, hint := range secretKeyHints {
		if !isSecretKeyName("prefix_" + hint + "_suffix") {
			t.Errorf("hint %q must match in the middle of a key name", hint)
		}
	}
}

// The mask is spliced over the bytes the match decoded from, so a wrong offset
// map shows up as damage either side of it.
func TestD1_7_MaskCoversExactlyTheEncodedRun(t *testing.T) {
	const value = "st?te tok!n Value"
	for name, tc := range map[string]struct {
		rendered *url.URL
		want     string
	}{
		"fragment": {
			rendered: &url.URL{Scheme: "https", Host: "api.example.com", Path: "/cb", Fragment: value},
			want:     "GET https://api.example.com/cb#" + maskFor(value) + "\n",
		},
		"path": {
			rendered: &url.URL{Scheme: "https", Host: "api.example.com", Path: "/v1/" + value + "/profile"},
			want:     "GET https://api.example.com/v1/" + maskFor(value) + "/profile\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			text := "GET " + tc.rendered.String() + "\n"
			if !strings.Contains(text, "%20") {
				t.Fatalf("probe does not exercise percent-encoding: %q", text)
			}
			got := NewValueRedactor(map[string]any{"client_secret": value}, true, nil).Redact(text)
			if got != tc.want {
				t.Errorf("Redact() = %q, want %q", got, tc.want)
			}
		})
	}
}
