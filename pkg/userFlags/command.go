package userflags

import (
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// flagAliases maps long-form flag names to their short form.
// The long form is hidden from help output; only the short form is shown
// with both names on one line (e.g. "-fp, --file-path").
var flagAliases = map[string]string{
	"file-path":   "fp",
	"file":        "f",
	"environment": "env",
	"version":     "v",
	"help":        "h",
}

// hiddenFlags are omitted from help output entirely (utility flags
// that don't need to clutter command-specific help)
var hiddenFlags = map[string]bool{
	"h": true, "v": true,
}

// command represents a CLI command with optional subcommands and flags
type command struct {
	Name        string                    // primary name (e.g. "gql")
	Aliases     []string                  // alternative names (e.g. "graphql", "GraphQL")
	Short       string                    // one-line description for parent's help listing
	Long        string                    // detailed help shown with --help
	Examples    []*utils.CommandHelp      // usage examples (printed via WriteCommandHelp)
	Flags       *flag.FlagSet             // scoped flags for this command
	Args        []argDef                  // positional arg descriptions (for help only)
	SubCommands []*command                // nested subcommands
	Run         func(args []string) error // handler; nil means subcommand-only
}

// ArgDef describes a positional argument for help output
type argDef struct {
	Name     string
	Required bool
	Desc     string
}

// Execute dispatches args to the correct subcommand or runs this command
func (cmd *command) Execute(args []string) error {
	// No args: show help if subcommand-only, or run with empty args
	if len(args) == 0 {
		if cmd.Run == nil {
			cmd.printHelp()
			return nil
		}
		return cmd.Run(args)
	}

	// Check for help request as first arg
	if isHelpArg(args[0]) {
		cmd.printHelp()
		return nil
	}

	// Try to match a subcommand
	if sub := cmd.findSub(args[0]); sub != nil {
		return sub.Execute(args[1:])
	}

	// Unknown subcommand — show error + help if this command only has subcommands
	if len(cmd.SubCommands) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		utils.PrintRed(fmt.Sprintf("unknown command %q for %s", args[0], cmd.Name))
		fmt.Println()
		cmd.printHelp()
		os.Exit(1)
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

		// Suppress Go's default usage dump so we can show hulak-styled errors
		cmd.Flags.Usage = func() {}
		cmd.Flags.SetOutput(io.Discard)

		if err := cmd.Flags.Parse(args); err != nil {
			return fmt.Errorf("%s\nSee 'hulak %s --help' for usage", err, cmd.Name)
		}

		if helpFlag {
			cmd.printHelp()
			return nil
		}

		args = cmd.Flags.Args()
	}

	if cmd.Run == nil {
		cmd.printHelp()
		return nil
	}

	return cmd.Run(args)
}

// findSub returns the subcommand matching name by Name or Aliases, or nil
func (cmd *command) findSub(name string) *command {
	for _, sub := range cmd.SubCommands {
		if sub.Name == name || slices.Contains(sub.Aliases, name) {
			return sub
		}
	}
	return nil
}

// printHelp prints the command's help to stdout in a style similar to gh CLI
func (cmd *command) printHelp() {
	if cmd.Long != "" {
		fmt.Println(cmd.Long)
		fmt.Println()
	}

	if len(cmd.SubCommands) > 0 {
		utils.PrintWarning("COMMANDS")
		var entries []*utils.CommandHelp
		for _, sub := range cmd.SubCommands {
			name := sub.Name
			if len(sub.Aliases) > 0 {
				name += " (" + strings.Join(sub.Aliases, ", ") + ")"
			}
			entries = append(entries, &utils.CommandHelp{
				Command:     name,
				Description: sub.Short,
			})
		}
		_ = utils.WriteCommandHelp(entries)
		fmt.Println()
	}

	if cmd.Flags != nil {
		printFlags(cmd.Flags)
	}

	if len(cmd.Args) > 0 {
		utils.PrintWarning("ARGUMENTS")
		var entries []*utils.CommandHelp
		for _, a := range cmd.Args {
			name := "<" + a.Name + ">"
			desc := a.Desc
			if a.Required {
				desc += " (required)"
			}
			entries = append(entries, &utils.CommandHelp{
				Command:     name,
				Description: desc,
			})
		}
		_ = utils.WriteCommandHelp(entries)
		fmt.Println()
	}

	if len(cmd.Examples) > 0 {
		utils.PrintWarning("EXAMPLES")
		for _, ex := range cmd.Examples {
			fmt.Printf("  $ %s\n", ex.Command)
			fmt.Printf("    %s\n", ex.Description)
		}
		fmt.Println()
	}

	utils.PrintWarning("LEARN MORE")
	fmt.Println("  Use `hulak <command> --help` for more information about a command.")
}

// printFlags renders the FLAGS section, merging short/long aliases onto one
// line (e.g. "-fp, --file-path  string") and skipping hidden flags
func printFlags(fs *flag.FlagSet) {
	// Collect which short names have a long alias
	longFor := make(map[string]string) // short → long
	for long, short := range flagAliases {
		if fs.Lookup(long) != nil && fs.Lookup(short) != nil {
			longFor[short] = long
		}
	}

	hasVisible := false
	fs.VisitAll(func(f *flag.Flag) {
		if !hiddenFlags[f.Name] && flagAliases[f.Name] == "" {
			hasVisible = true
		}
	})
	if !hasVisible {
		return
	}

	utils.PrintWarning("FLAGS")
	fs.VisitAll(func(f *flag.Flag) {
		// Skip hidden and long-form aliases (shown with their short form)
		if hiddenFlags[f.Name] || flagAliases[f.Name] != "" {
			return
		}

		// Build flag name: "-fp, --file-path" or just "-debug"
		label := "  -" + f.Name
		if long, ok := longFor[f.Name]; ok {
			label += ", --" + long
		}

		// Show type hint for non-bool flags using the flag's actual type
		if f.DefValue != "false" && f.DefValue != "true" {
			typeName := fmt.Sprintf("%T", f.Value)
			// flag.Value wraps types as *flag.stringValue, *flag.intValue, etc.
			// Extract the underlying type name from the wrapper
			switch {
			case strings.Contains(typeName, "int"):
				label += " int"
			case strings.Contains(typeName, "float"):
				label += " float"
			case strings.Contains(typeName, "duration"):
				label += " duration"
			default:
				label += " string"
			}
		}

		fmt.Println(label)
		usage := f.Usage
		if f.DefValue != "" && f.DefValue != "false" {
			usage += fmt.Sprintf(" (default %q)", f.DefValue)
		}
		fmt.Printf("    \t%s\n", usage)
	})
	fmt.Println()
}

// isHelpArg returns true if the argument is a help request
func isHelpArg(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}
