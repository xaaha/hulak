package userflags

import (
	"flag"
	"fmt"
	"os"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/migration"
	"github.com/xaaha/hulak/pkg/runner"
	"github.com/xaaha/hulak/pkg/tui/gqlexplorer"
	"github.com/xaaha/hulak/pkg/utils"
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
		newMigrateCmd(),
		newDoctorCmd(),
		newGQLCmd(),
		newEnvCmd(),
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

	return &command{
		Name:  "init",
		Short: "Initialize a hulak project",
		Long: fmt.Sprintf(
			"Set up a new hulak project in the current directory.\n\n"+
				"Creates an env/ directory with global.env, a .gitignore entry,\n"+
				"and an example '%s' file. Use -env to create specific environments.",
			utils.APIOptions,
		),
		Examples: []*utils.CommandHelp{
			{Command: "hulak init", Description: "Default setup (global.env + example file)"},
			{
				Command:     "hulak init -env staging prod",
				Description: "Create staging.env and prod.env",
			},
		},
		Flags: fs,
		Args: []argDef{
			{Name: "envNames", Desc: "Environment names to create (used with -env)"},
		},
		Run: func(args []string) error {
			if *createEnvFlag {
				if len(args) == 0 {
					utils.PrintWarning("No environment names provided after -env flag")
					return nil
				}
				for _, env := range args {
					if err := envparser.CreateDefaultEnvs(&env); err != nil {
						utils.PrintRed(err.Error())
					}
				}
				return nil
			}
			return InitDefaultProject()
		},
	}
}

func newMigrateCmd() *command {
	return &command{
		Name:  "migrate",
		Short: "Migrate Postman collections to hulak format",
		Long:  "Convert Postman v2.1 environment and collection JSON exports into hulak .hk.yaml and .env files.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak migrate collection.json", Description: "Migrate a Postman collection"},
			{
				Command:     "hulak migrate env.json collection.json",
				Description: "Migrate environment and collection together",
			},
		},
		Args: []argDef{
			{Name: "files", Required: true, Desc: "Postman JSON export files"},
		},
		Run: migration.CompleteMigration,
	}
}

func newDoctorCmd() *command {
	return &command{
		Name:  "doctor",
		Short: "Check project health",
		Long:  "Inspect your hulak project for common issues: missing .gitignore entries,\nloose file permissions on env files, and secrets leaked into git history.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak doctor", Description: "Run all health checks"},
		},
		Run: func(_ []string) error {
			runDoctor()
			return nil
		},
	}
}

func newGQLCmd() *command {
	fs := flag.NewFlagSet("gql", flag.ContinueOnError)
	envFlagVal := registerEnvFlag(fs, "", "Environment to use (skips interactive selector)")

	gqlCmd := &command{
		Name:    "gql",
		Aliases: []string{"graphql", "GraphQL"},
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
			},
		},
	}

	gqlCmd.Run = func(args []string) error {
		if len(args) == 0 {
			gqlCmd.printHelp()
			return nil
		}
		data, refreshFn, warnings := loadGraphQLOperations(args[0], *envFlagVal)
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
	fs.BoolVar(&sequential, "sequential", false, "Run directory files sequentially")
	fs.BoolVar(&sequential, "seq", false, "Run directory files sequentially")
	fs.BoolVar(&debug, "debug", false, "Enable debug mode")

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
		},
		Flags: fs,
		Args: []argDef{
			{Name: "path", Required: true, Desc: "File or directory to run"},
		},
	}

	runCmd.Run = func(args []string) error {
		if len(args) == 0 {
			runCmd.printHelp()
			return nil
		}

		f, err := parseRunArgs(*envFlagVal, sequential, debug, args)
		if err != nil {
			return err
		}

		runner.Execute(f)
		return nil
	}

	return runCmd
}

// parseRunArgs builds a runner.Flags from the path and parsed flag values.
// The path routes to FilePath (file), Dir (concurrent), or Dirseq (sequential).
func parseRunArgs(envFlagVal string, sequential, debug bool, args []string) (*runner.Flags, error) {
	path := args[0]

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access %q: %w", path, err)
	}

	f := &runner.Flags{Debug: debug}

	if envFlagVal != "" {
		f.Env = envFlagVal
		f.EnvSet = true
	} else {
		f.Env = utils.DefaultEnvVal
	}

	if info.IsDir() {
		if sequential {
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
		Name:  "env",
		Short: "Manage encrypted environment secrets",
		Long: "Manage environment secrets stored in the encrypted vault (.hulak/store.age).\n\n" +
			"Secrets are organized by environment (e.g. global, staging, prod).\n" +
			"The default environment is \"global\" unless --env is specified.",
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak env list",
				Description: "List environment names defined in the vault",
			},
			{
				Command:     "hulak env set API_KEY sk-123 --env prod",
				Description: "Set a secret in the prod environment",
			},
			{
				Command:     "hulak env get API_KEY --env staging",
				Description: "Get a secret from the staging environment",
			},
			{
				Command:     "hulak env keys --env prod",
				Description: "List keys in the prod environment (values masked)",
			},
			{
				Command:     "hulak env delete OLD_KEY",
				Description: "Delete a key from the default environment",
			},
		},
	}

	// set — store a key-value pair
	setFs := flag.NewFlagSet("env set", flag.ContinueOnError)
	setEnv := registerEnvFlag(setFs, utils.DefaultEnvVal, "Environment to operate on")
	setStdin := setFs.Bool("stdin", false, "Read value from stdin")

	// get — retrieve a value by key
	getFs := flag.NewFlagSet("env get", flag.ContinueOnError)
	getEnv := registerEnvFlag(getFs, utils.DefaultEnvVal, "Environment to operate on")

	// list — show environment names
	listFs := flag.NewFlagSet("env list", flag.ContinueOnError)

	// keys — list keys within an environment
	keysFs := flag.NewFlagSet("env keys", flag.ContinueOnError)
	keysEnv := registerEnvFlag(keysFs, utils.DefaultEnvVal, "Environment to operate on")
	keysShow := keysFs.Bool("show", false, "Reveal values instead of masking them")
	keysSearch := keysFs.String(
		"search",
		"",
		"Filter keys by case-insensitive substring or glob pattern",
	)

	// delete — remove a key
	deleteFs := flag.NewFlagSet("env delete", flag.ContinueOnError)
	deleteEnv := registerEnvFlag(deleteFs, utils.DefaultEnvVal, "Environment to operate on")

	// edit — interactive editor; empty default → TUI picker, like `hulak run`
	editFs := flag.NewFlagSet("env edit", flag.ContinueOnError)
	editEnv := registerEnvFlag(editFs, "", "Environment to edit (omit to pick interactively)")

	// import-key — import an age identity
	importKeyFs := flag.NewFlagSet("env import-key", flag.ContinueOnError)
	importKeyFs.Bool("stdin", false, "Read key from stdin")

	// export-key — export the age identity
	exportKeyFs := flag.NewFlagSet("env export-key", flag.ContinueOnError)
	exportKeyFs.Bool("armor", false, "Output in ASCII-armored format")

	notImplemented := func(name string) func([]string) error {
		return func(_ []string) error {
			fmt.Printf("hulak env %s is not yet implemented\n", name)
			return nil
		}
	}

	envCmd.SubCommands = []*command{
		{
			Name:    "set",
			Aliases: []string{"add"},
			Short:   "Set a key-value pair",
			Long:    "Store a secret in the encrypted vault.\n\nIf VALUE is omitted, you'll be prompted to enter it (no echo, no shell history).\nUse --stdin to pipe the value from standard input (useful for scripts).",
			Flags:   setFs,
			Args: []argDef{
				{Name: "key", Required: true, Desc: "Secret key name"},
				{Name: "value", Desc: "Secret value (omit to be prompted, or use --stdin)"},
			},
			Examples: []*utils.CommandHelp{
				{
					Command:     "hulak env set API_KEY sk-123",
					Description: "Set a value in the default (global) environment",
				},
				{
					Command:     "hulak env set DB_URL --env prod",
					Description: "Prompt for the value (no shell history)",
				},
				{
					Command:     "echo -n \"$TOKEN\" | hulak env set TOKEN --stdin",
					Description: "Read value from stdin (scripts/CI)",
				},
				{
					Command:     "hulak env set FEATURE_FLAG true --env staging",
					Description: "Set a value in a specific environment",
				},
			},
			Run: func(args []string) error { return runEnvSet(args, *setEnv, *setStdin) },
		},
		{
			Name:    "get",
			Aliases: []string{"g", "show", "view"},
			Short:   "Get a value by key",
			Long:    "Retrieve a secret from the encrypted vault and print it to stdout.\n\nOutput is raw — no formatting — suitable for $(...) substitution in scripts.\nExits non-zero if the key is missing.",
			Flags:   getFs,
			Args: []argDef{
				{Name: "key", Required: true, Desc: "Secret key to retrieve"},
			},
			Examples: []*utils.CommandHelp{
				{
					Command:     "hulak env get API_KEY",
					Description: "Print API_KEY from the default environment",
				},
				{
					Command:     "hulak env get DB_URL --env prod",
					Description: "Print DB_URL from the prod environment",
				},
				{
					Command:     "API_KEY=$(hulak env get API_KEY --env staging)",
					Description: "Capture a value into a shell variable",
				},
			},
			Run: func(args []string) error { return runEnvGet(args, *getEnv) },
		},
		{
			Name:    "list",
			Aliases: []string{"ls", "l"},
			Short:   "List environment names",
			Long:    "Show all environment names defined in the encrypted vault.\n\nThis lists the environments themselves (e.g. global, staging, prod).\nUse `hulak env keys --env <name>` to list keys within an environment.",
			Flags:   listFs,
			Examples: []*utils.CommandHelp{
				{Command: "hulak env list", Description: "List all environment names"},
				{Command: "hulak env ls", Description: "Same as list (alias)"},
			},
			Run: runEnvList,
		},
		{
			Name:    "keys",
			Aliases: []string{"key"},
			Short:   "List keys in an environment",
			Long:    "Show secret keys within an environment.\n\nValues are masked by default (••••) so the output is safe to share in screen recordings\nand meetings. Use --show to reveal them.\nUse --search to filter by case-insensitive substring or glob pattern (e.g. \"API*\", \"DB_?\").",
			Flags:   keysFs,
			Examples: []*utils.CommandHelp{
				{
					Command:     "hulak env keys --env prod",
					Description: "List keys in prod with values masked",
				},
				{Command: "hulak env keys --env prod --show", Description: "Reveal actual values"},
				{
					Command:     "hulak env keys --env prod --search \"API*\"",
					Description: "Filter keys by glob pattern",
				},
				{
					Command:     "hulak env keys --env staging --search api",
					Description: "Filter by case-insensitive substring",
				},
			},
			Run: func(args []string) error {
				return runEnvKeys(args, *keysEnv, *keysSearch, *keysShow)
			},
		},
		{
			Name:    "delete",
			Aliases: []string{"rm", "remove", "del"},
			Short:   "Delete a key",
			Long:    "Remove a secret from the encrypted vault.\n\nExits non-zero if the key doesn't exist.",
			Flags:   deleteFs,
			Args: []argDef{
				{Name: "key", Required: true, Desc: "Secret key to delete"},
			},
			Examples: []*utils.CommandHelp{
				{
					Command:     "hulak env delete OLD_KEY",
					Description: "Delete OLD_KEY from the default environment",
				},
				{
					Command:     "hulak env rm STALE_TOKEN --env staging",
					Description: "Delete from a specific environment (alias)",
				},
			},
			Run: func(args []string) error { return runEnvDelete(args, *deleteEnv) },
		},
		{
			Name:  "edit",
			Short: "Edit secrets interactively",
			Long:  "Open the decrypted environment in $EDITOR (falls back to vi).\n\nWhen --env is omitted you'll be prompted to pick an environment from a TUI list,\nthe same flow as `hulak run`. To create a brand-new environment, pass --env\nexplicitly with the new name.\n\nThe decrypted JSON is written to a temp file with 0600 permissions inside .hulak/.\nOn editor exit the JSON is validated, merged back into the store, and re-encrypted\natomically. If the editor exits non-zero or the file is unchanged, no write occurs.",
			Flags: editFs,
			Examples: []*utils.CommandHelp{
				{
					Command:     "hulak env edit",
					Description: "Pick an environment from the TUI, then edit",
				},
				{
					Command:     "hulak env edit --env prod",
					Description: "Edit prod directly (skip the picker)",
				},
				{
					Command:     "hulak env edit --env new_one",
					Description: "Create a brand-new environment by name",
				},
				{
					Command:     "EDITOR=nvim hulak env edit --env staging",
					Description: "Use a specific editor",
				},
				{
					Command:     "EDITOR=\"zed --wait\" hulak env edit --env staging",
					Description: "GUI editors need a wait flag so hulak waits until you save (zed --wait, code -w)",
				},
			},
			Run: func(args []string) error { return runEnvEdit(args, *editEnv) },
		},
		{
			Name:  "import-key",
			Short: "Import an age identity file",
			Long:  "Import an age private key from a file or stdin into the hulak config directory.\n\nValidates the key format before storing. Writes to ~/.config/hulak/identity.txt\n(or the platform-specific config dir).",
			Flags: importKeyFs,
			Args: []argDef{
				{Name: "path", Desc: "Path to the identity file (omit to read from stdin)"},
			},
			Examples: []*utils.CommandHelp{
				{
					Command:     "hulak env import-key /path/to/backup.txt",
					Description: "Import from a backup file",
				},
				{
					Command:     "echo \"AGE-SECRET-KEY-1QF...\" | hulak env import-key --stdin",
					Description: "Import from stdin (scripts)",
				},
			},
			Run: notImplemented("import-key"),
		},
		{
			Name:  "export-key",
			Short: "Export the age identity file",
			Long:  "Print the age private key to stdout for backup or transfer to another machine.\n\nUse --armor for ASCII-armored output suitable for copy-paste.",
			Flags: exportKeyFs,
			Examples: []*utils.CommandHelp{
				{
					Command:     "hulak env export-key",
					Description: "Print the private key (with security warning)",
				},
				{
					Command:     "hulak env export-key > ~/backup.txt",
					Description: "Save to a backup file",
				},
				{
					Command:     "hulak env export-key --armor",
					Description: "ASCII-armored output for copy-paste",
				},
			},
			Run: notImplemented("export-key"),
		},
		{
			Name:  "add-recipient",
			Short: "Add a recipient for shared vault access",
			Long:  "Add an age public key as a recipient so another user can decrypt the vault.\n\nThe vault is re-encrypted to all current recipients plus the new one.",
			Args:  []argDef{{Name: "public-key", Required: true, Desc: "Age public key to add"}},
			Examples: []*utils.CommandHelp{
				{
					Command:     "hulak env add-recipient age1ql3z...",
					Description: "Add a teammate's public key",
				},
			},
			Run: notImplemented("add-recipient"),
		},
		{
			Name:  "remove-recipient",
			Short: "Remove a recipient",
			Long:  "Remove an age public key from the recipient list and re-encrypt the vault.\n\nNote: removed users can still decrypt copies of the vault from before this point.\nIf revocation matters, also rotate the underlying secrets.",
			Args:  []argDef{{Name: "public-key", Required: true, Desc: "Age public key to remove"}},
			Examples: []*utils.CommandHelp{
				{
					Command:     "hulak env remove-recipient age1ql3z...",
					Description: "Remove a teammate's public key",
				},
			},
			Run: notImplemented("remove-recipient"),
		},
		{
			Name:  "list-recipients",
			Short: "List all recipients",
			Long:  "Show all age public keys that can decrypt the vault.",
			Examples: []*utils.CommandHelp{
				{
					Command:     "hulak env list-recipients",
					Description: "Show all recipients with names and key prefixes",
				},
			},
			Run: notImplemented("list-recipients"),
		},
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
			{Command: "hulak env --help", Description: "Show help for a specific command"},
			{Command: "hulak env keys --help", Description: "Show help for a nested subcommand"},
		},
		Run: func(_ []string) error {
			root.printHelp()
			return nil
		},
	}
}
