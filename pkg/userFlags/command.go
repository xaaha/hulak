package userflags

import (
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"text/tabwriter"
)

// Command represents a CLI command with optional subcommands and flags
type Command struct {
	Name        string                    // primary name (e.g. "gql")
	Aliases     []string                  // alternative names (e.g. "graphql", "GraphQL")
	Short       string                    // one-line description for parent's help listing
	Long        string                    // detailed help shown with --help
	Flags       *flag.FlagSet             // scoped flags for this command
	Args        []ArgDef                  // positional arg descriptions (for help only)
	SubCommands []*Command                // nested subcommands
	Run         func(args []string) error // handler; nil means subcommand-only
}

// ArgDef describes a positional argument for help output
type ArgDef struct {
	Name     string
	Required bool
	Desc     string
}

// Execute dispatches args to the correct subcommand or runs this command
func (cmd *Command) Execute(args []string) error {
	// No args: show help if subcommand-only, or run with empty args
	if len(args) == 0 {
		if cmd.Run == nil {
			cmd.printHelp(os.Stdout)
			return nil
		}
		return cmd.Run(args)
	}

	// Check for help request as first arg
	if isHelpArg(args[0]) {
		cmd.printHelp(os.Stdout)
		return nil
	}

	// Try to match a subcommand
	if sub := cmd.findSub(args[0]); sub != nil {
		return sub.Execute(args[1:])
	}

	// No subcommand matched — parse flags and run
	if cmd.Flags != nil {
		// Register a help flag so -h/--help works through flag parsing
		var helpFlag bool
		if cmd.Flags.Lookup("help") == nil {
			cmd.Flags.BoolVar(&helpFlag, "help", false, "Show help for this command")
		}
		if cmd.Flags.Lookup("h") == nil {
			cmd.Flags.BoolVar(&helpFlag, "h", false, "Show help for this command")
		}

		if err := cmd.Flags.Parse(args); err != nil {
			return err
		}

		if helpFlag {
			cmd.printHelp(os.Stdout)
			return nil
		}

		args = cmd.Flags.Args()
	}

	if cmd.Run == nil {
		cmd.printHelp(os.Stdout)
		return nil
	}

	return cmd.Run(args)
}

// findSub returns the subcommand matching name by Name or Aliases, or nil
func (cmd *Command) findSub(name string) *Command {
	for _, sub := range cmd.SubCommands {
		if sub.Name == name || slices.Contains(sub.Aliases, name) {
			return sub
		}
	}
	return nil
}

// printHelp writes the command's help text to w
func (cmd *Command) printHelp(w io.Writer) {
	if cmd.Long != "" {
		fmt.Fprintln(w, cmd.Long) //nolint:gosec // CLI help text, not web output
		fmt.Fprintln(w)           //nolint:gosec // CLI help text, not web output
	}

	if len(cmd.SubCommands) > 0 {
		fmt.Fprintln(w, "Subcommands:")
		tw := tabwriter.NewWriter(w, 0, 0, 4, ' ', 0)
		for _, sub := range cmd.SubCommands {
			fmt.Fprintf(tw, "  %s\t%s\n", sub.Name, sub.Short) //nolint:gosec // CLI help text, not web output
		}
		tw.Flush()
		fmt.Fprintln(w)
	}

	if cmd.Flags != nil {
		fmt.Fprintln(w, "Flags:")
		cmd.Flags.SetOutput(w)
		cmd.Flags.PrintDefaults()
		fmt.Fprintln(w)
	}

	if len(cmd.Args) > 0 {
		fmt.Fprintln(w, "Arguments:")
		tw := tabwriter.NewWriter(w, 0, 0, 4, ' ', 0)
		for _, a := range cmd.Args {
			req := ""
			if a.Required {
				req = " (required)"
			}
			fmt.Fprintf(tw, "  <%s>\t%s%s\n", a.Name, a.Desc, req) //nolint:gosec // CLI help text, not web output
		}
		tw.Flush()
	}
}

// isHelpArg returns true if the argument is a help request
func isHelpArg(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}
