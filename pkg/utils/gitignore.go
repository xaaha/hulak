package utils

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// EnsureGitignoreEntry adds entry to .gitignore (in cwd) if not already present.
// Idempotent. Re-running is a no-op once the entry exists. The check treats
// "foo/" and "foo" as the same entry so trailing slashes don't double up.
func EnsureGitignoreEntry(entry string) error {
	gitignorePath, err := CreatePath(".gitignore")
	if err != nil {
		return fmt.Errorf("could not resolve .gitignore path: %w", err)
	}

	existing, err := os.ReadFile(gitignorePath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("could not read .gitignore: %w", err)
	}

	bare := strings.TrimRight(entry, "/")
	for line := range strings.SplitSeq(string(existing), "\n") {
		line = strings.TrimSpace(line)
		if line == entry || line == bare {
			return nil
		}
	}

	prefix := ""
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		prefix = "\n"
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, FilePer)
	if err != nil {
		return fmt.Errorf("could not open .gitignore for writing: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "%s%s\n", prefix, entry); err != nil {
		return fmt.Errorf("could not write to .gitignore: %w", err)
	}

	return nil
}
