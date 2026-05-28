// Builds the `hulak secrets identity` subgroup that holds every command
// dealing with age identities (your private key) and recipients (public
// keys allowed to decrypt the vault).
//
// The leaf factories themselves live in their feature files (env_keys.go,
// env_gen_identity.go, env_list_identity.go, env_rotate_key.go,
// env_recipients.go). This file is purely the subgroup assembler.
package secrets

import (
	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/utils"
)

func newEnvIdentityCmd() *cli.Command {
	return &cli.Command{
		Name:  "identity",
		Short: "Manage age identities and recipients",
		Long: "Manage age identities and recipients for the encrypted vault.\n\n" +
			"An identity is a private key (yours) that can decrypt the vault.\n" +
			"A recipient is a public key (yours or a teammate's) that the vault\n" +
			"is encrypted to. Adding a recipient grants decrypt access. Removing\n" +
			"one followed by `hulak secrets sync` revokes it.\n\n" +
			"Operations here are vault-global and do not take --env.\n\n" +
			"Two commands sound similar but do different things:\n" +
			"  `secrets identity rotate`  generates a NEW keypair and swaps it\n" +
			"                             into recipients. Use after compromise.\n" +
			"  `secrets sync`             re-encrypts the store to the current\n" +
			"                             recipients without changing any keys.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets identity list",
				Description: "Show identities on this machine that can decrypt the vault",
			},
			{
				Command:     "hulak secrets identity list-recipients",
				Description: "Show every public key the vault is encrypted to",
			},
			{
				Command:     "hulak secrets identity generate --name alice-laptop",
				Description: "Generate a keypair and auto-register it as a recipient",
			},
			{
				Command:     "hulak secrets identity import ~/backup-identity.txt",
				Description: "Restore an age private key from a backup file",
			},
			{
				Command:     "hulak secrets identity export --out ~/backup-identity.txt",
				Description: "Save your private key to a 0600 file for backup",
			},
			{
				Command:     "hulak secrets identity add-recipient --github alice --name Alice",
				Description: "Grant Alice decrypt access using her GitHub SSH keys",
			},
			{
				Command:     "hulak secrets identity remove-recipient Alice",
				Description: "Revoke Alice's access (matches by name label or pubkey)",
			},
			{
				Command:     "hulak secrets identity rotate",
				Description: "Rotate your age keypair and re-encrypt the store",
			},
		},
		SubCommands: []*cli.Command{
			newIdentityAddRecipientCmd(),
			newIdentityExportCmd(),
			newIdentityGenCmd(),
			newIdentityImportCmd(),
			newIdentityListCmd(),
			newIdentityListRecipientsCmd(),
			newIdentityRemoveRecipientCmd(),
			newIdentityRotateCmd(),
		},
	}
}
