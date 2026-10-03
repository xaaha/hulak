package testutil

import (
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// AssertNoSecretForm fails when any encoding of value survives in out.
//
// It normalises out — percent-decoding, JSON-unescaping, base64-decoding and
// undoing query plus-encoding, repeatedly and in every order — instead of
// enumerating the forms the redactor registers. Restating that list would make
// the assertion blind to exactly the forms the redactor forgot.
func AssertNoSecretForm(t *testing.T, where, out, value string) {
	t.Helper()
	if visible, leaked := SecretFormVisible(out, value); leaked {
		t.Errorf("%s leaked the secret %q, visible in:\n%s", where, value, visible)
	}
}

// SecretFormVisible returns the normalisation of out that shows value, and
// whether one exists. Callers sweeping many values count misses with it
// instead of failing on the first.
func SecretFormVisible(out, value string) (string, bool) {
	spellings := valueSpellings(value)
	for _, variant := range decodedVariants(out) {
		for _, spelling := range spellings {
			if strings.Contains(variant, spelling) {
				return variant, true
			}
		}
	}
	return "", false
}

// valueSpellings returns value together with every text reachable from it by
// case-folding it and by decoding some of the percent escapes it itself holds.
// Normalising only out decodes those escapes on one side alone, so a value
// whose own escape survived beside a neighbour the renderer escaped would read
// as clean, and a renderer that upper-cases a value (HTTP methods) hands the
// redactor a spelling it was never given.
func valueSpellings(value string) []string {
	seen := map[string]bool{value: true}
	queue := []string{value}
	for _, folded := range []string{strings.ToUpper(value), strings.ToLower(value)} {
		if !seen[folded] {
			seen[folded] = true
			queue = append(queue, folded)
		}
	}
	for i := 0; i < len(queue); i++ {
		for at := 0; at+3 <= len(queue[i]); at++ {
			if queue[i][at] != '%' {
				continue
			}
			n, err := strconv.ParseUint(queue[i][at+1:at+3], 16, 8)
			if err != nil {
				continue
			}
			one := queue[i][:at] + string([]byte{byte(n)}) + queue[i][at+3:]
			if !seen[one] {
				seen[one] = true
				queue = append(queue, one)
			}
		}
	}
	return queue
}

// decodedVariants returns out together with every text reachable from it by
// repeatedly applying the decodings a renderer's escaping is undone by. Each
// decoding either shortens its input or leaves it alone, so the walk ends.
func decodedVariants(out string) []string {
	seen := map[string]bool{out: true}
	queue := []string{out}
	for i := 0; i < len(queue); i++ {
		decodings := []string{
			percentUnescape(queue[i]),
			jsonUnescape(queue[i]),
			strings.ReplaceAll(queue[i], "+", " "),
		}
		for _, enc := range base64Encodings {
			decodings = append(decodings, base64Unescape(queue[i], enc))
		}
		for _, decoded := range decodings {
			if !seen[decoded] {
				seen[decoded] = true
				queue = append(queue, decoded)
			}
		}
	}
	return queue
}

// percentUnescape decodes every %XX in s and leaves the rest untouched, so a
// stray percent sign in surrounding text does not abandon the whole decode the
// way url.PathUnescape would.
func percentUnescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '%' && i+3 <= len(s) {
			if n, err := strconv.ParseUint(s[i+1:i+3], 16, 8); err == nil {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// jsonUnescape undoes the backslash escapes encoding/json writes inside a
// string, over text that is not itself a complete JSON document.
func jsonUnescape(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			i++
			continue
		}
		switch s[i+1] {
		case '"', '\\', '/':
			b.WriteByte(s[i+1])
			i += 2
		case 'n':
			b.WriteByte('\n')
			i += 2
		case 'r':
			b.WriteByte('\r')
			i += 2
		case 't':
			b.WriteByte('\t')
			i += 2
		case 'b':
			b.WriteByte('\b')
			i += 2
		case 'f':
			b.WriteByte('\f')
			i += 2
		case 'u':
			pair, err := hex.DecodeString(safeSlice(s, i+2, i+6))
			if err != nil || len(pair) != 2 {
				b.WriteByte(s[i])
				i++
				continue
			}
			b.WriteRune(rune(pair[0])<<8 | rune(pair[1]))
			i += 6
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

func safeSlice(s string, start, end int) string {
	if start > len(s) || end > len(s) {
		return ""
	}
	return s[start:end]
}

// base64Encodings covers the four spellings encoding/base64 offers, so a
// secret rendered through basicAuth is found whichever one produced it.
var base64Encodings = []*base64.Encoding{
	base64.StdEncoding, base64.RawStdEncoding,
	base64.URLEncoding, base64.RawURLEncoding,
}

// Shorter than this a run decodes to too little to hold a secret, and the
// walk would spend its time on garbage.
const minBase64Run = 8

// base64Unescape replaces every base64 run in s that decodes to text with that
// text. Runs that decode to bytes a renderer would never have printed are left
// alone, which is what keeps the variant walk from exploding on ordinary words.
func base64Unescape(s string, enc *base64.Encoding) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		end := i
		for end < len(s) && isBase64Byte(s[end]) {
			end++
		}
		if end == i {
			b.WriteByte(s[i])
			i++
			continue
		}
		text, used := longestDecodable(s[i:end], enc)
		b.WriteString(text)
		b.WriteString(s[i+used : end])
		i = end
	}
	return b.String()
}

// longestDecodable returns the text the longest decodable prefix of run holds,
// and how many bytes of run it consumed.
func longestDecodable(run string, enc *base64.Encoding) (string, int) {
	for n := len(run); n >= minBase64Run; n-- {
		raw, err := enc.DecodeString(run[:n])
		if err != nil || !printableText(raw) {
			continue
		}
		return string(raw), n
	}
	return "", 0
}

func isBase64Byte(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' ||
		c == '+' || c == '/' || c == '-' || c == '_' || c == '='
}

func printableText(raw []byte) bool {
	if !utf8.Valid(raw) {
		return false
	}
	for _, r := range string(raw) {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}
