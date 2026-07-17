package utils

import "strings"

const (
	TemplateFuncGetFile    = "getFile"
	TemplateFuncGetValueOf = "getValueOf"
	TemplateFuncBasicAuth  = "basicAuth"
	TemplateFuncOs         = "os"
)

// templateFuncNames is the canonical set of template action names. It is the
// single source the name resolver derives its variants from — add a new action
// here and every case/underscore spelling of it resolves automatically.
var templateFuncNames = []string{
	TemplateFuncGetFile,
	TemplateFuncGetValueOf,
	TemplateFuncBasicAuth,
	TemplateFuncOs,
}

// canonicalActions maps a normalized action name to its canonical spelling.
var canonicalActions = buildCanonicalActions()

func buildCanonicalActions() map[string]string {
	m := make(map[string]string, len(templateFuncNames))
	for _, name := range templateFuncNames {
		m[normalizeActionName(name)] = name
	}
	return m
}

// normalizeActionName lowercases and strips underscores so getFile, getfile,
// GetFile, and get_file all collapse to the same key. Underscore is therefore
// reserved as a word separator for future multi-word action names.
func normalizeActionName(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, "_", ""))
}

// CanonicalActionName returns the canonical spelling of a template action name
// written in any case/underscore variant, and whether it is a known action.
func CanonicalActionName(token string) (string, bool) {
	canonical, ok := canonicalActions[normalizeActionName(token)]
	return canonical, ok
}
