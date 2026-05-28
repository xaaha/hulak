// Contains command factory and handler for hulak secrets identity rotate.
package userflags

import (
	"fmt"
	"os"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

func newIdentityRotateCmd() *cli.Command {
	return &cli.Command{
		Name:  "rotate",
		Short: "Rotate your age identity (keypair)",
		Long: "Generate a new age keypair, swap it in recipients.txt, and re-encrypt the store.\n\n" +
			"This is the \"my key was compromised\" command. After rotation:\n" +
			"  - Your old private key is backed up to identity.txt.old\n" +
			"  - The store is re-encrypted to the new key\n" +
			"  - Teammates' keys are untouched\n\n" +
			"Distinct from 'hulak secrets sync' (which re-encrypts without changing keys).",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets identity rotate",
				Description: "Rotate your identity and re-encrypt the store",
			},
		},
		Run: runRotateKey,
	}
}

// rotationState holds all the data collected during a key rotation.
type rotationState struct {
	store          *vault.Store
	newKey         vault.AgeKey
	updatedEntries []vault.RecipientEntry
	swapOldKey     string
	replacedCount  int
	recovering     bool
}

// runRotateKey handles `hulak secrets identity rotate`.
// Generates a new age keypair, swaps it in recipients.txt, re-encrypts the
// store, and backs up the old private key to identity.txt.old.
func runRotateKey(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}
	if err := cli.RequireVaultProject(); err != nil {
		return err
	}

	if !vault.IdentityExists() {
		return fmt.Errorf(
			"rotate requires an age identity (identity.txt)\n\n" +
				"This vault uses SSH keys. To rotate:\n" +
				"  1. Generate new SSH key: ssh-keygen -t ed25519\n" +
				"  2. Add new key: hulak secrets identity add-recipient \"$(cat ~/.ssh/new_key.pub)\"\n" +
				"  3. Re-encrypt: hulak secrets sync\n" +
				"  4. Remove old key: hulak secrets identity remove-recipient \"ssh-ed25519 <old>\"\n" +
				"  5. Verify: hulak doctor",
		)
	}

	if os.Getenv(utils.MasterKey) != "" {
		return fmt.Errorf(
			"identity rotate cannot run while %s is set — "+
				"run 'hulak secrets identity import' to move your key to disk first",
			utils.MasterKey,
		)
	}

	return vault.WithStoreLock(func() error {
		rs, err := prepareRotation()
		if err != nil {
			return err
		}

		if err := writeRotation(rs); err != nil {
			return err
		}

		printRotationSummary(rs)
		return nil
	})
}

// prepareRotation loads the current identity, decrypts the store (with .old
// fallback for crash recovery), generates a new keypair, and builds the
// updated recipient list. No disk writes happen here.
func prepareRotation() (*rotationState, error) {
	currentIdentity, err := vault.LoadIdentity()
	if err != nil {
		return nil, fmt.Errorf("failed to load identity: %w", err)
	}

	store, recovering, err := decryptForRotation(currentIdentity)
	if err != nil {
		return nil, err
	}
	if recovering {
		utils.PrintWarningStderr("Detected interrupted rotation — resuming")
	}

	newKey, err := resolveNewKey(recovering)
	if err != nil {
		return nil, err
	}

	swapOldKey := currentIdentity.Recipient().String()
	if recovering {
		oldIdentity, loadErr := vault.LoadIdentityOld()
		if loadErr != nil {
			return nil, fmt.Errorf("recovery failed — cannot load backup identity: %w", loadErr)
		}
		swapOldKey = oldIdentity.Recipient().String()
	}

	updatedEntries, replacedCount, err := swapRecipients(swapOldKey, newKey.Recipient.String())
	if err != nil {
		return nil, err
	}

	return &rotationState{
		store:          store,
		newKey:         newKey,
		updatedEntries: updatedEntries,
		swapOldKey:     swapOldKey,
		replacedCount:  replacedCount,
		recovering:     recovering,
	}, nil
}

// resolveNewKey either loads the existing keypair (recovery) or generates a fresh one.
func resolveNewKey(recovering bool) (vault.AgeKey, error) {
	if recovering {
		key, err := vault.LoadKeypair()
		if err != nil {
			return vault.AgeKey{}, fmt.Errorf("failed to load new identity for recovery: %w", err)
		}
		return key, nil
	}
	key, err := vault.GenerateKeyPair()
	if err != nil {
		return vault.AgeKey{}, fmt.Errorf("failed to generate new keypair: %w", err)
	}
	return key, nil
}

// swapRecipients reads recipients.txt, finds entries matching oldKey, and
// replaces them with newKey. Returns the updated entries and count of replaced keys.
func swapRecipients(oldKey, newKey string) ([]vault.RecipientEntry, int, error) {
	recipientPath, err := vault.RecipientsFilePath()
	if err != nil {
		return nil, 0, err
	}
	data, err := os.ReadFile(recipientPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read recipients: %w", err)
	}
	entries, err := vault.ParseRecipientsFileContent(data)
	if err != nil {
		return nil, 0, err
	}

	entryName := extractRecipientName(entries, oldKey)

	return vault.SwapRecipientKey(entries, oldKey, newKey, entryName)
}

// extractRecipientName finds the name from the first entry matching key,
// stripping the date suffix added by FormatRecipientName.
func extractRecipientName(entries []vault.RecipientEntry, key string) string {
	for _, e := range entries {
		if e.Key == key {
			name, _ := vault.ParseRecipientName(e.Name)
			return name
		}
	}
	return ""
}

// writeRotation performs the identity-first disk writes:
// backup identity → new identity → store.age → recipients.txt
func writeRotation(rs *rotationState) error {
	recipients, err := vault.RecipientsFromEntries(rs.updatedEntries)
	if err != nil {
		return err
	}

	if !rs.recovering {
		if err := vault.BackupIdentity(); err != nil {
			return fmt.Errorf("failed to back up identity: %w", err)
		}
		if err := vault.SetIdentity(rs.newKey.Identity.String()); err != nil {
			return fmt.Errorf("failed to write new identity: %w", err)
		}
	}

	if err := vault.WriteStore(rs.store, recipients...); err != nil {
		return fmt.Errorf("failed to re-encrypt store: %w", err)
	}

	if err := vault.SaveRecipients(rs.updatedEntries); err != nil {
		return fmt.Errorf("failed to write recipients: %w", err)
	}

	return nil
}

// printRotationSummary outputs the rotation result to stderr.
func printRotationSummary(rs *rotationState) {
	oldBackupPath, _ := vault.IdentityOldPath()

	if rs.recovering {
		utils.PrintSuccessStderr("Completed interrupted key rotation")
	} else {
		utils.PrintSuccessStderr("Rotated identity key")
	}
	utils.PrintInfoStderr(fmt.Sprintf("  Old public key: %s", rs.swapOldKey))
	utils.PrintInfoStderr(
		fmt.Sprintf(
			"  New public key: %s  <- share this with your team",
			rs.newKey.Recipient.String(),
		),
	)
	utils.PrintInfoStderr(fmt.Sprintf("  Old private key backed up to %s", oldBackupPath))
	utils.PrintInfoStderr(
		fmt.Sprintf("  Replaced %d old key(s) in recipients.txt", rs.replacedCount),
	)
	utils.PrintWarningStderr(
		"Your old private key may still decrypt copies of store.age from before this rotation. " +
			"Rotate upstream secrets if compromise is suspected.",
	)
}

// decryptForRotation attempts to decrypt the store with the current identity.
// If that fails and an identity.txt.old exists, tries the backup (interrupted
// rotation recovery). Returns the store, whether we're in recovery mode, and error.
func decryptForRotation(currentIdentity *age.X25519Identity) (*vault.Store, bool, error) {
	store, err := vault.DecryptStore(currentIdentity)
	if err == nil {
		return store, false, nil
	}

	// Current identity failed. Try .old for interrupted rotation recovery.
	oldIdentity, oldErr := vault.LoadIdentityOld()
	if oldErr != nil {
		return nil, false, fmt.Errorf(
			"cannot decrypt store with current identity: %w", err,
		)
	}

	store, err = vault.DecryptStore(oldIdentity)
	if err != nil {
		return nil, false, fmt.Errorf(
			"cannot decrypt store with current or backup identity — both keys failed",
		)
	}

	return store, true, nil
}
