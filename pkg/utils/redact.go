package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strings"
)

// MaskedValue is the placeholder printed in place of sensitive content
// (env values, auth-style headers) when --show is off. Shared so every
// command masks with the same glyph.
const MaskedValue = "••••"

// sensitiveHeaders lists header names whose values are masked by default
// when printing requests. Compared case-insensitively. Kept narrow on
// purpose — over-redaction frustrates users more than under-redaction.
// Extend deliberately, not speculatively.
var sensitiveHeaders = map[string]bool{
	"authorization":        true,
	"proxy-authorization":  true,
	"proxy-authenticate":   true,
	"www-authenticate":     true,
	"cookie":               true,
	"set-cookie":           true,
	"x-api-key":            true,
	"x-auth-token":         true,
	"x-csrf-token":         true,
	"x-amz-security-token": true,
}

// IsSensitiveHeader reports whether name is in the sensitive-headers set.
// Comparison is case-insensitive.
func IsSensitiveHeader(name string) bool {
	return sensitiveHeaders[strings.ToLower(name)]
}

// RedactHeaders returns a copy of headers with sensitive values masked.
// When show is true, returns a copy with values unchanged so callers do
// not have to branch. Original map is never mutated.
func RedactHeaders(headers map[string]string, show bool) map[string]string {
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		if !show && IsSensitiveHeader(k) {
			out[k] = MaskedValue
			continue
		}
		out[k] = v
	}
	return out
}

// minMaskedValueLen is the shortest resolved value worth replacing. Below it
// the value is short enough to occur in unrelated text, and masking would
// blank out more than the secret.
const minMaskedValueLen = 8

// secretKeyHints classify a key name as holding a secret when the values came
// from plain env files rather than the encrypted vault. Matched as a
// case-insensitive substring.
var secretKeyHints = []string{
	"secret", "token", "password", "key", "credential", "auth", "jwt",
}

// fingerprintSalt makes a mask's fingerprint comparable within one process and
// meaningless outside it. Without it, the fingerprint of a weak secret can be
// brute forced by hashing candidates until one matches.
var fingerprintSalt = randomFingerprintSalt()

func randomFingerprintSalt() []byte {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		panic("generating secret fingerprint salt: " + err.Error())
	}
	return salt
}

func fingerprint(value string) string {
	sum := sha256.Sum256(slices.Concat(fingerprintSalt, []byte(value)))
	return hex.EncodeToString(sum[:])[:4]
}

func maskFor(value string) string {
	return fmt.Sprintf("%s(%d chars, #%s)", MaskedValue, len(value), fingerprint(value))
}

// A rendered form missing from this list is printed in clear: replacement
// matches literal text, and the renderer escapes before the redactor runs.
func renderedForms(value string) []string {
	forms := []string{value}
	for _, encoded := range []string{
		url.QueryEscape(value),
		url.PathEscape(value),
		jsonStringForm(value),
	} {
		if !slices.Contains(forms, encoded) {
			forms = append(forms, encoded)
		}
	}
	return forms
}

func jsonStringForm(value string) string {
	quoted, err := json.Marshal(value)
	if err != nil {
		return value
	}
	return string(quoted[1 : len(quoted)-1])
}

// The vault decodes numbers as json.Number, so a type assertion to string
// drops every numeric secret from the mask list and prints it in clear.
func secretText(raw any) (string, bool) {
	switch value := raw.(type) {
	case string:
		return value, true
	case json.Number:
		return value.String(), true
	case bool, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprintf("%v", value), true
	}
	return "", false
}

func isSecretKeyName(name string) bool {
	lower := strings.ToLower(name)
	for _, hint := range secretKeyHints {
		if strings.Contains(lower, hint) {
			return true
		}
	}
	return false
}

type maskedSecret struct {
	value string
	mask  string
}

// ValueRedactor replaces resolved secret values with a masked placeholder
// carrying the value's length and a salted fingerprint.
type ValueRedactor struct {
	secrets    []maskedSecret
	unresolved []string
}

// NewValueRedactor builds a redactor over values. When allSecret is true every
// entry is a secret; otherwise only entries whose key name matches a secret
// hint are. Values below the masking floor are left alone, and ones that
// resolved empty are reported through UnresolvedLine instead.
func NewValueRedactor(values map[string]any, allSecret bool) *ValueRedactor {
	r := &ValueRedactor{}
	for name, raw := range values {
		if !allSecret && !isSecretKeyName(name) {
			continue
		}
		value, ok := secretText(raw)
		if !ok {
			continue
		}
		switch {
		case value == "":
			r.unresolved = append(r.unresolved, name)
		case len(value) >= minMaskedValueLen:
			mask := maskFor(value)
			for _, form := range renderedForms(value) {
				r.secrets = append(r.secrets, maskedSecret{value: form, mask: mask})
			}
		}
	}
	// Longest first, so a secret that is a substring of another is never
	// replaced inside it and left partially revealed.
	sort.Slice(r.secrets, func(i, j int) bool {
		if len(r.secrets[i].value) != len(r.secrets[j].value) {
			return len(r.secrets[i].value) > len(r.secrets[j].value)
		}
		return r.secrets[i].value < r.secrets[j].value
	})
	sort.Strings(r.unresolved)
	return r
}

// Redact returns text with every known secret value replaced by its mask. A
// nil receiver returns text unchanged, so callers that mask nothing can pass
// nil instead of branching.
func (r *ValueRedactor) Redact(text string) string {
	if r == nil {
		return text
	}
	for _, s := range r.secrets {
		text = strings.ReplaceAll(text, s.value, s.mask)
	}
	return text
}

// UnresolvedLine names the secret-classified variables that resolved to an
// empty string, or returns "" when none did. An empty value cannot be masked
// by replacement, so it is reported rather than hidden.
func (r *ValueRedactor) UnresolvedLine() string {
	if r == nil || len(r.unresolved) == 0 {
		return ""
	}
	return "// unresolved: " + strings.Join(r.unresolved, ", ")
}
