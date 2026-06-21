// Contains the top-level command tree and factories for non-env commands
// (run, version, init, migrate, doctor, gql, help). Env subcommand factories
// live in their respective env_*.go files; newEnvCmd assembles them here.
package userflags

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/migration"
	"github.com/xaaha/hulak/pkg/runner"
	"github.com/xaaha/hulak/pkg/tui/gqlexplorer"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// registerEnvFlag adds both --env and --environment aliases to a FlagSet,
// pointing to the same underlying variable, and returns a pointer so
// Run handlers can read the parsed value.
func registerEnvFlag(fs *flag.FlagSet, defaultVal string, usage string) *string {
	var envVal string
	fs.StringVar(&envVal, "env", defaultVal, usage)
	fs.StringVar(&envVal, "environment", defaultVal, usage)
	return &envVal
}

// subCommands builds the full sub-command tree for hulak
func subCommands() *command {
	root := &command{
		Name:  "hulak",
		Long:  "hulak — a file-based API client for the terminal",
		Flags: flag.CommandLine,
		Examples: []*utils.CommandHelp{
			{Command: "hulak", Description: "Interactive mode: pick a file, then an environment"},
			{
				Command:     "hulak -env staging -fp getUser.yaml",
				Description: "Run a specific file with a specific environment",
			},
			{
				Command:     "hulak -env global -f getUser",
				Description: "Find and run all files named 'getUser'",
			},
			{
				Command:     "hulak -fp getUser.yaml -debug",
				Description: "Run in debug mode (full request/response details)",
			},
			{
				Command:     "hulak -env prod -dir path/to/dir",
				Description: "Run all files in a directory concurrently",
			},
			{
				Command:     "hulak -env prod -dirseq path/to/dir",
				Description: "Run all files in a directory sequentially",
			},
		},
	}

	root.SubCommands = []*command{
		newRunCmd(),
		newVersionCmd(),
		newInitCmd(),
		newExampleCmd(),
		newMigrateCmd(),
		newDoctorCmd(),
		newGQLCmd(),
		newEnvCmd(),
		newCompletionCmd(),
		newGenDocsCmd(),
		newHelpCmd(root),
	}

	return root
}

func newVersionCmd() *command {
	return &command{
		Name:  "version",
		Short: "Print hulak version",
		Long:  "Print the current hulak version.\n\nUseful for bug reports and verifying installs.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak version", Description: "Print the installed hulak version"},
		},
		Run: func(_ []string) error {
			getVersion()
			return nil
		},
	}
}

func newInitCmd() *command {
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

	return &command{
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
		Args: []argDef{
			{Name: "envNames", Desc: "Environment names to create (used with -env)"},
		},
		SubCommands: []*command{newInitClassicCmd()},
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

func newInitClassicCmd() *command {
	fs := flag.NewFlagSet("init classic", flag.ContinueOnError)
	createEnvFlag := fs.Bool(
		"env",
		false,
		"Create specific environment files instead of the default setup",
	)

	return &command{
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
		Args: []argDef{
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

func newMigrateCmd() *command {
	return &command{
		Name:  "migrate",
		Short: "Migrate Postman collections to hulak format",
		Long: "Convert Postman v2.1 environment and collection JSON exports into hulak .hk.yaml and .env files.\n\n" +
			"Only Postman collections and environments are supported at this time.\n" +
			"To migrate plaintext env/ files to the encrypted vault, use 'hulak secrets migrate' instead.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak migrate collection.json", Description: "Migrate a Postman collection"},
			{
				Command:     "hulak migrate env.json collection.json",
				Description: "Migrate environment and collection together",
			},
		},
		Args: []argDef{
			{Name: "files", Required: true, Desc: "Postman JSON export files", Kind: "file"},
		},
		Run: migration.CompleteMigration,
	}
}

func newDoctorCmd() *command {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fixFlag := fs.Bool("fix", false, "Auto-repair safe issues (chmod, .gitignore)")
	yesFlag := fs.Bool("yes", false, "Skip confirmation prompts (use with --fix)")
	jsonFlag := fs.Bool("json", false, "Output findings as JSON to stdout")

	return &command{
		Name:  "doctor",
		Short: "Check project health",
		Long: "Inspect your hulak project for common issues.\n\n" +
			"Vault backend: identity, store, recipients, and drift checks.\n" +
			"Classic backend: .gitignore, file permissions, git history.",
		Flags: fs,
		Examples: []*utils.CommandHelp{
			{Command: "hulak doctor", Description: "Run all health checks"},
			{Command: "hulak doctor --fix", Description: "Auto-repair safe issues"},
			{Command: "hulak doctor --fix --yes", Description: "Auto-repair without prompts"},
			{Command: "hulak doctor --json", Description: "Output findings as JSON"},
		},
		Run: func(_ []string) error {
			os.Exit(runDoctor(doctorOpts{
				fix:     *fixFlag,
				yes:     *yesFlag,
				jsonOut: *jsonFlag,
			}))
			return nil // unreachable
		},
	}
}

func newGQLCmd() *command {
	fs := flag.NewFlagSet("gql", flag.ContinueOnError)
	envFlagVal := registerEnvFlag(fs, "", "Environment to use (skips interactive selector)")

	gqlCmd := &command{
		Name:    "gql",
		Aliases: []string{"graphql"},
		Short:   "Open the GraphQL explorer",
		Long:    "Launch an interactive TUI to browse and run GraphQL operations\ndefined in your .yml/.yaml files.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak gql .",
				Description: "Explore all GraphQL files in the current directory",
			},
			{
				Command:     "hulak gql path/to/schema.yml",
				Description: "Explore a single GraphQL source file",
			},
			{
				Command:     "hulak gql -env staging .",
				Description: "Use the staging environment (skip env picker)",
			},
		},
		Flags: fs,
		Args: []argDef{
			{
				Name:     "path",
				Required: true,
				Desc:     "File or directory containing GraphQL definitions",
				Kind:     "yaml",
			},
		},
	}

	gqlCmd.Run = func(args []string) error {
		if len(args) == 0 {
			gqlCmd.printHelp()
			return nil
		}
		data, refreshFn, warnings, err := loadGraphQLOperations(args[0], *envFlagVal)
		if err != nil {
			return err
		}
		if data.Operations == nil {
			return nil
		}
		if err := gqlexplorer.RunExplorerWithRefresh(&data, refreshFn, warnings); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	}

	return gqlCmd
}

func newRunCmd() *command {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	envFlagVal := registerEnvFlag(fs, "", "Environment to use")
	var sequential bool
	var debug bool
	var quiet bool
	var timeout time.Duration
	var sshIdentity string
	fs.BoolVar(&sequential, "sequential", false, "Run directory files sequentially")
	fs.BoolVar(&sequential, "seq", false, "Run directory files sequentially")
	fs.BoolVar(&debug, "debug", false, "Enable debug mode")
	fs.BoolVar(&quiet, "quiet", false, "Suppress the end-of-run summary table")
	fs.BoolVar(&quiet, "q", false, "Suppress the end-of-run summary table")
	dryRun := registerDryRunFlag(fs)
	show := registerShowFlag(
		fs,
		"Reveal sensitive headers (Authorization, Cookie, etc.) in --dry-run output",
	)
	fs.DurationVar(
		&timeout,
		"timeout",
		0,
		"Per-request timeout, e.g. 5m or 90s (default 60s)",
	)
	fs.StringVar(&sshIdentity, "ssh-identity", "", "Path to SSH private key for vault decryption")

	runCmd := &command{
		Name:  "run",
		Short: "Run API request file(s) or directory",
		Long: "Execute one or more API request files.\n\n" +
			"Pass a file path to run a single request, or a directory to run all files in it.\n" +
			"Directories run concurrently by default; use --sequential for ordered execution.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak run path/to/file.yaml", Description: "Run a single request file"},
			{
				Command:     "hulak run path/to/file.yaml --env staging",
				Description: "Run with a specific environment",
			},
			{
				Command:     "hulak run path/to/dir/",
				Description: "Run all files in a directory concurrently",
			},
			{
				Command:     "hulak run path/to/dir/ --sequential",
				Description: "Run directory files sequentially",
			},
			{
				Command:     "hulak run path/to/file.yaml --ssh-identity ~/.ssh/work_ed25519",
				Description: "Use a specific SSH key for vault decryption",
			},
			{
				Command:     "hulak run path/to/file.yaml --dry-run",
				Description: "Print the built request and exit (sensitive headers masked)",
			},
			{
				Command:     "hulak run path/to/file.yaml --dry-run --show",
				Description: "Same as --dry-run but reveal sensitive headers",
			},
		},
		Flags: fs,
		Args: []argDef{
			{Name: "path", Required: true, Desc: "File or directory to run", Kind: "yaml"},
		},
	}

	runCmd.Run = func(args []string) error {
		if len(args) == 0 {
			runCmd.printHelp()
			return nil
		}

		f, err := parseRunArgs(runCmdArgs{
			Env:         *envFlagVal,
			Sequential:  sequential,
			Debug:       debug,
			Quiet:       quiet,
			DryRun:      *dryRun,
			Show:        *show,
			Timeout:     timeout,
			SSHIdentity: sshIdentity,
			Args:        args,
		})
		if err != nil {
			return err
		}

		// Propagate runner errors so the top-level exit code reflects failures.
		// Per-file detail has already been printed; an empty error message is
		// the runner's signal of "exit non-zero, no extra output needed".
		return runner.Execute(f)
	}

	return runCmd
}

// runCmdArgs bundles the values parsed from the `run` subcommand flagset
// plus the positional args. Passing them as one struct keeps parseRunArgs
// from growing a parameter list every time a flag is added.
type runCmdArgs struct {
	Env         string
	Sequential  bool
	Debug       bool
	Quiet       bool
	DryRun      bool
	Show        bool
	Timeout     time.Duration
	SSHIdentity string
	Args        []string
}

// parseRunArgs builds a runner.Flags from the path and parsed flag values.
// The path routes to FilePath (file), Dir (concurrent), or Dirseq (sequential).
func parseRunArgs(a runCmdArgs) (*runner.Flags, error) {
	path := a.Args[0]

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access %q: %w", path, err)
	}

	f := &runner.Flags{
		Debug:       a.Debug,
		Quiet:       a.Quiet,
		DryRun:      a.DryRun,
		Show:        a.Show,
		Timeout:     a.Timeout,
		SSHIdentity: a.SSHIdentity,
	}

	if a.Env != "" {
		f.Env = a.Env
		f.EnvSet = true
	}

	if info.IsDir() {
		if a.Sequential {
			f.Dirseq = path
		} else {
			f.Dir = path
		}
	} else {
		f.FilePath = path
	}

	return f, nil
}

func newEnvCmd() *command {
	envCmd := &command{
		Name:    "secrets",
		Aliases: []string{"env"},
		Short:   "Manage encrypted environment secrets",
		Long: "Manage environment secrets stored in the encrypted vault (.hulak/store.age).\n\n" +
			"Secrets are organized by environment (e.g. global, staging, prod).\n" +
			"When --env is omitted, you'll be prompted to pick an environment interactively.\n\n" +
			"'env' is retained as an alias for backward compatibility with pre-0.3 docs.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets list",
				Description: "List environment names defined in the vault",
			},
			{
				Command:     "hulak secrets set API_KEY sk-123 --env prod",
				Description: "Set a secret in the prod environment",
			},
			{
				Command:     "hulak secrets get API_KEY --env staging",
				Description: "Get a secret from the staging environment",
			},
			{
				Command:     "hulak secrets keys --env prod",
				Description: "List keys in the prod environment (values masked)",
			},
			{
				Command:     "hulak secrets delete OLD_KEY",
				Description: "Delete a key from the default environment",
			},
		},
	}

	envCmd.SubCommands = []*command{
		newEnvSetCmd(),
		newEnvGetCmd(),
		newEnvListCmd(),
		newEnvKeysCmd(),
		newEnvDeleteCmd(),
		newEnvEditCmd(),
		newEnvImportKeyCmd(),
		newEnvExportKeyCmd(),
		newEnvGenIdentityCmd(),
		newEnvListIdentityCmd(),
		newEnvAddRecipientCmd(),
		newEnvRemoveRecipientCmd(),
		newEnvListRecipientsCmd(),
		newEnvSyncCmd(),
		newEnvRotateKeyCmd(),
		newEnvMigrateCmd(),
		newEnvBackupCmd(),
		newEnvRestoreCmd(),
	}

	return envCmd
}

func newHelpCmd(root *command) *command {
	return &command{
		Name:  "help",
		Short: "Show help for hulak",
		Long:  "Print the top-level hulak help.\n\nFor help on a specific command, use `hulak <command> --help` instead.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak help", Description: "Show top-level help"},
			{Command: "hulak secrets --help", Description: "Show help for a specific command"},
			{Command: "hulak secrets keys --help", Description: "Show help for a nested subcommand"},
		},
		Run: func(_ []string) error {
			root.printHelp()
			return nil
		},
	}
}
