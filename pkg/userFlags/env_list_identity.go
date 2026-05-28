// Contains command factory and handler for hulak secrets identity list.
package userflags

import (
	"errors"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// newIdentityListCmd returns the command struct for `hulak secrets identity list`.
//
// Surfaces which identities on this machine can actually decrypt the current
// vault — replacing the per-decrypt stderr noise of "Decrypted with X" with
// an explicit user-driven inspection command. The first row is the default
// (marked with utils.Asterisk): it's what hulak would use for any read path.
func newIdentityListCmd() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Short:   "List identities that can decrypt the vault",
		Long: "Show every configured identity on this machine that can currently\n" +
			"decrypt store.age. Probes each source in precedence order:\n\n" +
			"  1. $HULAK_MASTER_KEY env var\n" +
			"  2. ~/.config/hulak/identity.txt\n" +
			"  3. $HULAK_SSH_IDENTITY env path\n" +
			"  4. ~/.ssh/id_ed25519\n\n" +
			"The first row that decrypts is the default (marked with " + utils.Asterisk + "): it's\n" +
			"the identity hulak uses for read paths. NAME and ADDED come from\n" +
			"recipients.txt when the pubkey matches an entry there.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets identity list",
				Description: "Show every decrypting identity on this machine",
			},
		},
		Run: runListIdentity,
	}
}

// runListIdentity handles `hulak secrets identity list`.
func runListIdentity(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}
	if err := cli.RequireVaultProject(); err != nil {
		return err
	}

	storePath, err := vault.StorePath()
	if err != nil {
		return err
	}
	cipherText, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("no store.age found in this project")
		}
		return fmt.Errorf("failed to read store: %w", err)
	}

	hits := vault.SourcesThatDecrypt(cipherText)
	if len(hits) == 0 {
		return errors.New(
			"no configured identity can decrypt store.age. " +
				"Run 'hulak doctor' to inspect identity sources",
		)
	}

	// Look up name + added-date per pubkey from recipients.txt.
	nameByKey := loadRecipientNames()

	rows := make([][]string, 0, len(hits))
	for i, h := range hits {
		marker := " "
		if i == 0 {
			marker = utils.Asterisk
		}
		name, added := vault.ParseRecipientName(nameByKey[h.PublicKey])
		rows = append(rows, []string{marker, h.Path, h.PublicKey, name, added})
	}

	return utils.PrintTable(
		os.Stdout,
		utils.StdoutHeaders([]string{"", "PATH", "RECIPIENT", "NAME", "ADDED"}),
		rows,
		utils.DefaultTableMaxCellWidth,
	)
}

// loadRecipientNames builds a pubkey → "Name (added DATE)" lookup from
// recipients.txt. Returns an empty map on any read failure — list-identity
// still has useful output without the join.
func loadRecipientNames() map[string]string {
	recipPath, err := vault.RecipientsFilePath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(recipPath)
	if err != nil {
		return nil
	}
	entries, err := vault.ParseRecipientsFileContent(data)
	if err != nil {
		return nil
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		out[e.Key] = e.Name
	}
	return out
}
