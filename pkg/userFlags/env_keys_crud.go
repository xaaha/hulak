// Command factories for the four leaf subcommands under `hulak secrets keys`:
//
//	secrets keys list       — list keys within an environment
//	secrets keys set        — set a key-value pair
//	secrets keys get        — get a value by key
//	secrets keys delete     — delete a key
//
// Each factory builds a fresh FlagSet (FlagSets are not reusable across
// commands). The Run handlers themselves live in env_key_handlers.go.
package userflags

import (
	"flag"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

func keysListCmd() *command {
	fs := flag.NewFlagSet("keys list", flag.ContinueOnError)
	envName := registerEnvFlag(fs, "", "Environment to operate on")
	show := registerShowFlag(fs, "Reveal values instead of masking them")
	search := fs.String(
		"search",
		"",
		"Filter keys by case-insensitive substring or glob pattern",
	)

	return &command{
		Name:    "list",
		Aliases: []string{"ls"},
		Short:   "List keys in an environment",
		Long:    "Show secret keys within an environment.\n\nValues are masked by default (••••) so the output is safe to share in screen recordings\nand meetings. Use --show to reveal them.\nUse --search to filter by case-insensitive substring or glob pattern (e.g. \"API*\", \"DB_?\").",
		Flags:   fs,
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets keys list --env prod",
				Description: "List keys in prod with values masked",
			},
			{
				Command:     "hulak secrets keys list --env prod --show",
				Description: "Reveal actual values",
			},
			{
				Command:     "hulak secrets keys list --env prod --search \"API*\"",
				Description: "Filter keys by glob pattern",
			},
		},
		Run: func(args []string) error {
			return runEnvKeys(args, *envName, *search, *show)
		},
	}
}

func keysSetCmd() *command {
	fs := flag.NewFlagSet("keys set", flag.ContinueOnError)
	envName := registerEnvFlag(fs, "", "Environment to operate on")
	useStdin := fs.Bool("stdin", false, "Read value from stdin")
	var typeName string
	typeUsage := "Value type: " + strings.Join(validSetTypes[:], "|")
	fs.StringVar(&typeName, "type", "string", typeUsage)
	fs.StringVar(&typeName, "t", "string", typeUsage)

	return &command{
		Name:    "set",
		Aliases: []string{"add"},
		Short:   "Set a key-value pair",
		Long:    "Store a secret in the encrypted vault.\n\nIf VALUE is omitted, you'll be prompted to enter it (no echo, no shell history).\nUse --stdin to pipe the value from standard input.\nUse --type to store numbers, booleans, or JSON instead of strings.",
		Flags:   fs,
		Args: []argDef{
			{Name: "key", Required: true, Desc: "Secret key name"},
			{Name: "value", Desc: "Secret value (omit to be prompted, or use --stdin)"},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets keys set API_KEY sk-123 --env prod",
				Description: "Set a secret in the prod environment",
			},
			{
				Command:     "echo -n \"$TOKEN\" | hulak secrets keys set TOKEN --stdin --env prod",
				Description: "Read value from stdin (scripts/CI)",
			},
			{
				Command:     "hulak secrets keys set userAge 3939 --type int --env staging",
				Description: "Store as an integer (preserved through JSON bodies)",
			},
		},
		Run: func(args []string) error {
			return runEnvSet(args, *envName, *useStdin, typeName)
		},
	}
}

func keysGetCmd() *command {
	fs := flag.NewFlagSet("keys get", flag.ContinueOnError)
	envName := registerEnvFlag(fs, "", "Environment to operate on")

	return &command{
		Name:  "get",
		Short: "Get a value by key",
		Long:  "Retrieve a secret from the encrypted vault and print it to stdout.\n\nOutput is raw — no formatting — suitable for $(...) substitution in scripts.\nExits non-zero if the key is missing.",
		Flags: fs,
		Args: []argDef{
			{Name: "key", Required: true, Desc: "Secret key to retrieve"},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets keys get DB_URL --env prod",
				Description: "Print DB_URL from the prod environment",
			},
			{
				Command:     "TOKEN=$(hulak secrets keys get TOKEN --env staging)",
				Description: "Capture a value into a shell variable",
			},
		},
		Run: func(args []string) error {
			return runEnvGet(args, *envName)
		},
	}
}

func keysDeleteCmd() *command {
	fs := flag.NewFlagSet("keys delete", flag.ContinueOnError)
	envName := registerEnvFlag(fs, "", "Environment to operate on")

	return &command{
		Name:    "delete",
		Aliases: []string{"rm"},
		Short:   "Delete a key from an environment",
		Long:    "Remove a secret from the encrypted vault.\n\nExits non-zero if the key doesn't exist.",
		Flags:   fs,
		Args: []argDef{
			{Name: "key", Required: true, Desc: "Secret key to delete"},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets keys delete OLD_KEY --env prod",
				Description: "Delete OLD_KEY from prod",
			},
		},
		Run: func(args []string) error {
			return runEnvDelete(args, *envName)
		},
	}
}
