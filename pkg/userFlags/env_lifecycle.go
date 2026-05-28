// Command factories and handlers for the environment-lifecycle commands:
//
//	secrets create   — create a new empty environment
//
// `delete` and `rename` join this file in subsequent chunks. Lifecycle is
// kept separate from key-level CRUD (env_key_handlers.go) because the unit
// of work is different — these commands operate on the env itself, not on
// keys within it.
package userflags

import (
	"errors"
	"flag"
	"fmt"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// newEnvCreateCmd returns the command struct for `hulak secrets create`.
func newEnvCreateCmd() *command {
	fs := flag.NewFlagSet("env create", flag.ContinueOnError)
	envName := registerEnvFlag(
		fs,
		"",
		"Name of the new environment to create (required)",
	)

	return &command{
		Name:  "create",
		Short: "Create a new empty environment",
		Long: "Create a new empty environment in the encrypted vault.\n\n" +
			"--env is required — unlike commands that target an existing environment,\n" +
			"there's no TUI picker fallback because we're creating a new one.\n" +
			"Fails if the environment already exists. Use `hulak secrets keys set ...`\n" +
			"to populate it afterwards, or `hulak secrets edit --env NAME` to edit\n" +
			"the JSON directly.",
		Flags: fs,
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets create --env staging",
				Description: "Create a new staging environment",
			},
		},
		Run: func(args []string) error { return runEnvCreate(args, *envName) },
	}
}

// runEnvCreate creates a new empty environment under the store lock. Returns
// a non-zero error if the name is missing, invalid, or already taken — at no
// point is an existing environment touched.
func runEnvCreate(args []string, envName string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none (use --env NAME)", len(args))
	}
	if envName == "" {
		return errors.New("--env is required: hulak secrets create --env NAME")
	}
	if err := requireVaultProject(); err != nil {
		return err
	}
	if err := utils.ValidateEnvName(envName); err != nil {
		return err
	}

	return vault.WithStoreLock(func() error {
		store, err := vault.ReadStore()
		if err != nil {
			return err
		}
		if !store.EnsureSection(envName) {
			return fmt.Errorf("environment %q already exists", envName)
		}
		if err := vault.WriteStoreToRecipients(store); err != nil {
			return err
		}
		utils.PrintSuccessStderr(fmt.Sprintf("Created environment %q", envName))
		return nil
	})
}
