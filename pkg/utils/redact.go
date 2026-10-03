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

// A base64 credential is found by its shape rather than by a registered value,
// so the scan for one needs the text with its percent-encoding already undone.
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
		if n, ok := escapeAt(text, i); ok {
			b.WriteByte(n)
			offsets = append(offsets, i)
			i += 3
			continue
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

// escapeAt reports the byte s spells as a percent escape at i. ParseUint reads
// either hex case, which is what lets %de and %DE match each other.
func escapeAt(s string, i int) (byte, bool) {
	if i+3 > len(s) || s[i] != '%' {
		return 0, false
	}
	n, err := strconv.ParseUint(s[i+1:i+3], 16, 8)
	if err != nil {
		return 0, false
	}
	return byte(n), true
}

// startBytes reports which bytes a run spelling value can begin with: the
// value's first byte, a percent when the renderer escaped that byte or the
// value's own percent sign, a plus standing in for a leading space, and the
// byte a leading escape of the value's own decodes to. Every first step of a
// spelling needs one of these, so the rest of the text is skipped whole.
func startBytes(value string) [256]bool {
	var starts [256]bool
	starts[value[0]] = true
	starts['%'] = true
	if value[0] == ' ' {
		starts['+'] = true
	}
	if b, ok := escapeAt(value, 0); ok {
		starts[b] = true
	}
	return starts
}

// A spellingMatcher finds where a run of text spelling one secret ends. A
// percent is ambiguous: the renderer may escape a byte, leave an escape the
// value itself holds standing, escape its percent sign, or decode it, and
// net/url does different ones of those to neighbouring bytes of the same
// value. Every reading is walked rather than picked, so no spelling has to be
// enumerated in advance. The readings meet again on the same (text index,
// value index) pair, so the walk advances a frontier of those pairs instead of
// recursing down each combination: two readings of every escape in the value
// is a doubling per escape when a pair is re-explored.
type spellingMatcher struct {
	value   string
	escapes []escape
	starts  [256]bool
	rows    [4]spellingRow
}

// An escape is the byte a percent escape spells, held per value index so the
// walk reads the value's own escapes off a table instead of re-parsing them at
// every visit.
type escape struct {
	b  byte
	ok bool
}

// A spellingRow holds the value indices the walk still owes one text index.
// queued keeps a pair off the frontier a second time; pending lists what to
// clear, so a start position costs only the pairs it actually reached.
type spellingRow struct {
	pending []int
	queued  []bool
}

func newSpellingMatcher(value string) *spellingMatcher {
	m := &spellingMatcher{
		value:   value,
		escapes: make([]escape, len(value)),
		starts:  startBytes(value),
	}
	for vi := range len(value) {
		b, ok := escapeAt(value, vi)
		m.escapes[vi] = escape{b: b, ok: ok}
	}
	for i := range m.rows {
		m.rows[i].queued = make([]bool, len(value)+1)
		m.rows[i].pending = make([]int, 0, len(value)+1)
	}
	return m
}

func (r *spellingRow) clear() {
	for _, vi := range r.pending {
		r.queued[vi] = false
	}
	r.pending = r.pending[:0]
}

// A reading advances the text by at most 3, so four rows hold every index the
// frontier can still reach and ti&3 never collides with a live one.
func (m *spellingMatcher) queue(ti, vi int) int {
	row := &m.rows[ti&3]
	if row.queued[vi] {
		return 0
	}
	row.queued[vi] = true
	row.pending = append(row.pending, vi)
	return 1
}

// end returns the offset just past the longest run of text from at on that
// spells the whole value, or -1 when none does. Every row is empty on entry
// and left empty on return, so a start that spells nothing costs no clearing.
func (m *spellingMatcher) end(text string, at int) int {
	value := m.value
	best := -1
	live := m.queue(at, 0)
	// Every reading consumes a value byte, so no spelling outruns this.
	last := min(len(text), at+3*len(value))
	for ti := at; ti <= last && live > 0; ti++ {
		row := &m.rows[ti&3]
		live -= len(row.pending)
		for _, vi := range row.pending {
			if vi == len(value) {
				best = ti
				continue
			}
			if ti >= len(text) {
				continue
			}
			tb, textEscaped := escapeAt(text, ti)
			if text[ti] == value[vi] || (value[vi] == ' ' && text[ti] == '+') {
				live += m.queue(ti+1, vi+1)
			}
			if textEscaped && tb == value[vi] {
				live += m.queue(ti+3, vi+1)
			}
			if ve := m.escapes[vi]; ve.ok {
				if textEscaped && tb == ve.b {
					live += m.queue(ti+3, vi+3)
				}
				if text[ti] == ve.b {
					live += m.queue(ti+1, vi+3)
				}
			}
		}
		row.clear()
	}
	if live > 0 {
		for i := range m.rows {
			m.rows[i].clear()
		}
	}
	return best
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
	claimed := make([]bool, len(text))
	spans := r.secretSpans(text, claimed)
	spans = append(spans, basicAuthSpans(text, claimed)...)
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

type maskedSpan struct {
	start, end int
	mask       string
}

// secretSpans returns the byte range of every run of text that spells a known
// secret, marking each one claimed. Longest secret first, so a secret nested
// inside another never claims the bytes that would leave the outer one's tail
// standing beside a mask.
func (r *ValueRedactor) secretSpans(text string, claimed []bool) []maskedSpan {
	var spans []maskedSpan
	for _, s := range r.secrets {
		matcher := newSpellingMatcher(s.value)
		for at := 0; at < len(text); at++ {
			if !matcher.starts[text[at]] || claimed[at] {
				continue
			}
			end := matcher.end(text, at)
			if end < 0 || slices.Contains(claimed[at:end], true) {
				continue
			}
			for i := at; i < end; i++ {
				claimed[i] = true
			}
			spans = append(spans, maskedSpan{at, end, s.mask})
			at = end - 1
		}
	}
	return spans
}

// basicAuthSpans returns the byte range of every base64 credential in text,
// found through each decoding because the run itself may be percent- or
// plus-encoded. Bytes a secret already claimed are left to that mask.
func basicAuthSpans(text string, claimed []bool) []maskedSpan {
	var spans []maskedSpan
	for _, decode := range matchDecodings {
		decoded, offsets := decode(text)
		for _, payload := range basicAuthPayloads(decoded) {
			start, end := offsets[payload[0]], offsets[payload[1]]
			if slices.Contains(claimed[start:end], true) {
				continue
			}
			for i := start; i < end; i++ {
				claimed[i] = true
			}
			spans = append(spans, maskedSpan{start, end, MaskedValue})
		}
	}
	return spans
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
