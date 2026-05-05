// Contains command factories and handlers for hulak secrets import-key and export-key.
package userflags

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// newEnvImportKeyCmd returns the command struct for `hulak secrets import-key`.
func newEnvImportKeyCmd() *command {
	fs := flag.NewFlagSet("env import-key", flag.ContinueOnError)
	importKeyStdin := fs.Bool("stdin", false, "Read key from stdin")
	importKeyForce := fs.Bool("force", false, "Overwrite existing identity file")

	return &command{
		Name:    "import-key",
		Aliases: []string{"import-identity"},
		Short:   "Import an age identity (private key)",
		Long:    "Import an age private key from a file or stdin into the hulak config directory.\n\nValidates the key before writing. Refuses to overwrite an existing identity unless --force is passed.\nAtomic write (tmp + rename) so an interrupted import can never corrupt an existing identity.",
		Flags:   fs,
		Args: []argDef{
			{Name: "path", Desc: "Path to the identity file (omit with --stdin)"},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets import-key ~/backup-identity.txt",
				Description: "Import from a backup file",
			},
			{
				Command:     "hulak secrets import-key ~/backup.txt --force",
				Description: "Overwrite existing identity",
			},
			{
				Command:     "echo \"AGE-SECRET-KEY-1QF...\" | hulak secrets import-key --stdin",
				Description: "Import from stdin (password manager pipe)",
			},
		},
		Run: func(args []string) error { return runImportKey(args, *importKeyStdin, *importKeyForce) },
	}
}

// runImportKey handles `hulak secrets import-key [path] [--stdin] [--force]`.
func runImportKey(args []string, useStdin, force bool) error {
	if useStdin && len(args) > 0 {
		return errors.New("cannot use both --stdin and a positional path — pick one")
	}

	var raw string
	switch {
	case useStdin:
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		raw = string(data)

	case len(args) > 0:
		if len(args) > 1 {
			return fmt.Errorf("too many arguments: got %d, expected 1 (path)", len(args))
		}
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", args[0], err)
		}
		raw = string(data)

	default:
		return errors.New("missing required argument: path (or use --stdin)")
	}

	if err := vault.ImportKey(raw, force); err != nil {
		return err
	}

	identityPath, err := vault.IdentityPath()
	if err != nil {
		return err
	}
	utils.PrintSuccessStderr(fmt.Sprintf("Identity imported to %s", identityPath))
	return nil
}

// newEnvExportKeyCmd returns the command struct for `hulak secrets export-key`.
func newEnvExportKeyCmd() *command {
	fs := flag.NewFlagSet("env export-key", flag.ContinueOnError)
	var outVal string
	fs.StringVar(&outVal, "out", "", "Write key to file instead of stdout (mode 0600)")
	fs.StringVar(&outVal, "o", "", "Write key to file instead of stdout (mode 0600)")

	return &command{
		Name:    "export-key",
		Aliases: []string{"export-identity"},
		Short:   "Export the age identity (private key)",
		Long:    "Print the age private key to stdout for backup or transfer.\n\nUse --out to write directly to a file with 0600 permissions instead of stdout.",
		Flags:   fs,
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets export-key",
				Description: "Print the private key (with security warning on stderr)",
			},
			{
				Command:     "hulak secrets export-key --out ~/backup-identity.txt",
				Description: "Save to a file with 0600 permissions",
			},
		},
		Run: func(args []string) error { return runExportKey(args, outVal) },
	}
}

// runExportKey handles `hulak secrets export-key [--out path]`.
func runExportKey(args []string, outPath string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}

	key, err := vault.ExportKey()
	if err != nil {
		return err
	}

	if outPath != "" {
		if utils.FileExists(outPath) {
			return fmt.Errorf(
				"file %q already exists — use a different path or remove it first",
				outPath,
			)
		}
		if err := os.WriteFile(outPath, []byte(key+"\n"), utils.SecretPer); err != nil {
			return fmt.Errorf("failed to write key to %s: %w", outPath, err)
		}
		utils.PrintSuccessStderr(fmt.Sprintf("Key written to %s (mode 0600)", outPath))
		return nil
	}

	// Print to stdout with a security warning on stderr.
	utils.PrintWarningStderr("This is your age private key. Treat it like a password.")
	utils.PrintWarningStderr("Do not commit it or share it. Store it in a password manager.")
	fmt.Println(key)
	return nil
}
