package doctor

import (
	"flag"
	"os"

	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/utils"
)

// New builds the `hulak doctor` command. Registered at the top-level
// dispatch tree alongside other leaf commands.
func New() *cli.Command {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fixFlag := fs.Bool("fix", false, "Auto-repair safe issues (chmod, .gitignore)")
	yesFlag := fs.Bool("yes", false, "Skip confirmation prompts (use with --fix)")
	jsonFlag := fs.Bool("json", false, "Output findings as JSON to stdout")

	return &cli.Command{
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
			// doctor signals severity via exit code (0 ok, 1 warn, 2 error)
			// rather than a returned error, so callers like CI can branch on
			// it. Exit directly to preserve that contract — returning the
			// code as an error would collapse 1 and 2 into the dispatcher's
			// generic non-zero exit.
			os.Exit(runDoctor(doctorOpts{
				fix:     *fixFlag,
				yes:     *yesFlag,
				jsonOut: *jsonFlag,
			}))
			return nil // unreachable
		},
	}
}
