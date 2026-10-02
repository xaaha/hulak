package testutil

import (
	"encoding/hex"
	"strconv"
	"strings"
	"testing"
)

// AssertNoSecretForm fails when any encoding of value survives in out.
//
// It normalises out — percent-decoding, JSON-unescaping and undoing query
// plus-encoding, repeatedly and in every order — instead of enumerating the
// forms the redactor registers. Restating that list would make the assertion
// blind to exactly the forms the redactor forgot.
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
	for _, variant := range decodedVariants(out) {
		if strings.Contains(variant, value) {
			return variant, true
		}
	}
	return "", false
}

// decodedVariants returns out together with every text reachable from it by
// repeatedly applying the decodings a renderer's escaping is undone by. Each
// decoding either shortens its input or leaves it alone, so the walk ends.
func decodedVariants(out string) []string {
	seen := map[string]bool{out: true}
	queue := []string{out}
	for i := 0; i < len(queue); i++ {
		for _, decoded := range []string{
			percentUnescape(queue[i]),
			jsonUnescape(queue[i]),
			strings.ReplaceAll(queue[i], "+", " "),
		} {
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
