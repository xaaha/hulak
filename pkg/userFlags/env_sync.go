// Contains command factory and handler for hulak secrets sync.
package userflags

import (
	"fmt"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

func newEnvSyncCmd() *command {
	return &command{
		Name:  "sync",
		Short: "Re-encrypt the store to current recipients",
		Long: "Re-encrypt store.age to match the current recipients.txt.\n\n" +
			"Use this after manually editing recipients.txt. Not needed after\n" +
			"add-recipient or remove-recipient — those re-encrypt automatically.\n\n" +
			"`sync` only re-encrypts; it never changes keys. To rotate a\n" +
			"compromised keypair, use `hulak secrets identity rotate` instead —\n" +
			"that's the only command in hulak that issues a new private key.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets sync",
				Description: "Re-encrypt store to match recipients.txt",
			},
		},
		Run: runSync,
	}
}

// runSync handles `hulak secrets sync`.
// Re-encrypts store.age to match the current recipients.txt.
// Useful after manually editing recipients.txt.
func runSync(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}
	if err := requireVaultProject(); err != nil {
		return err
	}

	return vault.WithStoreLock(func() error {
		store, err := vault.ReadStore()
		if err != nil {
			return err
		}

		if err := vault.WriteStoreToRecipients(store); err != nil {
			return err
		}

		utils.PrintSuccessStderr("Re-encrypted store to current recipients")
		return nil
	})
}
