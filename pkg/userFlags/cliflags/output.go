package cliflags

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// RegisterOutput binds both --out and its short form -o on fs to a single
// underlying string. Returns a pointer so the handler reads the parsed value
// after flag.Parse. Pair with ResolveOutputPath at call time to keep
// "user pointed at this path" semantics consistent across every command that
// accepts an output path.
func RegisterOutput(fs *flag.FlagSet, usage string) *string {
	var out string
	fs.StringVar(&out, "out", "", usage)
	fs.StringVar(&out, "o", "", usage)
	return &out
}

// ResolveOutputPath converts a user-supplied -o/--out value into a final
// destination path. It distinguishes "the user pointed at a directory"
// (append canonical filename) from "the user pointed at a file path" (use
// verbatim), so commands behave consistently no matter how the user wrote
// the path.
//
// Non-default paths are expanded through utils.ExpandPath first, so leading
// ~ works consistently on Unix and Windows and returned paths are absolute.
// Rules (first match wins):
//  1. outPath empty, ".", or "./" → cwd/canonical (absolute)
//  2. outPath ends in '/' or platform separator → dir mode: outPath/canonical
//  3. outPath names an existing directory → dir mode: outPath/canonical
//  4. outPath has an extension matching knownExts (case-insensitive) → file mode
//  5. knownExts is empty AND outPath has any extension → file mode
//  6. otherwise → dir mode (DWIM): outPath/canonical
//
// knownExts may include or omit leading dots — both ".yaml" and "yaml" work.
//
// The rule-5 fallback (any-extension → file) is the right default for
// commands like `secrets identity export` where the user picks whatever
// extension suits their workflow (.txt, .pem, .key). Commands that want to
// restrict to a specific format (e.g. backup → .age) should pass knownExts
// explicitly.
func ResolveOutputPath(outPath, canonical string, knownExts ...string) (string, error) {
	// Normalize "." / "./" / "" → treat as "use cwd". filepath.Clean folds all
	// three to "." so a single comparison handles them.
	if outPath == "" || filepath.Clean(outPath) == "." {
		return utils.CreatePath(canonical)
	}

	// Record explicit directory syntax before ExpandPath cleans the trailing
	// separator. This matters for non-existent directories containing a dot.
	sep := string(filepath.Separator)
	dirMode := strings.HasSuffix(outPath, "/") || strings.HasSuffix(outPath, sep)

	expanded, err := utils.ExpandPath(outPath)
	if err != nil {
		return "", fmt.Errorf("expanding output path: %w", err)
	}
	outPath = expanded

	if dirMode || utils.DirExists(outPath) {
		return filepath.Join(outPath, canonical), nil
	}

	ext := strings.ToLower(filepath.Ext(outPath))
	if ext != "" {
		if len(knownExts) == 0 {
			return outPath, nil
		}
		for _, e := range knownExts {
			e = strings.ToLower(e)
			if !strings.HasPrefix(e, ".") {
				e = "." + e
			}
			if ext == e {
				return outPath, nil
			}
		}
	}
	return filepath.Join(outPath, canonical), nil
}
