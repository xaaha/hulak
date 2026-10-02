package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
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

// Below this, a value also occurs in unrelated text and masking blanks that out.
const minMaskedValueLen = 8

var secretKeyHints = []string{
	"secret", "token", "password", "key", "credential", "auth", "jwt",
}

// Unsalted, a weak secret's fingerprint falls to hashing candidates until one matches.
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

// A decoding rewrites rendered text back towards the value that produced it and
// reports, for every byte of the result, where in the original it came from.
// offsets has one more entry than the decoded text so a match's end maps too.
type decoding func(text string) (decoded string, offsets []int)

// net/url has five escaping tables and they disagree; each one is still only a
// percent-encoding, so undoing that reaches every URL spelling at once.
var matchDecodings = []decoding{verbatim, percentDecoded, queryDecoded}

func verbatim(text string) (string, []int) {
	offsets := make([]int, len(text)+1)
	for i := range offsets {
		offsets[i] = i
	}
	return text, offsets
}

func percentDecoded(text string) (string, []int) { return percentDecode(text, false) }

// url.Values.Encode spells a space as +, which no percent-decoding undoes.
func queryDecoded(text string) (string, []int) { return percentDecode(text, true) }

func percentDecode(text string, plusIsSpace bool) (string, []int) {
	var b strings.Builder
	b.Grow(len(text))
	offsets := make([]int, 0, len(text)+1)
	for i := 0; i < len(text); {
		if text[i] == '%' && i+3 <= len(text) {
			if n, err := strconv.ParseUint(text[i+1:i+3], 16, 8); err == nil {
				b.WriteByte(byte(n))
				offsets = append(offsets, i)
				i += 3
				continue
			}
		}
		if plusIsSpace && text[i] == '+' {
			b.WriteByte(' ')
		} else {
			b.WriteByte(text[i])
		}
		offsets = append(offsets, i)
		i++
	}
	return b.String(), append(offsets, len(text))
}

// basicAuth joins a username the redactor was never handed to the secret and
// base64-encodes the pair, so no decoding of the output and no registered value
// reaches it. Only an Authorization header is covered by header-name masking.
const basicAuthPrefix = "Basic "

// basicAuthPayloads returns the byte range of each base64 credential in text.
func basicAuthPayloads(text string) [][2]int {
	var found [][2]int
	for at := 0; at < len(text); {
		i := strings.Index(text[at:], basicAuthPrefix)
		if i < 0 {
			break
		}
		from := at + i + len(basicAuthPrefix)
		end := from
		for end < len(text) && isBase64Byte(text[end]) {
			end++
		}
		if n, ok := credentialLen(text[from:end]); ok {
			found = append(found, [2]int{from, from + n})
		}
		at = from
	}
	return found
}

// credentialLen reports how much of run is the longest prefix decoding to a
// colon-joined pair. A shorter prefix would decode to the username alone and
// leave the password beside the mask.
func credentialLen(run string) (int, bool) {
	for n := len(run) - len(run)%4; n >= 4; n -= 4 {
		raw, err := base64.StdEncoding.DecodeString(run[:n])
		if err == nil && bytes.ContainsRune(raw, ':') {
			return n, true
		}
	}
	return 0, false
}

func isBase64Byte(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' ||
		c == '+' || c == '/' || c == '='
}

func jsonStringForm(value string) string {
	quoted, err := json.Marshal(value)
	if err != nil {
		return value
	}
	return string(quoted[1 : len(quoted)-1])
}

// The vault decodes numbers as json.Number, which a string assertion would drop.
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
// hint are. Values below the masking floor are left alone. referenced names
// the variables the request uses: those of them that resolved empty are
// reported through UnresolvedLine, whatever their key name looks like.
func NewValueRedactor(values map[string]any, allSecret bool, referenced []string) *ValueRedactor {
	r := &ValueRedactor{}
	for name, raw := range values {
		value, ok := secretText(raw)
		if !ok {
			continue
		}
		if value == "" {
			if slices.Contains(referenced, name) {
				r.unresolved = append(r.unresolved, name)
			}
			continue
		}
		if !allSecret && !isSecretKeyName(name) {
			continue
		}
		if len(value) >= minMaskedValueLen {
			mask := maskFor(value)
			r.secrets = append(r.secrets, maskedSecret{value: value, mask: mask})
			// Backslash escapes are not a percent-encoding, so no decoding reaches them.
			if escaped := jsonStringForm(value); escaped != value {
				r.secrets = append(r.secrets, maskedSecret{value: escaped, mask: mask})
			}
		}
	}
	// Longest first: a secret inside another must not be left partially revealed.
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
	for _, decode := range matchDecodings {
		text = r.redactThrough(text, decode)
	}
	return text
}

type maskedSpan struct {
	start, end int
	mask       string
}

// redactThrough masks every secret visible once text is run back through
// decode, splicing the mask over the bytes of text the match decoded from.
func (r *ValueRedactor) redactThrough(text string, decode decoding) string {
	decoded, offsets := decode(text)
	claimed := make([]bool, len(decoded))
	var spans []maskedSpan
	for _, s := range r.secrets {
		for at := 0; at <= len(decoded)-len(s.value); {
			found := strings.Index(decoded[at:], s.value)
			if found < 0 {
				break
			}
			start := at + found
			end := start + len(s.value)
			if !slices.Contains(claimed[start:end], true) {
				for i := start; i < end; i++ {
					claimed[i] = true
				}
				spans = append(spans, maskedSpan{offsets[start], offsets[end], s.mask})
			}
			at = start + 1
		}
	}
	for _, payload := range basicAuthPayloads(decoded) {
		if slices.Contains(claimed[payload[0]:payload[1]], true) {
			continue
		}
		spans = append(spans, maskedSpan{offsets[payload[0]], offsets[payload[1]], MaskedValue})
	}
	if len(spans) == 0 {
		return text
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })

	var b strings.Builder
	written := 0
	for _, span := range spans {
		if span.start < written {
			continue
		}
		b.WriteString(text[written:span.start])
		b.WriteString(span.mask)
		written = span.end
	}
	b.WriteString(text[written:])
	return b.String()
}

// Unresolved names the variables the request references that resolved to an
// empty string, sorted. A nil receiver reports none.
func (r *ValueRedactor) Unresolved() []string {
	if r == nil {
		return nil
	}
	return slices.Clone(r.unresolved)
}

// UnresolvedLine names the variables the request references that resolved to
// an empty string, or returns "" when none did. An empty value cannot be
// masked by replacement, so it is reported rather than hidden.
func (r *ValueRedactor) UnresolvedLine() string {
	if r == nil || len(r.unresolved) == 0 {
		return ""
	}
	return "// unresolved: " + strings.Join(r.unresolved, ", ")
}
