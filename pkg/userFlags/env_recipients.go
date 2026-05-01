// Contains command factories and handlers for hulak env add-recipient, remove-recipient, and list-recipients.
package userflags

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

const (
	// RotationReminder is the security notice printed after removing a recipient.
	RotationReminder = "Note: %s can still decrypt copies of store.age from before this point. Rotate upstream secrets if compromise is suspected."
	// Length after which Ellipsis start
	EllipsisLength = 20
)

// newEnvAddRecipientCmd returns the "add-recipient" command with its flag set.
func newEnvAddRecipientCmd() *command {
	addRecipientFs := flag.NewFlagSet("env add-recipient", flag.ContinueOnError)
	addRecipientName := addRecipientFs.String("name", "", "Human-readable label for the recipient")
	addRecipientStdin := addRecipientFs.Bool("stdin", false, "Read keys from stdin (one per line)")

	return &command{
		Name:  "add-recipient",
		Short: "Add a recipient for shared vault access",
		Long:  "Add an age public key as a recipient so another user can decrypt the vault.\n\nThe vault is re-encrypted to all current recipients plus the new one.\nUse --name to add a human-readable label.",
		Flags: addRecipientFs,
		Args:  []argDef{{Name: "public-key", Required: true, Desc: "Age public key to add"}},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak env add-recipient age1ql3z...",
				Description: "Add a teammate's public key",
			},
			{
				Command:     "hulak env add-recipient age1ql3z... --name Alice",
				Description: "Add with a label",
			},
			{
				Command:     "cat keys.txt | hulak env add-recipient --stdin --name Team",
				Description: "Add multiple keys from stdin",
			},
		},
		Run: func(args []string) error { return runAddRecipient(args, *addRecipientName, *addRecipientStdin) },
	}
}

// resolveRecipientKeys returns public keys to add. From --stdin (one per line,
// blank lines and # comments ignored) or from positional arg. Error if both.
func resolveRecipientKeys(args []string, useStdin bool) ([]string, error) {
	if useStdin && len(args) > 0 {
		return nil, errors.New("cannot use both --stdin and a positional key — pick one")
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
		return nil, errors.New("missing required argument: public-key")
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("too many arguments: got %d, expected 1 (public-key)", len(args))
	}
	return []string{args[0]}, nil
}

// runAddRecipient handles `hulak env add-recipient <public-key> [--name n] [--stdin]`.
func runAddRecipient(args []string, name string, useStdin bool) error {
	pubKeys, err := resolveRecipientKeys(args, useStdin)
	if err != nil {
		return err
	}

	return vault.WithStoreLock(func() error {
		// Read current entries
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

		// Add each key
		for _, pubKey := range pubKeys {
			entries, err = vault.AddRecipientEntry(entries, pubKey, name)
			if err != nil {
				return err
			}
		}

		// Re-encrypt store to all recipients including new ones
		identity, err := vault.ResolveIdentity()
		if err != nil {
			return fmt.Errorf("failed to load identity: %w", err)
		}
		store, err := vault.ReadStore(identity)
		if err != nil {
			return err
		}

		// Write store first (prefer store updated + stale recipients
		//	over recipients updated + stale store)
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

		if len(pubKeys) == 1 {
			keyPrefix := pubKeys[0]
			if len(keyPrefix) > EllipsisLength {
				keyPrefix = keyPrefix[:EllipsisLength] + utils.Ellipsis
			}
			utils.PrintSuccessStderr(fmt.Sprintf("Added recipient %s", keyPrefix))
		} else {
			utils.PrintSuccessStderr(fmt.Sprintf("Added %d recipients", len(pubKeys)))
		}
		return nil
	})
}

// newEnvRemoveRecipientCmd returns the "remove-recipient" command.
func newEnvRemoveRecipientCmd() *command {
	return &command{
		Name:  "remove-recipient",
		Short: "Remove a recipient",
		Long:  "Remove an age public key from the recipient list and re-encrypt the vault.\n\nMatch by key string or name label. Refuses to remove the last recipient.\nNote: removed users can still decrypt copies from before this point.",
		Args: []argDef{
			{
				Name:     "key-or-name",
				Required: true,
				Desc:     "Age public key or name label to remove",
			},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak env remove-recipient age1ql3z...",
				Description: "Remove by public key",
			},
			{
				Command:     "hulak env remove-recipient Alice",
				Description: "Remove by name label",
			},
		},
		Run: runRemoveRecipient,
	}
}

// runRemoveRecipient handles `hulak env remove-recipient <key-or-name>`.
func runRemoveRecipient(args []string) error {
	if len(args) == 0 {
		return errors.New("missing required argument: public-key or name label")
	}
	if len(args) > 1 {
		return fmt.Errorf("too many arguments: got %d, expected 1", len(args))
	}
	query := args[0]

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
		identity, err := vault.ResolveIdentity()
		if err != nil {
			return fmt.Errorf("failed to load identity: %w", err)
		}
		store, err := vault.ReadStore(identity)
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

// newEnvListRecipientsCmd returns the "list-recipients" command.
func newEnvListRecipientsCmd() *command {
	return &command{
		Name:  "list-recipients",
		Short: "List all recipients",
		Long:  "Show all age public keys that can decrypt the vault, with labels.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak env list-recipients",
				Description: "Show all recipients with names and key prefixes",
			},
		},
		Run: runListRecipients,
	}
}

// runListRecipients handles `hulak env list-recipients`.
func runListRecipients(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
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
		rows[idx] = []string{name, keyPrefix}
	}
	return utils.PrintTable(
		os.Stdout,
		utils.StdoutHeaders([]string{"NAME", "KEY"}),
		rows,
		0,
	)
}
