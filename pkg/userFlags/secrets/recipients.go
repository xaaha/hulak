// Contains command factories and handlers for the recipient leaves under
// `hulak secrets identity`: add-recipient, remove-recipient, list-recipients.
package secrets

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/userFlags/cliflags"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

const (
	// RotationReminder is the security notice printed after removing a recipient.
	RotationReminder = "Note: %s can still decrypt copies of store.age from before this point. Rotate upstream secrets if compromise is suspected."
	// Length after which Ellipsis start
	EllipsisLength = 20
)

// newIdentityAddRecipientCmd returns the "add-recipient" command with its flag set.
func newIdentityAddRecipientCmd() *cli.Command {
	addRecipientFs := flag.NewFlagSet("identity add-recipient", flag.ContinueOnError)
	addRecipientName := cliflags.RegisterName(addRecipientFs, "Human-readable label for the recipient (defaults to OS username)")
	addRecipientStdin := addRecipientFs.Bool("stdin", false, "Read keys from stdin (one per line)")
	addRecipientGitHub := addRecipientFs.String(
		"github",
		"",
		"Fetch ed25519 keys from GitHub (username)",
	)
	addRecipientKeyserver := addRecipientFs.String(
		"keyserver",
		"",
		"Base URL of keyserver (e.g. https://gitlab.com)",
	)
	addRecipientAllowRSA := addRecipientFs.Bool(
		"allow-rsa",
		false,
		"Also accept ssh-rsa keys (lower security margin)",
	)

	return &cli.Command{
		Name:  "add-recipient",
		Short: "Add a recipient for shared vault access",
		Long: "Add an age or SSH public key as a recipient so another user can decrypt the vault.\n\n" +
			"The vault is re-encrypted to all current recipients plus the new one.\n" +
			"Use --name to add a human-readable label.\n" +
			"Use --github to fetch a user's SSH keys directly from GitHub.",
		Flags: addRecipientFs,
		Args: []cli.ArgDef{
			{Name: "public-key", Desc: "Age or SSH public key to add (not needed with --github)"},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets identity add-recipient age1ql3z...",
				Description: "Add a teammate's age public key",
			},
			{
				Command:     "hulak secrets identity add-recipient \"ssh-ed25519 AAAA...\" --name Alice",
				Description: "Add an SSH ed25519 public key",
			},
			{
				Command:     "hulak secrets identity add-recipient --github alice --name Alice",
				Description: "Fetch and add Alice's ed25519 keys from GitHub",
			},
			{
				Command:     "hulak secrets identity add-recipient --github alice --keyserver https://gitlab.com --name Alice",
				Description: "Fetch from a self-hosted GitLab",
			},
			{
				Command:     "cat keys.txt | hulak secrets identity add-recipient --stdin --name Team",
				Description: "Add multiple keys from stdin",
			},
		},
		Run: func(args []string) error {
			return runAddRecipient(
				args,
				*addRecipientName,
				*addRecipientStdin,
				*addRecipientGitHub,
				*addRecipientKeyserver,
				*addRecipientAllowRSA,
			)
		},
	}
}

// resolveRecipientKeys returns public keys to add from --stdin, --github, or positional arg.
func resolveRecipientKeys(
	args []string,
	useStdin bool,
	gitHubUser, keyserverURL string,
	allowRSA bool,
) ([]string, error) {
	sources := 0
	if useStdin {
		sources++
	}
	if gitHubUser != "" {
		sources++
	}
	if len(args) > 0 {
		sources++
	}
	if sources > 1 {
		return nil, errors.New("use exactly one of: positional key, --stdin, or --github")
	}

	if gitHubUser != "" {
		baseURL := vault.GitHubKeysBase
		if keyserverURL != "" {
			baseURL = keyserverURL
		}
		url := vault.KeyserverKeysURL(baseURL, gitHubUser)
		return fetchAndFilterKeys(url, allowRSA)
	}

	if useStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read stdin: %w", err)
		}
		var keys []string
		for line := range strings.SplitSeq(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, utils.Comment) {
				continue
			}
			keys = append(keys, line)
		}
		if len(keys) == 0 {
			return nil, errors.New("no keys found in stdin")
		}
		return keys, nil
	}

	if len(args) == 0 {
		return nil, errors.New("missing required argument: public-key (or use --github <username>)")
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments: got %d, expected 1 (public-key)", len(args))
	}
	return []string{args[0]}, nil
}

// fetchAndFilterKeys fetches keys from a URL, filters by type, and returns
// the keys to add. Prints warnings for skipped key types.
func fetchAndFilterKeys(url string, allowRSA bool) ([]string, error) {
	allKeys, err := vault.FetchKeysFromURL(url, nil)
	if err != nil {
		return nil, err
	}

	ed25519Keys, rsaKeys, skipped := vault.FilterKeysByType(allKeys)

	var keys []string
	keys = append(keys, ed25519Keys...)

	if len(rsaKeys) > 0 {
		if allowRSA {
			utils.PrintWarningStderr(fmt.Sprintf(
				"Including %d ssh-rsa key(s) (lower security margin)", len(rsaKeys),
			))
			keys = append(keys, rsaKeys...)
		} else {
			utils.PrintWarningStderr(fmt.Sprintf(
				"Skipped %d ssh-rsa key(s) — use --allow-rsa to include them", len(rsaKeys),
			))
		}
	}

	if len(skipped) > 0 {
		utils.PrintWarningStderr(fmt.Sprintf(
			"Skipped %d unsupported key(s)", len(skipped),
		))
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no ed25519 keys found — ask the user to upload an ed25519 SSH key")
	}

	return keys, nil
}

// runAddRecipient handles `hulak secrets identity add-recipient`.
func runAddRecipient(
	args []string,
	name string,
	useStdin bool,
	gitHubUser, keyserverURL string,
	allowRSA bool,
) error {
	if err := cli.RequireVaultProject(); err != nil {
		return err
	}

	pubKeys, err := resolveRecipientKeys(args, useStdin, gitHubUser, keyserverURL, allowRSA)
	if err != nil {
		return err
	}

	return vault.WithStoreLock(func() error {
		recipPath, err := vault.RecipientsFilePath()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(recipPath)
		if err != nil {
			return fmt.Errorf("failed to read recipients file: %w", err)
		}
		entries, err := vault.ParseRecipientsFileContent(data)
		if err != nil {
			return err
		}

		// User-supplied label wins; otherwise fall back to the GitHub username
		// when the keys came from a github fetch.
		recipientName := cliflags.ResolveRecipientName(name, gitHubUser)

		added := 0
		for _, pubKey := range pubKeys {
			newEntries, addErr := vault.AddRecipientEntry(entries, pubKey, recipientName, allowRSA)
			if addErr != nil {
				if strings.Contains(addErr.Error(), "already in recipients") {
					continue // skip duplicates silently
				}
				return addErr
			}
			entries = newEntries
			added++
		}

		if added == 0 {
			utils.PrintWarningStderr("No new recipients added (all duplicates)")
			return nil
		}

		// Re-encrypt store to all recipients including new ones
		store, err := vault.ReadStore()
		if err != nil {
			return err
		}

		// Write store first (prefer store updated + stale recipients
		// over recipients updated + stale store)
		recipients, err := vault.RecipientsFromEntries(entries)
		if err != nil {
			return err
		}
		if err := vault.WriteStore(store, recipients...); err != nil {
			return err
		}
		if err := vault.SaveRecipients(entries); err != nil {
			return err
		}

		if added == 1 {
			utils.PrintSuccessStderr("Added 1 recipient")
		} else {
			utils.PrintSuccessStderr(fmt.Sprintf("Added %d recipients", added))
		}
		return nil
	})
}

// newIdentityRemoveRecipientCmd returns the "remove-recipient" command.
func newIdentityRemoveRecipientCmd() *cli.Command {
	return &cli.Command{
		Name:  "remove-recipient",
		Short: "Remove a recipient",
		Long:  "Remove an age public key from the recipient list and re-encrypt the vault.\n\nMatch by key string or name label. Refuses to remove the last recipient.\nNote: removed users can still decrypt copies from before this point.",
		Args: []cli.ArgDef{
			{
				Name:     "key-or-name",
				Required: true,
				Desc:     "Age public key or name label to remove",
			},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets identity remove-recipient age1ql3z...",
				Description: "Remove by public key",
			},
			{
				Command:     "hulak secrets identity remove-recipient Alice",
				Description: "Remove by name label",
			},
		},
		Run: runRemoveRecipient,
	}
}

// runRemoveRecipient handles `hulak secrets identity remove-recipient <key-or-name>`.
func runRemoveRecipient(args []string) error {
	if len(args) == 0 {
		return errors.New("missing required argument: public-key or name label")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments: got %d, expected 1", len(args))
	}
	query := args[0]

	if err := cli.RequireVaultProject(); err != nil {
		return err
	}

	return vault.WithStoreLock(func() error {
		recipPath, err := vault.RecipientsFilePath()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(recipPath)
		if err != nil {
			return fmt.Errorf("failed to read recipients file: %w", err)
		}
		entries, err := vault.ParseRecipientsFileContent(data)
		if err != nil {
			return err
		}

		entries, removed, err := vault.RemoveRecipientEntry(entries, query)
		if err != nil {
			return err
		}
		if !removed {
			utils.PrintWarningStderr(
				fmt.Sprintf("No recipient matching %q found — no changes made", query),
			)
			return nil
		}

		// Re-encrypt to remaining recipients
		store, err := vault.ReadStore()
		if err != nil {
			return err
		}

		// Write store first (same atomicity reasoning as add-recipient)
		recipients, err := vault.RecipientsFromEntries(entries)
		if err != nil {
			return err
		}
		if err := vault.WriteStore(store, recipients...); err != nil {
			return err
		}
		if err := vault.SaveRecipients(entries); err != nil {
			return err
		}

		utils.PrintSuccessStderr("Removed recipient")
		utils.PrintWarningStderr(fmt.Sprintf(RotationReminder, query))
		return nil
	})
}

// newIdentityListRecipientsCmd returns the "list-recipients" command.
func newIdentityListRecipientsCmd() *cli.Command {
	return &cli.Command{
		Name:  "list-recipients",
		Short: "List all recipients",
		Long:  "Show all age public keys that can decrypt the vault, with labels.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets identity list-recipients",
				Description: "Show all recipients with names and key prefixes",
			},
		},
		Run: runListRecipients,
	}
}

// runListRecipients handles `hulak secrets identity list-recipients`.
func runListRecipients(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}
	if err := cli.RequireVaultProject(); err != nil {
		return err
	}

	recipPath, err := vault.RecipientsFilePath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(recipPath)
	if err != nil {
		return fmt.Errorf("failed to read recipients file: %w", err)
	}
	entries, err := vault.ParseRecipientsFileContent(data)
	if err != nil {
		return err
	}

	rows := make([][]string, len(entries))
	for idx, entry := range entries {
		name := entry.Name
		if name == "" {
			name = "(no label)"
		}
		keyPrefix := entry.Key
		if len(keyPrefix) > EllipsisLength {
			keyPrefix = keyPrefix[:EllipsisLength] + utils.Ellipsis
		}
		kt := vault.ClassifyKeyType(entry.Key)
		rows[idx] = []string{name, string(kt), keyPrefix}
	}
	return utils.PrintTable(
		os.Stdout,
		utils.StdoutHeaders([]string{"NAME", "TYPE", "KEY"}),
		rows,
		0,
	)
}
