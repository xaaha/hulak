package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
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
	const value = "s3cr3t-value-0123456789abcd"
	r := NewValueRedactor(map[string]any{"client_secret": value}, true, nil)

	got := r.Redact("body=" + value)
	want := fmt.Sprintf("body=%s(%d chars, #%s)", MaskedValue, len(value), fingerprint(value))
	if got != want {
		t.Errorf("Redact() = %q, want %q", got, want)
	}
	if !regexp.MustCompile(`^body=••••\(27 chars, #[0-9a-f]{4}\)$`).MatchString(got) {
		t.Errorf("mask does not match the agreed shape: %q", got)
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

// leakyValues exercise the transforms a rendered request applies to a value:
// percent-encoding in a query string or urlencoded body, and JSON string
// escaping in a JSON body.
var leakyValues = []string{
	"super-secret-client-value",
	"Zm9vYmFy/c2VjcmV0+dmFsdWU=",
	`pa$$w"rd-1234567890`,
	`with space and \ backslash`,
	"angle<brackets>and&ersand-value",
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

func TestLeak2_NonStringVaultValuesAreMasked(t *testing.T) {
	for name, raw := range map[string]any{
		"json_number_int":   json.Number("987654321098765"),
		"json_number_float": json.Number("1234.5678"),
		"int":               int(987654321098765),
		"int64":             int64(987654321098765),
		"float64":           float64(1234.5678),
		"bool":              true,
	} {
		t.Run(name, func(t *testing.T) {
			rendered := fmt.Sprintf("%v", raw)
			if n, ok := raw.(json.Number); ok {
				rendered = n.String()
			}
			got := NewValueRedactor(map[string]any{"account_secret": raw}, true, nil).
				Redact("secret=" + rendered)
			if len(rendered) < minMaskedValueLen {
				if got != "secret="+rendered {
					t.Errorf("value under the masking floor must be left alone: %q", got)
				}
				return
			}
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
