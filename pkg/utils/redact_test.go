package utils

import (
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
