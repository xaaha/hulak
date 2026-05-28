// Contains command factory and handler for hulak secrets identity generate.
package userflags

import (
	"flag"
	"fmt"

	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// newIdentityGenCmd returns the command struct for `hulak secrets identity gen`.
//
// Use case: a teammate joining an existing vault on a new machine needs an age
// keypair without the side effects of `hulak init` (which creates a fresh
// .hulak/ in cwd). Pass --name when you already have decrypt access (e.g.
// via SSH) and want the new key auto-registered as a recipient.
func newIdentityGenCmd() *cli.Command {
	fs := flag.NewFlagSet("identity gen", flag.ContinueOnError)
	genIdentityName := registerNameFlag(
		fs,
		"Auto-register the new key as a recipient with this label (requires an existing decrypt path; defaults label to OS username)",
	)

	return &cli.Command{
		Name:    "generate",
		Aliases: []string{"gen"},
		Short:   "Generate a new age keypair without creating a vault",
		Long: "Generate a fresh age keypair and write it to ~/.config/hulak/identity.txt.\n\n" +
			"Unlike 'hulak init', this command does not create .hulak/ files in the\n" +
			"current directory. Two common uses:\n\n" +
			"  - New machine, no vault access yet: run without --name, send the\n" +
			"    printed pubkey to a current vault member, and they add it with\n" +
			"    'hulak secrets identity add-recipient'.\n\n" +
			"  - Already have decrypt access (SSH, master key, another identity):\n" +
			"    run with --name to auto-register the new key as a recipient in\n" +
			"    one step. No teammate intervention needed.\n\n" +
			"Refuses to overwrite an existing identity. To rotate, use 'hulak\n" +
			"secrets identity rotate' instead.",
		Flags: fs,
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets identity generate",
				Description: "Generate a keypair and print the public key (no auto-register)",
			},
			{
				Command:     "hulak secrets identity generate --name alice-laptop",
				Description: "Generate + auto-register as a recipient (needs another working identity)",
			},
		},
		Run: func(args []string) error {
			return runGenIdentity(args, *genIdentityName)
		},
	}
}

// runGenIdentity handles `hulak secrets identity generate [--name LABEL]`.
//
// Refuses if ~/.config/hulak/identity.txt already exists — overwriting it
// silently would lose access to whatever vault that key was a recipient of.
//
// With --name: registers the new pubkey as a recipient first (using whatever
// identity currently decrypts the vault), then writes identity.txt. If the
// recipient add fails, no identity is written — atomic-ish.
//
// On success: prints the new public key to stdout (so it can be piped or
// captured) and, when --name was not set, the suggested add-recipient
// invocation to stderr.
func runGenIdentity(args []string, name string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}

	identityPath, err := vault.IdentityPath()
	if err != nil {
		return err
	}
	if utils.FileExists(identityPath) {
		return utils.HelpfulError(
			fmt.Sprintf("identity already exists at %s", identityPath),
			"What to do instead",
			[]string{
				"To replace it (e.g. after a key compromise), run 'hulak secrets identity rotate'",
				"To use this identity with another vault, add its pubkey as a recipient there — the same identity can decrypt multiple vaults",
			},
		)
	}

	key, err := vault.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate keypair: %w", err)
	}
	pubKey := key.Recipient.String()

	if name != "" {
		if err := cli.RequireVaultProject(); err != nil {
			return fmt.Errorf("--name needs a vault project in cwd: %w", err)
		}
		if err := registerPubKeyAsRecipient(pubKey, name); err != nil {
			return err
		}
	}

	if err := vault.SetIdentity(key.Identity.String()); err != nil {
		return fmt.Errorf("failed to write identity: %w", err)
	}

	utils.PrintSuccessStderr(fmt.Sprintf("Identity written to %s", identityPath))
	if name == "" {
		utils.PrintInfoStderr("")
		utils.PrintInfoStderr("Send your public key to a vault member and have them run:")
		utils.PrintInfoStderr(fmt.Sprintf("  hulak secrets identity add-recipient %s", pubKey))
		utils.PrintInfoStderr("")
	}
	utils.PrintWarningStderr(
		"Back up the identity file. Losing it means losing access to the vault.",
	)

	// Pubkey on stdout so it can be piped or captured: $(hulak secrets identity generate)
	fmt.Println(pubKey)
	return nil
}
