package initcmd

import (
	"flag"
	"fmt"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// New builds the `hulak init` command (default vault-mode setup). Includes
// the `init classic` subcommand for the plaintext env/ flow.
func New() *cli.Command {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	createEnvFlag := fs.Bool(
		"env",
		false,
		"Create specific environment files instead of the default setup",
	)
	sshFlag := fs.Bool(
		"ssh",
		false,
		"Use SSH ed25519 key (~/.ssh/id_ed25519) instead of generating an age keypair",
	)
	sshIdentityFlag := fs.String(
		"ssh-identity",
		"",
		"Path to SSH private key (implies --ssh; overrides the default path)",
	)

	return &cli.Command{
		Name:  "init",
		Short: "Initialize a hulak project",
		Long: "Set up a new hulak project in the current directory.\n\n" +
			"By default, creates an encrypted vault (.hulak/store.age) with an age keypair.\n" +
			"Use --ssh to bootstrap with your default SSH ed25519 key (~/.ssh/id_ed25519),\n" +
			"or --ssh-identity <path> for a custom key.\n\n" +
			"To scaffold an example request file, use 'hulak example <type>' after init.\n" +
			"Run 'hulak init classic' (aliases: plain, no-vault) to use the plaintext env/\n" +
			"layout instead. Use -env to scaffold specific environments.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak init", Description: "Default setup (encrypted vault + age keypair)"},
			{
				Command:     "hulak init --ssh",
				Description: "Use ~/.ssh/id_ed25519 instead of generating an age keypair",
			},
			{
				Command:     "hulak init --ssh-identity ~/.ssh/work_ed25519",
				Description: "Use a custom SSH key",
			},
			{
				Command:     "hulak init -env staging prod",
				Description: "Scaffold staging and prod environments alongside global",
			},
			{
				Command:     "hulak example api",
				Description: "Scaffold a runnable REST example after init",
			},
			{
				Command:     "hulak init classic",
				Description: "Use the plaintext env/ layout (aliases: plain, no-vault)",
			},
		},
		Flags: fs,
		Args: []cli.ArgDef{
			{Name: "envNames", Desc: "Environment names to create (used with -env)"},
		},
		SubCommands: []*cli.Command{newClassic()},
		Run: func(args []string) error {
			var envNames []string
			if *createEnvFlag {
				if len(args) == 0 {
					utils.PrintWarningStderr("No environment names provided after -env flag")
				} else {
					envNames = args
				}
			}

			sshPath := *sshIdentityFlag
			if sshPath == "" && *sshFlag {
				sshPath = vault.DefaultSSHIdentityPath()
				if sshPath == "" {
					return fmt.Errorf("could not determine home directory for default SSH key path")
				}
			}

			return InitVaultProject(envNames, sshPath)
		},
	}
}

// newClassic builds the `hulak init classic` subcommand for the plaintext
// env/ flow. Registered as a subcommand of `init`, not at top-level — the
// classic flow is a deliberate opt-out from the default vault setup.
func newClassic() *cli.Command {
	fs := flag.NewFlagSet("init classic", flag.ContinueOnError)
	createEnvFlag := fs.Bool(
		"env",
		false,
		"Create specific environment files instead of the default setup",
	)

	return &cli.Command{
		Name:    "classic",
		Aliases: []string{"plain", "no-vault"},
		Short:   "Initialize with the plaintext env/ layout",
		Long: "Initialize a hulak project using the plaintext env/ layout.\n\n" +
			"Creates an env/ directory with global.env and a .gitignore entry.\n" +
			"Use this when you don't want the encrypted vault — for example, in\n" +
			"throwaway scripts or when secrets are managed entirely outside hulak.\n\n" +
			"To scaffold an example request file, use 'hulak example <type>' after init.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak init classic",
				Description: "Plaintext setup (env/global.env + .gitignore)",
			},
			{Command: "hulak init plain", Description: "Same as classic (alias)"},
			{Command: "hulak init no-vault", Description: "Same as classic (alias)"},
			{
				Command:     "hulak init classic -env staging prod",
				Description: "Create staging.env and prod.env in the env/ directory",
			},
		},
		Flags: fs,
		Args: []cli.ArgDef{
			{Name: "envNames", Desc: "Environment names to create (used with -env)"},
		},
		Run: func(args []string) error {
			if !*createEnvFlag {
				return InitClassicProject()
			}
			if len(args) == 0 {
				utils.PrintWarningStderr("No environment names provided after -env flag")
				return nil
			}
			// Collect failures so the user sees every one, but return the
			// first error so the exit code reflects that something went wrong.
			var firstErr error
			for _, env := range args {
				if err := envparser.CreateDefaultEnvs(&env); err != nil {
					utils.PrintErrorStderr(err.Error())
					if firstErr == nil {
						firstErr = err
					}
				}
			}
			return firstErr
		},
	}
}
