// Command factories and handlers for the environment-lifecycle commands:
//
//	secrets create   — create a new empty environment
//	secrets delete   — delete an environment (with confirm prompt)
//
// `rename` joins this file in a subsequent chunk. Lifecycle is kept separate
// from key-level CRUD (env_key_handlers.go) because the unit of work is
// different — these commands operate on the env itself, not on keys within it.
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

// newEnvDeleteCmd returns the command struct for `hulak secrets delete`.
func newEnvDeleteCmd() *command {
	fs := flag.NewFlagSet("env delete", flag.ContinueOnError)
	envName := registerEnvFlag(
		fs,
		"",
		"Environment to delete (omit to pick interactively)",
	)
	yes := registerYesFlag(fs, "Skip the destructive confirm prompt")

	return &command{
		Name:    "delete",
		Aliases: []string{"rm"},
		Short:   "Delete an environment",
		Long: "Delete an environment from the encrypted vault.\n\n" +
			"All keys in the environment are destroyed along with it. If the\n" +
			"environment contains keys you'll be asked to confirm; an empty\n" +
			"environment deletes without prompting because there's nothing\n" +
			"to lose. Pass --yes (or -y) to skip the prompt in scripts and CI.",
		Flags: fs,
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets delete --env old_staging",
				Description: "Delete an environment (prompts if non-empty)",
			},
			{
				Command:     "hulak secrets delete --env temp --yes",
				Description: "Delete without the confirm prompt",
			},
			{
				Command:     "hulak secrets rm --env temp",
				Description: "Same as delete (alias)",
			},
		},
		Run: func(args []string) error { return runDeleteEnv(args, *envName, *yes) },
	}
}

// runDeleteEnv removes an environment from the encrypted vault.
//
// Confirmation rules (see confirmDestroy):
//   - empty env (0 keys): no prompt, just delete.
//   - non-empty env, no --yes: prompt with the key count.
//   - --yes (force): skip prompt at any count.
//
// On decline, no write happens; the store stays untouched.
func runDeleteEnv(args []string, envName string, force bool) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments: got %d, expected none", len(args))
	}

	envName, cancelled, err := resolveAndValidateEnv(envName)
	if err != nil {
		return err
	}
	if cancelled {
		return nil
	}

	return vault.WithStoreLock(func() error {
		store, err := vault.ReadStore()
		if err != nil {
			return err
		}
		env, err := requireEnvExists(store, envName)
		if err != nil {
			return err
		}

		count := len(env)
		desc := fmt.Sprintf("keys in %q", envName)
		if count == 1 {
			desc = fmt.Sprintf("key in %q", envName)
		}

		ok, err := confirmDestroy(desc, count, force)
		if err != nil {
			return err
		}
		if !ok {
			utils.PrintInfoStderr("Aborted")
			return nil
		}

		store.DeleteEnv(envName)
		if err := vault.WriteStoreToRecipients(store); err != nil {
			return err
		}
		utils.PrintSuccessStderr(fmt.Sprintf("Deleted environment %q", envName))
		return nil
	})
}
