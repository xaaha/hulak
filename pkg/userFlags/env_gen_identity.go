// Contains command factory and handler for hulak secrets gen-identity.
package userflags

import (
	"fmt"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// newEnvGenIdentityCmd returns the command struct for `hulak secrets gen-identity`.
//
// Use case: a teammate joining an existing vault on a new machine needs an age
// keypair without the side effects of `hulak init` (which creates a fresh
// .hulak/ in cwd). This command creates just the global identity file and
// prints the public key for an existing vault member to add as a recipient.
func newEnvGenIdentityCmd() *command {
	return &command{
		Name:    "gen-identity",
		Aliases: []string{"generate-identity"},
		Short:   "Generate a new age keypair without creating a vault",
		Long: "Generate a fresh age keypair and write it to ~/.config/hulak/identity.txt.\n\n" +
			"Unlike 'hulak init', this command does not create .hulak/ files in the\n" +
			"current directory. Use it on a new machine joining an existing vault:\n" +
			"run this, send the printed pubkey to a current member, and they add it\n" +
			"with 'hulak secrets add-recipient'.\n\n" +
			"To rotate an existing identity (compromised key), use 'hulak secrets\n" +
			"rotate-key' instead — gen-identity refuses to overwrite identity.txt.\n\n" +
			"Note: the same identity can be a recipient of multiple vaults — you\n" +
			"don't need a separate keypair per project, just per machine.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets gen-identity",
				Description: "Generate a keypair and print the public key to stdout",
			},
		},
		Run: runGenIdentity,
	}
}

// runGenIdentity handles `hulak secrets gen-identity`.
//
// Refuses if ~/.config/hulak/identity.txt already exists — overwriting it
// silently would lose access to whatever vault that key was a recipient of.
// For deliberate replacement, the user should run 'rotate-key' instead.
//
// On success: prints the new public key to stdout (so it can be piped or
// captured) and the suggested add-recipient invocation to stderr.
func runGenIdentity(args []string) error {
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
				"To replace it (e.g. after a key compromise), run 'hulak secrets rotate-key'",
				"To use this identity with another vault, add its pubkey as a recipient there — the same identity can decrypt multiple vaults",
			},
		)
	}

	key, err := vault.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate keypair: %w", err)
	}
	if err := vault.SetIdentity(key.Identity.String()); err != nil {
		return fmt.Errorf("failed to write identity: %w", err)
	}

	pubKey := key.Recipient.String()

	utils.PrintSuccessStderr(fmt.Sprintf("Identity written to %s", identityPath))
	utils.PrintInfoStderr("")
	utils.PrintInfoStderr("Send your public key to a vault member and have them run:")
	utils.PrintInfoStderr(fmt.Sprintf("  hulak secrets add-recipient %s", pubKey))
	utils.PrintInfoStderr("")
	utils.PrintWarningStderr(
		"Back up the identity file — losing it means losing access to the vault.",
	)

	// Pubkey on stdout so it can be piped or captured: $(hulak secrets gen-identity)
	fmt.Println(pubKey)
	return nil
}
