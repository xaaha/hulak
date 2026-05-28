// Package secrets implements the `hulak secrets` command tree: environment
// CRUD, key-level CRUD (`secrets keys ...`), identity management (`secrets
// identity ...`), backup/restore, sync, migrate, and the env picker that
// fronts every command that takes --env. New() returns the assembled tree
// for registration by the top-level dispatch.
package secrets

import (
	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/utils"
)

// New builds the `hulak secrets` command tree. The `env` alias is kept for
// backward compatibility with pre-vault muscle memory.
func New() *cli.Command {
	envCmd := &cli.Command{
		Name:    "secrets",
		Aliases: []string{"env"},
		Short:   "Manage encrypted environment secrets",
		Long: "Manage environment secrets stored in the encrypted vault (.hulak/store.age).\n\n" +
			"Secrets are organized by environment (e.g. global, staging, prod).\n\n" +
			"Three concern-scoped groups live here:\n" +
			"  - this level: environment listing, edit, backup/restore, sync, migrate.\n" +
			"  - `secrets keys ...`     for key-level CRUD inside an environment.\n" +
			"  - `secrets identity ...` for age identities and recipient management.\n\n" +
			"When --env is omitted on a command that takes one, you'll be prompted\n" +
			"to pick an environment from a TUI list.\n\n" +
			"'env' is kept as an alias of `secrets` for backward compatibility.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets list",
				Description: "List environment names defined in the vault",
			},
			{
				Command:     "hulak secrets keys list --env prod",
				Description: "List keys in the prod environment (values masked)",
			},
			{
				Command:     "hulak secrets keys set API_KEY sk-123 --env prod",
				Description: "Set a key in the prod environment",
			},
			{
				Command:     "hulak secrets keys get API_KEY --env staging",
				Description: "Get a value from the staging environment",
			},
			{
				Command:     "hulak secrets keys delete OLD_KEY --env staging",
				Description: "Delete a key from the staging environment",
			},
		},
	}

	envCmd.SubCommands = []*cli.Command{
		newEnvCreateCmd(),
		newEnvDeleteCmd(),
		newEnvListCmd(),
		newEnvKeysCmd(),
		newEnvEditCmd(),
		newEnvIdentityCmd(),
		newEnvRenameCmd(),
		newEnvSyncCmd(),
		newEnvMigrateCmd(),
		newEnvBackupCmd(),
		newEnvRestoreCmd(),
	}

	return envCmd
}
