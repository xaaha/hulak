package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileRefPrefix marks a resolved attachment inside a template result.
//
// A template action can only hand back a string, and the request map is
// re-encoded to YAML mid-pipeline, so file bytes cannot travel that path.
// attachFile returns this marker plus an absolute path instead, and the body
// encoder opens the file itself.
//
// The prefix carries a per-run random nonce because values reach the body
// encoder from places an attacker can influence: saved responses via
// getValueOf, environment variables, secrets. A guessable prefix would let any
// of those mint a marker and turn "forward this field" into "read and upload
// any local file". With the nonce, only attachFile in this process can produce
// a string the encoder honors; anything else stays literal text.
var FileRefPrefix = "hulak:file:" + refNonce() + ":"

func refNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// FileRef builds the marker for an already-resolved absolute path.
func FileRef(absPath string) string {
	return FileRefPrefix + absPath
}

// FileRefPath returns the path carried by a marker. Only an exact whole-value
// match counts: a marker embedded in a longer string is a mistake the caller
// should report, not a literal to send.
func FileRefPath(value string) (string, bool) {
	path, ok := strings.CutPrefix(value, FileRefPrefix)
	if !ok || path == "" {
		return "", false
	}
	return path, true
}

// ResolveAttachPath resolves a path for attachFile.
//
// It deliberately differs from ResolveProjectFile: an upload usually comes from
// outside the repo, so an explicitly absolute or ~-prefixed path may leave the
// project root. A relative path still resolves against the project root and
// still may not escape it, because "assets/../../../etc/passwd" reads as
// innocuous where "/etc/passwd" does not. Leaving the repo has to be spelled
// out in the path, which is the disclosure; no warning is needed on top.
func ResolveAttachPath(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("attachFile needs a path")
	}

	explicit := filepath.IsAbs(filePath) || strings.HasPrefix(filePath, "~")

	absPath, err := ExpandPath(filePath)
	if err != nil {
		return "", err
	}

	if !explicit {
		projectRoot, found := FindProjectRoot()
		if !found {
			return "", fmt.Errorf("not a hulak project: could not find project root")
		}
		absPath = anchorToRoot(filepath.Clean(filePath), projectRoot)
		if !withinRoot(absPath, projectRoot) {
			return "", fmt.Errorf(
				"attachFile %s escapes the project root; pass an absolute or ~ path to attach it deliberately",
				filePath,
			)
		}
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("attachFile: no such file %s", absPath)
		}
		return "", err
	}
	if !info.Mode().IsRegular() {
		// Directories, FIFOs, sockets, and device nodes are all rejected here.
		// A FIFO would block PrepareStruct with no writer, and /dev/zero would
		// stream without end; only a regular file has a size we can declare.
		return "", fmt.Errorf("attachFile: %s is not a regular file", filePath)
	}

	return absPath, nil
}
