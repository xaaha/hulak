// Contains command factory and handler for hulak secrets rotate (sync/reencrypt).
package userflags

import (
	"fmt"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

func newEnvSyncCmd() *command {
	return &command{
		Name:    "rotate",
		Aliases: []string{"sync", "reencrypt"},
		Short:   "Re-encrypt the store to current recipients",
		Long: "Re-encrypt store.age to match the current recipients.txt.\n\n" +
			"Use this after manually editing recipients.txt. Not needed after\n" +
			"add-recipient or remove-recipient — those re-encrypt automatically.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets rotate",
				Description: "Re-encrypt store to match recipients.txt",
			},
			{
				Command:     "hulak secrets sync",
				Description: "Same as rotate (alias)",
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
		identity, err := vault.ResolveIdentity()
		if err != nil {
			return fmt.Errorf("failed to load identity: %w", err)
		}

		store, err := vault.ReadStore(identity)
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
