// Contains command factories and handlers for hulak secrets import-key and export-key.
package userflags

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// newEnvImportKeyCmd returns the command struct for `hulak secrets import-key`.
func newEnvImportKeyCmd() *command {
	fs := flag.NewFlagSet("env import-key", flag.ContinueOnError)
	importKeyStdin := fs.Bool("stdin", false, "Read key from stdin")
	importKeyForce := fs.Bool("force", false, "Overwrite existing identity file")
	importKeyName := registerNameFlag(
		fs,
		"Auto-register the imported key as a recipient with this label (requires an existing decrypt path; defaults label to OS username)",
	)

	return &command{
		Name:    "import-key",
		Aliases: []string{"import-identity"},
		Short:   "Import an age identity (private key)",
		Long: "Import an age private key from a file or stdin and save it to hulak's\n" +
			"config directory so it can decrypt the vault.\n\n" +
			"Inside a vault project, hulak first checks that the key can decrypt\n" +
			"store.age. This catches the common 'wrong file' mistake. Outside a vault,\n" +
			"the check is skipped automatically.\n\n" +
			"Pass --name to auto-register the imported key as a recipient (requires\n" +
			"another working identity, e.g. SSH, that can decrypt store.age right now).\n" +
			"This skips the wrong-file check because the key is being added to the\n" +
			"vault as part of the same operation.\n\n" +
			"Will not overwrite an existing identity unless --force is passed.",
		Flags: fs,
		Args: []argDef{
			{Name: "path", Desc: "Path to the identity file (omit with --stdin)"},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets import-key ~/backup-identity.txt",
				Description: "Restore a key that's already a recipient of the vault",
			},
			{
				Command:     "hulak secrets import-key ~/age-key.txt --name alice-laptop",
				Description: "Import + auto-register as recipient (needs another working identity)",
			},
			{
				Command:     "hulak secrets import-key ~/backup.txt --force",
				Description: "Overwrite an existing identity file",
			},
			{
				Command:     "echo \"AGE-SECRET-KEY-1QF...\" | hulak secrets import-key --stdin",
				Description: "Import from stdin (password manager pipe)",
			},
		},
		Run: func(args []string) error {
			return runImportKey(args, *importKeyStdin, *importKeyForce, *importKeyName)
		},
	}
}

// runImportKey handles `hulak secrets import-key [path] [--stdin] [--force] [--name LABEL]`.
//
// Two modes, picked by --name:
//
//   - With --name: STATE-CHANGING. The user is opting in to add this key to
//     the vault. Uses whatever currently decrypts store.age (SSH, master
//     key, or another identity) to re-encrypt the store including the new
//     pubkey and append to recipients.txt. The key becomes a real recipient
//     as part of this command — no wrong-file check needed afterwards.
//
//   - Without --name: READ-ONLY. Verifies the candidate already decrypts
//     store.age before saving it. For restoring a backup of a key that's
//     already a vault recipient. Touches nothing in the vault; just reads.
//     Catches "wrong file" mistakes before they surface later as confusing
//     decrypt failures.
//
// Why two branches instead of always self-registering? The restore case
// shouldn't mutate the vault — silently re-encrypting store.age on every
// import would surprise users who expected a non-destructive operation,
// and registration also requires being inside a vault project (pre-clone
// staging would break). --name is the explicit consent signal for state
// change.
//
// Outside a vault (no .hulak/ or no store.age), the read-only check
// auto-skips — useful for pre-clone staging. --name still requires a vault.
func runImportKey(args []string, useStdin, force bool, name string) error {
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

	// --name → state-changing self-register; otherwise → read-only verify.
	// See function doc for why we don't always self-register.
	if name != "" {
		if err := registerImportedKeyAsRecipient(raw, name); err != nil {
			return err
		}
	} else if err := validateImportAgainstVault(raw); err != nil {
		return err
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

// registerImportedKeyAsRecipient parses raw, derives the public key, and adds
// it to the vault as a recipient using whatever identity currently decrypts
// store.age. Used by import-key --name and gen-identity --name.
func registerImportedKeyAsRecipient(raw, name string) error {
	if err := requireVaultProject(); err != nil {
		return fmt.Errorf("--name needs a vault project in cwd: %w", err)
	}
	candidate, err := vault.ParseImportKey(raw)
	if err != nil {
		return err
	}
	return registerPubKeyAsRecipient(candidate.Recipient().String(), name)
}

// registerPubKeyAsRecipient adds pubKey to recipients.txt and re-encrypts the
// store. Wraps vault.AddRecipientAndReencrypt with the OS-username fallback
// and a clearer "already a recipient" message.
func registerPubKeyAsRecipient(pubKey, name string) error {
	added, err := vault.AddRecipientAndReencrypt(pubKey, resolveRecipientName(name, utils.Username()))
	if err != nil {
		return err
	}
	if added {
		utils.PrintSuccessStderr(fmt.Sprintf("Registered as recipient: %s", pubKey))
	} else {
		utils.PrintInfoStderr(fmt.Sprintf("Already a recipient: %s", pubKey))
	}
	return nil
}

// validateImportAgainstVault decrypt-tests the imported key against the local
// store.age. Returns nil when no vault is in cwd (pre-vault staging is a
// legitimate use case). Returns an actionable error when the key is parseable
// but doesn't decrypt — directing the user to add-recipient first.
func validateImportAgainstVault(raw string) error {
	storePath, err := vault.StorePath()
	if err != nil || storePath == "" || !utils.FileExists(storePath) {
		// No vault here — nothing to validate against. Pre-clone staging
		// is a legitimate use; let the import proceed.
		return nil
	}

	candidate, err := vault.ParseImportKey(raw)
	if err != nil {
		// Parse failure surfaces in vault.ImportKey with the same error;
		// defer to that for the canonical message.
		return nil
	}

	cipherText, err := os.ReadFile(storePath)
	if err != nil {
		return fmt.Errorf("failed to read store: %w", err)
	}

	if _, err := vault.DecryptText(cipherText, candidate); err != nil {
		// age's specific "no recipient match" error contains "did not match
		// any of the recipients". Any other error (truncated file, invalid
		// header, etc.) means store.age itself is the problem — don't
		// misreport that as "not a recipient."
		if !strings.Contains(err.Error(), "did not match any of the recipients") {
			return fmt.Errorf("failed to validate against store.age (possibly corrupt): %w", err)
		}
		pub := candidate.Recipient().String()
		return utils.HelpfulError(
			"this key cannot decrypt store.age (not a recipient of this vault)",
			"What to do",
			[]string{
				fmt.Sprintf("Public key derived from import: %s", pub),
				fmt.Sprintf("Ask a current vault member to add it: hulak secrets add-recipient %s", pub),
				"If you already have another working identity (SSH, master key), re-run with --name <label> to self-register",
			},
		)
	}
	return nil
}

// newEnvExportKeyCmd returns the command struct for `hulak secrets export-key`.
func newEnvExportKeyCmd() *command {
	fs := flag.NewFlagSet("env export-key", flag.ContinueOnError)
	out := registerOutputFlag(
		fs,
		"Write key to file instead of stdout (mode 0600). Directory inputs append 'identity.txt'.",
	)

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
		Run: func(args []string) error { return runExportKey(args, *out) },
	}
}

// runExportKey handles `hulak secrets export-key [--out path]`.
func runExportKey(args []string, outPath string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}
	if !vault.IdentityExists() {
		return fmt.Errorf(
			"no age identity found — this vault may use SSH keys instead\n\n" +
				"export-key only works with age identities (identity.txt)\n" +
				"SSH keys are managed with ssh-keygen and your system's key management",
		)
	}

	key, err := vault.ExportKey()
	if err != nil {
		return err
	}

	if outPath != "" {
		// Any-extension mode: user picks .txt, .pem, .key — whatever fits.
		// Paths without an extension are treated as directories; "identity.txt"
		// is appended.
		dest, err := resolveOutputPath(outPath, "identity.txt")
		if err != nil {
			return err
		}
		if utils.FileExists(dest) {
			return fmt.Errorf(
				"file %q already exists — use a different path or remove it first",
				dest,
			)
		}
		if parent := filepath.Dir(dest); parent != "." && parent != "" {
			if err := os.MkdirAll(parent, utils.DirPer); err != nil {
				return fmt.Errorf("creating parent dir for %q: %w", dest, err)
			}
		}
		if err := os.WriteFile(dest, []byte(key+"\n"), utils.SecretPer); err != nil {
			return fmt.Errorf("failed to write key to %s: %w", dest, err)
		}
		utils.PrintSuccessStderr(fmt.Sprintf("Key written to %s (mode 0600)", dest))
		return nil
	}

	// Print to stdout with a security warning on stderr.
	utils.PrintWarningStderr("This is your age private key. Treat it like a password.")
	utils.PrintWarningStderr("Do not commit it or share it. Store it in a password manager.")
	fmt.Println(key)
	return nil
}
