package utils

import (
	"path/filepath"
	"strings"
)

// RequestStem lowercases name and strips a request extension, so "login",
// "login.hk.yaml", and "LOGIN.yml" all reduce to the same key. Names without a
// request extension pass through unchanged (lowercased).
func RequestStem(name string) string {
	name = strings.ToLower(filepath.Base(name))
	for _, ext := range RequestExts {
		if stem, ok := strings.CutSuffix(name, ext); ok {
			return stem
		}
	}
	return name
}

// IsRequestFile reports whether base is a runnable request file: a .yaml/.yml
// that is not the options.yaml reference card. Response .json files and other
// extensions are excluded.
func IsRequestFile(base string) bool {
	if strings.EqualFold(base, OptionsReference) {
		return false
	}
	lower := strings.ToLower(base)
	return strings.HasSuffix(lower, YAML) || strings.HasSuffix(lower, YML)
}

// FindRequestFiles returns every request file under root whose stem matches
// name, recursively. name may be given with or without an extension. Unlike
// ListMatchingFiles (the -f resolver, which is yaml/yml/json and errors on
// zero matches), this is request-only and returns an empty slice — not an
// error — when nothing matches, so callers can distinguish "none" from a real
// filesystem error.
func FindRequestFiles(root, name string) ([]string, error) {
	want := RequestStem(name)
	files, err := ListFiles(root)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, f := range files {
		base := filepath.Base(f)
		if IsRequestFile(base) && RequestStem(base) == want {
			matches = append(matches, f)
		}
	}
	return matches, nil
}
