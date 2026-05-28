package utils

import (
	"bufio"
	"fmt"
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

	if FileExists(gitignorePath) {
		file, err := os.Open(gitignorePath)
		if err != nil {
			return fmt.Errorf("could not read .gitignore: %w", err)
		}
		defer file.Close()

		bare := strings.TrimRight(entry, "/")
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == entry || line == bare {
				return nil
			}
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading .gitignore: %w", err)
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, FilePer)
	if err != nil {
		return fmt.Errorf("could not open .gitignore for writing: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("could not stat .gitignore: %w", err)
	}

	prefix := ""
	if info.Size() > 0 {
		content, err := os.ReadFile(gitignorePath)
		if err != nil {
			return fmt.Errorf("could not read .gitignore: %w", err)
		}
		if len(content) > 0 && content[len(content)-1] != '\n' {
			prefix = "\n"
		}
	}

	if _, err := fmt.Fprintf(f, "%s%s\n", prefix, entry); err != nil {
		return fmt.Errorf("could not write to .gitignore: %w", err)
	}

	return nil
}
