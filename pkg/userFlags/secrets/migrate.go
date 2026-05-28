// Contains command factory and handler for hulak secrets migrate.
package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

func newEnvMigrateCmd() *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Short: "Migrate env/*.env files to the encrypted vault",
		Long: "Convert plaintext env/*.env files into the encrypted vault (.hulak/store.age).\n\n" +
			"Parses each .env file, creates environment sections in the store, and encrypts.\n" +
			"If the store already has values, existing values win on conflicts (safe to re-run).\n" +
			"The env/ directory is NOT deleted — remove it manually after verifying the migration.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets migrate",
				Description: "Migrate all env/*.env files to the vault",
			},
		},
		Run: func(_ []string) error { return runEnvMigrate() },
	}
}

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

	// only migrate to hulak vault
	result, err := vault.BootstrapVault(cwd, "")
	if err != nil {
		return err
	}

	if err := migrateEnvFiles(envDir, result.Store); err != nil {
		return err
	}

	if err := vault.WriteStoreToRecipients(result.Store); err != nil {
		return err
	}

	return printMigrateSummary(wasFresh, result)
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
func printMigrateSummary(wasFresh bool, result *vault.BootstrapResult) error {
	if wasFresh {
		utils.PrintInfoStderr(fmt.Sprintf("\n  Identity file: %s", result.IdentityDesc))
		utils.PrintInfoStderr(fmt.Sprintf("  Public key:    %s", result.RecipientKey))
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
