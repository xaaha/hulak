package userflags

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// runEnvMigrate converts env/*.env files into the encrypted vault store.
// Existing store values win on conflicts, making re-runs safe.
// The env/ directory is NOT deleted — users do that manually.
func runEnvMigrate() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine current directory: %w", err)
	}

	envDir := filepath.Join(cwd, utils.EnvironmentFolder)
	if err := requireDirectory(envDir); err != nil {
		return err
	}

	// Snapshot before EnsureKeypair so we only show the backup
	// warning when a brand-new identity is generated.
	wasFresh := !vault.IdentityExists()

	ageKey, store, err := bootstrapVault(cwd)
	if err != nil {
		return err
	}

	if err := migrateEnvFiles(envDir, store); err != nil {
		return err
	}

	if err := vault.WriteStoreToRecipients(store); err != nil {
		return err
	}

	return printMigrateSummary(wasFresh, ageKey)
}

// requireDirectory checks that path exists and is a directory.
func requireDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no %s/ directory found — nothing to migrate", filepath.Base(path))
		}
		return fmt.Errorf("cannot access %s/: %w", filepath.Base(path), err)
	}
	if !info.IsDir() {
		return fmt.Errorf("expected %s/ to be a directory, got a file", filepath.Base(path))
	}
	return nil
}

// bootstrapVault ensures .hulak/, identity, and recipients exist,
// then returns the keypair and the current store (empty if first run).
func bootstrapVault(projectRoot string) (vault.AgeKey, *vault.Store, error) {
	hulakDir := filepath.Join(projectRoot, utils.HiddenProjectName)
	if err := os.MkdirAll(hulakDir, utils.DirPer); err != nil {
		return vault.AgeKey{}, nil, fmt.Errorf("could not create %s/: %w", utils.HiddenProjectName, err)
	}

	ageKey, err := vault.EnsureKeypair()
	if err != nil {
		return vault.AgeKey{}, nil, err
	}

	if err := ensureRecipientsFile(ageKey); err != nil {
		return vault.AgeKey{}, nil, err
	}

	store, err := vault.ReadStore(ageKey.Identity)
	if err != nil {
		return vault.AgeKey{}, nil, err
	}

	return ageKey, store, nil
}

// migrateEnvFiles reads each *.env file in envDir and merges its
// key-value pairs into store. Existing store values take precedence
// so re-running migration never overwrites post-migration edits.
// Non-.env files are skipped with a warning.
func migrateEnvFiles(envDir string, store *vault.Store) error {
	entries, err := os.ReadDir(envDir)
	if err != nil {
		return fmt.Errorf("failed to read %s/: %w", filepath.Base(envDir), err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, utils.DefaultEnvFileSuffix) {
			utils.PrintWarningStderr(fmt.Sprintf("Skipped: %s (not a .env file)", name))
			continue
		}

		envName := strings.TrimSuffix(name, utils.DefaultEnvFileSuffix)
		if err := utils.ValidateEnvName(envName); err != nil {
			return fmt.Errorf("invalid env file %q: %w", name, err)
		}

		filePath := filepath.Join(envDir, name)
		if err := mergeEnvFileIntoStore(filePath, envName, store); err != nil {
			return err
		}
	}

	return nil
}

// mergeEnvFileIntoStore parses a single .env file with raw values
// (preserving $VAR literals) and adds new keys to the store section.
// Keys that already exist in the store are skipped.
func mergeEnvFileIntoStore(filePath, envName string, store *vault.Store) error {
	parsed, err := envparser.LoadEnvVarsRaw(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filepath.Base(filePath), err)
	}

	store.EnsureSection(envName)
	existing := store.GetEnv(envName)

	newKeys, skipped, invalidUTF8 := 0, 0, 0
	for key, val := range parsed {
		if _, exists := existing[key]; exists {
			skipped++
			continue
		}
		// JSON requires valid UTF-8. Skip binary/corrupt values with a warning.
		if str, ok := val.(string); ok && !utf8.ValidString(str) {
			invalidUTF8++
			utils.PrintWarningStderr(fmt.Sprintf(
				"Skipped: key %q in %s contains invalid UTF-8 bytes. "+
					"Store binary data as files and use {{getFile \"path\"}} instead.",
				key, filepath.Base(filePath),
			))
			continue
		}
		store.SetKey(envName, key, val)
		newKeys++
	}

	fileName := filepath.Base(filePath)
	var parts []string
	parts = append(parts, fmt.Sprintf("%d new", newKeys))
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skipped))
	}
	if invalidUTF8 > 0 {
		parts = append(parts, fmt.Sprintf("%d invalid UTF-8", invalidUTF8))
	}
	utils.PrintSuccessStderr(fmt.Sprintf(
		"Migrated %s → store.age[%s] (%s)", fileName, envName, strings.Join(parts, ", "),
	))

	return nil
}

// printMigrateSummary shows identity details on first-time setup
// and reminds the user that env/ is untouched.
func printMigrateSummary(wasFresh bool, ageKey vault.AgeKey) error {
	if wasFresh {
		identityPath, err := vault.IdentityPath()
		if err != nil {
			return fmt.Errorf("could not resolve identity path: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\n  Identity file: %s\n", identityPath)
		fmt.Fprintf(os.Stderr, "  Public key:    %s\n", ageKey.Recipient)
		utils.PrintWarningStderr(
			"Back up the identity file — losing it means losing access to the vault.",
		)
	}

	fmt.Fprintln(os.Stderr)
	utils.PrintInfoStderr(
		"env/ is untouched. The encrypted store now takes priority.\n" +
			"Delete it manually when ready: rm -rf env/",
	)
	return nil
}
