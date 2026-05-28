// Top-level command factories for `hulak secrets set/get/delete`.
//
// These build the *command structs (Name, Aliases, Flags, Examples, Run) for
// the three legacy top-level shortcuts. The Run handlers themselves live in
// env_key_handlers.go, shared with the nested `secrets keys set/get/delete`
// factories in env_keys_crud.go.
package userflags

import (
	"flag"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// newEnvSetCmd returns the command struct for `hulak secrets set`.
func newEnvSetCmd() *command {
	setFs := flag.NewFlagSet("env set", flag.ContinueOnError)
	setEnv := registerEnvFlag(setFs, "", "Environment to operate on")
	setStdin := setFs.Bool("stdin", false, "Read value from stdin")
	var setType string
	typeUsage := "Value type: " + strings.Join(validSetTypes[:], "|")
	setFs.StringVar(&setType, "type", "string", typeUsage)
	setFs.StringVar(&setType, "t", "string", typeUsage)

	return &command{
		Name:    "set",
		Aliases: []string{"add"},
		Short:   "Set a key-value pair",
		Long:    "Store a secret in the encrypted vault.\n\nIf VALUE is omitted, you'll be prompted to enter it (no echo, no shell history).\nUse --stdin to pipe the value from standard input (useful for scripts).\nUse --type to store numbers, booleans, or JSON instead of strings.",
		Flags:   setFs,
		Args: []argDef{
			{Name: "key", Required: true, Desc: "Secret key name"},
			{Name: "value", Desc: "Secret value (omit to be prompted, or use --stdin)"},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets set API_KEY sk-123",
				Description: "Pick an environment from the TUI, then set",
			},
			{
				Command:     "hulak secrets set DB_URL --env prod",
				Description: "Prompt for the value (no shell history)",
			},
			{
				Command:     "echo -n \"$TOKEN\" | hulak secrets set TOKEN --stdin",
				Description: "Read value from stdin (scripts/CI)",
			},
			{
				Command:     "hulak secrets set FEATURE_FLAG true --env staging",
				Description: "Set a value in a specific environment",
			},
			{
				Command:     "hulak secrets set userAge 3939 --type int",
				Description: "Store as an integer (preserved through to GraphQL/JSON bodies)",
			},
			{
				Command:     "hulak secrets set ENABLED true --type bool",
				Description: "Store as a boolean",
			},
			{
				Command:     "hulak secrets set config '{\"a\":1}' --type json",
				Description: "Store an arbitrary JSON value (object, array, number, etc.)",
			},
		},
		Run: func(args []string) error { return runEnvSet(args, *setEnv, *setStdin, setType) },
	}
}

// newEnvGetCmd returns the command struct for `hulak secrets get`.
func newEnvGetCmd() *command {
	getFs := flag.NewFlagSet("env get", flag.ContinueOnError)
	getEnv := registerEnvFlag(getFs, "", "Environment to operate on")

	return &command{
		Name:    "get",
		Aliases: nil,
		Short:   "Get a value by key",
		Long:    "Retrieve a secret from the encrypted vault and print it to stdout.\n\nOutput is raw — no formatting — suitable for $(...) substitution in scripts.\nExits non-zero if the key is missing.",
		Flags:   getFs,
		Args: []argDef{
			{Name: "key", Required: true, Desc: "Secret key to retrieve"},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets get API_KEY",
				Description: "Pick an environment from the TUI, then print API_KEY",
			},
			{
				Command:     "hulak secrets get DB_URL --env prod",
				Description: "Print DB_URL from the prod environment",
			},
			{
				Command:     "API_KEY=$(hulak secrets get API_KEY --env staging)",
				Description: "Capture a value into a shell variable",
			},
		},
		Run: func(args []string) error { return runEnvGet(args, *getEnv) },
	}
}

// newEnvDeleteCmd returns the command struct for `hulak secrets delete`.
func newEnvDeleteCmd() *command {
	deleteFs := flag.NewFlagSet("env delete", flag.ContinueOnError)
	deleteEnv := registerEnvFlag(deleteFs, "", "Environment to operate on")

	return &command{
		Name:    "delete",
		Aliases: []string{"rm"},
		Short:   "Delete a key",
		Long:    "Remove a secret from the encrypted vault.\n\nExits non-zero if the key doesn't exist.",
		Flags:   deleteFs,
		Args: []argDef{
			{Name: "key", Required: true, Desc: "Secret key to delete"},
		},
		Examples: []*utils.CommandHelp{
			{
				Command:     "hulak secrets delete OLD_KEY",
				Description: "Pick an environment from the TUI, then delete OLD_KEY",
			},
			{
				Command:     "hulak secrets rm STALE_TOKEN --env staging",
				Description: "Delete from a specific environment (alias)",
			},
		},
		Run: func(args []string) error { return runEnvDelete(args, *deleteEnv) },
	}
}
