package utils

import (
	"path/filepath"
	"strings"
)

// SiblingPath expands the getFile sibling shorthand. When arg begins with "*",
// the "*" stands for the current file's basename (its name with the hulak YAML
// suffix removed) and the remainder of arg is appended, yielding a path to a
// file sitting next to the current one in the same directory:
//
//	current: genesis/getUser.hk.yaml
//	  "*.gql"      -> genesis/getUser.gql
//	  "*.json"     -> genesis/getUser.json
//	  "*.tar.gz"   -> genesis/getUser.tar.gz
//
// It returns the resolved path and true. For any arg that does not begin with
// "*" it returns "", false, leaving normal getFile resolution untouched.
func SiblingPath(currentFile, arg string) (string, bool) {
	if !strings.HasPrefix(arg, "*") {
		return "", false
	}

	base := filepath.Base(currentFile)
	stem := base
	for _, suf := range RequestExts {
		if trimmed, ok := strings.CutSuffix(base, suf); ok {
			stem = trimmed
			break
		}
	}

	suffix := arg[len("*"):]
	return filepath.Join(filepath.Dir(currentFile), stem+suffix), true
}
