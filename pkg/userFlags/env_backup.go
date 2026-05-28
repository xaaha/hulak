// Contains command factories and handlers for hulak secrets backup and hulak secrets restore.
package userflags

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/userFlags/cliflags"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// backupPrefix is the filename prefix for default backups.
const backupPrefix = "store.age.bak."

// backupTimestampFormat formats timestamps for backup filenames.
// Uses dashes instead of colons so filenames are valid on all platforms.
const (
	backupTimestampFormat  = "2006-01-02T15-04-05"
	displayTimestampFormat = "2006-01-02 15:04:05"
)

func newEnvBackupCmd() *cli.Command {
	fs := flag.NewFlagSet("env backup", flag.ContinueOnError)
	var force bool
	out := cliflags.RegisterOutput(
		fs,
		"Custom output path for the backup file. Directory inputs append a timestamped name.",
	)
	fs.BoolVar(&force, "force", false, "Overwrite existing --out target")
	fs.BoolVar(&force, "f", false, "Overwrite existing --out target")

	return &cli.Command{
		Name:  "backup",
		Short: "Create a backup of the encrypted store",
		Long: "Copy store.age to a timestamped backup file after validating decryptability.\n\n" +
			"Default location: .hulak/backups/store.age.bak.<timestamp>\n" +
			"Use --out/-o for a custom path. Use 'hulak secrets backup list' to see existing backups.",
		Flags: fs,
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets backup",
				Description: "Create a timestamped backup of encrypted store.age",
			},
			{
				Command:     "hulak secrets backup -o ~/backups/my.age",
				Description: "Backup to a custom path",
			},
			{Command: "hulak secrets backup list", Description: "List existing backups"},
			{
				Command:     "hulak secrets backup --out existing.age --force",
				Description: "Overwrite an existing backup file",
			},
		},
		SubCommands: []*cli.Command{newEnvBackupListCmd()},
		Run: func(args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unexpected arguments: %v", args)
			}
			return runBackup(*out, force)
		},
	}
}

func newEnvBackupListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Short:   "List existing backups",
		Long:    "Show all backup files in .hulak/backups/ with timestamps.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak secrets backup list", Description: "List all backups"},
			{Command: "hulak secrets backup ls", Description: "Same as list (alias)"},
		},
		Run: func(args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unexpected arguments: %v", args)
			}
			return runBackupList()
		},
	}
}

// runBackup creates a backup of store.age.
func runBackup(outPath string, force bool) error {
	if err := cli.RequireVaultProject(); err != nil {
		return err
	}

	storePath, err := vault.StorePath()
	if err != nil {
		return err
	}

	cipherText, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no store.age found — nothing to back up")
		}
		return fmt.Errorf("failed to read store: %w", err)
	}

	// Validate: probe each identity to prove the store is healthy. Discard
	// the resolved identity — we only care that decryption is possible.
	if _, err := vault.ResolveIdentityFor(cipherText); err != nil {
		return fmt.Errorf(
			"store.age cannot be decrypted with current identity — refusing to back up a corrupt or inaccessible store: %w",
			err,
		)
	}

	destPath, err := resolveBackupDest(outPath, force)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(destPath), utils.DirPer); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	if err := os.WriteFile(destPath, cipherText, utils.SecretPer); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	// Auto-add .hulak/backups/ to .gitignore on first default-path backup
	if outPath == "" {
		if err := ensureGitignoreEntry(utils.HiddenProjectName + "/backups/"); err != nil {
			utils.PrintWarningStderr(fmt.Sprintf("could not update .gitignore: %v", err))
		}
	}

	utils.PrintSuccessStderr(fmt.Sprintf("Backup created: %s", destPath))
	return nil
}

// resolveBackupDest determines the backup destination path.
// For --out: applies the shared output-path rules. Directory inputs (trailing
// slash, existing dir, or no .age extension) get a fresh timestamped filename
// appended; .age inputs are used verbatim.
// For default: generates a timestamped path under .hulak/backups/ with collision handling.
func resolveBackupDest(outPath string, force bool) (string, error) {
	if outPath != "" {
		canonical := backupPrefix + time.Now().Format(backupTimestampFormat)
		dest, err := cliflags.ResolveOutputPath(outPath, canonical, ".age")
		if err != nil {
			return "", err
		}
		if !force && utils.FileExists(dest) {
			return "", fmt.Errorf("file already exists: %s (use --force to overwrite)", dest)
		}
		return dest, nil
	}

	backupsDir, err := vault.BackupsDir()
	if err != nil {
		return "", err
	}

	ts := time.Now().Format(backupTimestampFormat)
	base := filepath.Join(backupsDir, backupPrefix+ts)

	// Same-second collision: append .1, .2, etc. Cap at 999 to avoid unbounded spin.
	dest := base
	const maxCollisions = 999
	for counter := 1; utils.FileExists(dest); counter++ {
		if counter > maxCollisions {
			return "", fmt.Errorf("too many backups with timestamp %s (max %d)", ts, maxCollisions)
		}
		dest = fmt.Sprintf("%s.%d", base, counter)
	}

	return dest, nil
}

// runBackupList prints existing backups from .hulak/backups/.
func runBackupList() error {
	if err := cli.RequireVaultProject(); err != nil {
		return err
	}

	backupsDir, err := vault.BackupsDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			utils.PrintInfoStderr("No backups found")
			return nil
		}
		return fmt.Errorf("failed to read backups directory: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), backupPrefix) {
			names = append(names, e.Name())
		}
	}

	if len(names) == 0 {
		utils.PrintInfoStderr("No backups found")
		return nil
	}

	// Sort lexicographically (timestamps sort naturally), newest last
	sort.Strings(names)

	rows := make([][]string, len(names))
	for i, name := range names {
		tsStr := strings.TrimPrefix(name, backupPrefix)
		// Strip collision suffix if present (e.g. ".1")
		if idx := strings.LastIndex(tsStr, "."); idx > 0 {
			if _, err := fmt.Sscanf(tsStr[idx+1:], "%d", new(int)); err == nil {
				tsStr = tsStr[:idx]
			}
		}
		parsed, err := time.Parse(backupTimestampFormat, tsStr)
		if err != nil {
			rows[i] = []string{name, "(unknown)"}
			continue
		}
		rows[i] = []string{name, parsed.Format(displayTimestampFormat)}
	}

	return utils.PrintTable(
		os.Stdout,
		utils.StdoutHeaders([]string{"BACKUP", "CREATED"}),
		rows,
		0,
	)
}

func newEnvRestoreCmd() *cli.Command {
	fs := flag.NewFlagSet("env restore", flag.ContinueOnError)
	var force bool
	fs.BoolVar(&force, "force", false, "Skip confirmation prompt")
	fs.BoolVar(&force, "f", false, "Skip confirmation prompt")

	return &cli.Command{
		Name:  "restore",
		Short: "Restore the encrypted store from a backup",
		Long: "Restore store.age from a backup file.\n\n" +
			"With no arguments, restores the latest backup from .hulak/backups/.\n" +
			"Pass a path to restore a specific backup. The backup is decrypted and\n" +
			"re-encrypted to the current recipients.txt.",
		Flags: fs,
		Args: []cli.ArgDef{
			{Name: "path", Desc: "Path to backup file (default: latest from .hulak/backups/)"},
		},
		Examples: []*utils.CommandHelp{
			{Command: "hulak secrets restore", Description: "Restore the latest backup"},
			{
				Command:     "hulak secrets restore .hulak/backups/store.age.bak.2026-05-01T14-30-00",
				Description: "Restore a specific backup",
			},
			{
				Command:     "hulak secrets restore ~/backups/my.age",
				Description: "Restore from an external backup",
			},
			{Command: "hulak secrets restore --force", Description: "Skip confirmation prompt"},
		},
		Run: func(args []string) error {
			if len(args) > 1 {
				return fmt.Errorf(
					"too many arguments: got %d, expected at most 1 (path)",
					len(args),
				)
			}
			var path string
			if len(args) == 1 {
				path = args[0]
			}
			return runRestore(path, force)
		},
	}
}

// runRestore restores store.age from a backup.
func runRestore(backupPath string, force bool) error {
	if err := cli.RequireVaultProject(); err != nil {
		return err
	}

	resolvedPath, err := resolveRestorePath(backupPath)
	if err != nil {
		return err
	}

	// Decrypt to validate identity + version + JSON structure.
	// The age library handles format detection — non-age files
	// produce a clear decrypt error without hardcoding the header.
	store, err := vault.ReadStore(resolvedPath)
	if err != nil {
		return err
	}

	// Confirmation prompt
	if !force {
		utils.PrintInfoStderr(fmt.Sprintf("Restoring from %s", filepath.Base(resolvedPath)))
		confirmed, confirmErr := utils.ConfirmAction(
			"This will overwrite your current store.age. Continue? [y/N] ",
		)
		if confirmErr != nil {
			return fmt.Errorf("failed to read input: %w", confirmErr)
		}
		if !confirmed {
			utils.PrintInfoStderr("Restore cancelled")
			return nil
		}
	}

	// Re-encrypt to current recipients and write atomically
	return vault.WithStoreLock(func() error {
		if err := vault.WriteStoreToRecipients(store); err != nil {
			return fmt.Errorf("failed to write restored store: %w", err)
		}
		utils.PrintSuccessStderr("Store restored from backup")
		return nil
	})
}

// resolveRestorePath determines which backup to restore.
// If backupPath is empty, finds the latest backup in .hulak/backups/.
func resolveRestorePath(backupPath string) (string, error) {
	if backupPath != "" {
		if !utils.FileExists(backupPath) {
			return "", fmt.Errorf("backup file not found: %s", backupPath)
		}
		return backupPath, nil
	}

	backupsDir, err := vault.BackupsDir()
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf(
				"no backups found in %s/",
				utils.HiddenProjectName+"/backups",
			)
		}
		return "", fmt.Errorf("failed to read backups directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), backupPrefix) {
			names = append(names, entry.Name())
		}
	}

	if len(names) == 0 {
		return "", fmt.Errorf(
			"no backups found in %s/",
			utils.HiddenProjectName+"/backups",
		)
	}

	// Lexicographic sort — timestamps sort naturally, pick the latest
	sort.Strings(names)
	latest := names[len(names)-1]

	return filepath.Join(backupsDir, latest), nil
}
