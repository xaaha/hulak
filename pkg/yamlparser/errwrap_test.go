package yamlparser

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestErrorsNoANSI guards against regression of issue #180. Errors from
// migrated call sites must be plain text — no ANSI escape codes, no leading
// newline. Color belongs at the print site, not in the error chain.
func TestErrorsNoANSI(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{
			name: "EncodeBody empty Body{}",
			err: func() error {
				_, _, e := (&Body{}).EncodeBody()
				return e
			}(),
		},
		{
			name: "EncodeBody empty url-encoded map",
			err: func() error {
				_, e := EncodeXwwwFormURLBody(map[string]string{})
				return e
			}(),
		},
		{
			name: "EncodeBody empty multipart map",
			err: func() error {
				_, _, e := EncodeFormData(map[string]string{})
				return e
			}(),
		},
		{
			name: "parsePath empty path",
			err: func() error {
				_, e := parsePath("")
				return e
			}(),
		},
		{
			name: "parsePath empty segment",
			err: func() error {
				_, e := parsePath("a ->  -> b")
				return e
			}(),
		},
		{
			name: "AuthRequestFile nil body",
			err: func() error {
				var a *AuthRequestFile
				_, e := a.IsValid()
				return e
			}(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Fatalf("expected error, got nil")
			}
			msg := tc.err.Error()
			if strings.Contains(msg, "\x1b") {
				t.Errorf("error contains ANSI escape: %q", msg)
			}
			if strings.HasPrefix(msg, "\n") {
				t.Errorf("error starts with newline: %q", msg)
			}
		})
	}
}

// TestFinalStructForOAuth2WrapsErrors pins both #180 properties at the auth2
// site that previously had literal "%v" in the format string:
//  1. rendered message contains no literal printf verbs
//  2. errors.Unwrap returns non-nil, so the chain is intact for errors.Is/As
func TestFinalStructForOAuth2WrapsErrors(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "valid-yaml-bad-auth.yaml")
	// Decodes cleanly but fails IsValid — exercises the line 178 wrap site.
	if err := os.WriteFile(bad, []byte("kind: auth\nmethod: POST\n"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := FinalStructForOAuth2(bad, map[string]any{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	msg := err.Error()
	if strings.Contains(msg, "%v") || strings.Contains(msg, "%w") {
		t.Errorf("error contains literal printf verb: %q", msg)
	}
	if !strings.Contains(msg, "error on auth2 request body") {
		t.Errorf("expected outer wrap, got: %q", msg)
	}
	if errors.Unwrap(err) == nil {
		t.Error("expected wrapped chain via %w, got unwrap=nil")
	}
}
