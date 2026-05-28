// Package cli holds the CLI dispatch core: the Command struct, its Execute
// method, help rendering, and small cross-cutting helpers (RequireVaultProject)
// shared by every leaf command package. Leaf packages build *Command trees;
// this package owns parsing and dispatch.
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
)

// FlagAliases maps long-form flag names to their short form.
// The long form is hidden from help output; only the short form is shown
// with both names on one line (e.g. "-fp, --file-path"). Exported so that
// gendocs renders man pages using the same aliasing convention as runtime
// help output.
var FlagAliases = map[string]string{
	"file-path":   "fp",
	"file":        "f",
	"environment": "env",
	"version":     "v",
	"help":        "h",
	"quiet":       "q",
	"sequential":  "seq",
	"type":        "t",
}

// HiddenFlags are omitted from help output entirely (utility flags
// that don't need to clutter command-specific help). Exported alongside
// FlagAliases for the same reason: gendocs shares the runtime's
// visibility rules.
var HiddenFlags = map[string]bool{
	"h": true, "v": true,
}

// Command represents a CLI command with optional subcommands and flags.
type Command struct {
	Name        string                    // primary name (e.g. "gql")
	Aliases     []string                  // alternative names (e.g. "graphql")
	Short       string                    // one-line description for parent's help listing
	Long        string                    // detailed help shown with --help
	Hidden      bool                      // omit from help listings (still callable)
	Examples    []*utils.CommandHelp      // usage examples (printed via WriteCommandHelp)
	Flags       *flag.FlagSet             // scoped flags for this command
	Args        []ArgDef                  // positional arg descriptions (for help only)
	SubCommands []*Command                // nested subcommands
	Run         func(args []string) error // handler; nil means subcommand-only
}

// ArgDef describes a positional argument for help output.
type ArgDef struct {
	Name     string
	Required bool
	Desc     string
}

// Execute dispatches args to the correct subcommand or runs this command.
func (cmd *Command) Execute(args []string) error {
	// No args: show help if subcommand-only, or run with empty args
	if len(args) == 0 {
		if cmd.Run == nil {
			cmd.PrintHelp()
			return nil
		}
		return cmd.Run(args)
	}

	// Check for help request as first arg
	if isHelpArg(args[0]) {
		cmd.PrintHelp()
		return nil
	}

	// Try to match a subcommand. Scan past leading flags so verb-after-flag
	// (`secrets keys --env prod list`) dispatches the same as verb-first
	// (`secrets keys list --env prod`). This mirrors the iterative flag-anywhere
	// parsing further down — order of flags vs verb is not significant.
	if len(cmd.SubCommands) > 0 {
		if idx, firstNonFlag := cmd.FindSubcommandIndex(args); idx >= 0 {
			sub := cmd.FindSub(args[idx])
			rest := append(append([]string(nil), args[:idx]...), args[idx+1:]...)
			return sub.Execute(rest)
		} else if firstNonFlag >= 0 {
			// First non-flag token is not a subcommand — that's a typo, not
			// a flag-handler invocation. Show help and exit non-zero.
			fmt.Fprintln(os.Stderr)
			cmd.PrintHelp()
			return fmt.Errorf("unknown command %q for %s", args[firstNonFlag], cmd.Name)
		}
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

		// Iterative parse: stdlib flag.Parse stops at the first non-flag argument,
		// so `hulak set FOO bar --env prod` would leave '--env prod' unparsed.
		// Loop, peeling off one positional per iteration, until all flags (in any position) are consumed.
		// After this, args holds only positionals.
		var positionals []string
		remaining := args
		for {
			if err := cmd.Flags.Parse(remaining); err != nil {
				return fmt.Errorf("%s\nSee 'hulak %s --help' for usage", err, cmd.Name)
			}
			if helpFlag {
				cmd.PrintHelp()
				return nil
			}
			if cmd.Flags.NArg() == 0 {
				break
			}
			positionals = append(positionals, cmd.Flags.Arg(0))
			remaining = cmd.Flags.Args()[1:]
		}
		args = positionals
	}

	if cmd.Run == nil {
		cmd.PrintHelp()
		return nil
	}

	return cmd.Run(args)
}

// FindSub returns the subcommand matching name by Name or Aliases, or nil.
func (cmd *Command) FindSub(name string) *Command {
	for _, sub := range cmd.SubCommands {
		if sub.Name == name || slices.Contains(sub.Aliases, name) {
			return sub
		}
	}
	return nil
}

// FindSubcommandIndex scans args for the first non-flag token, skipping
// flags and (where determinable from cmd.Flags) their values. Returns:
//
//   - matchIdx ≥ 0  : args[matchIdx] resolves to a subcommand of cmd.
//   - matchIdx == -1, firstNonFlag ≥ 0 : a positional was found but it
//     isn't a subcommand (typo).
//   - both -1       : args contains only flags / is empty.
//
// Flag-value pairing rules:
//   - `--name=value` inlines the value — the next arg is not consumed.
//   - Bool flags (per the flag.Value IsBoolFlag contract) take no value.
//   - Anything else is assumed to consume the next arg as its value.
//
// This is the dispatch-level analogue of the iterative flag-anywhere parsing
// in Execute: ordering between flags and the verb is not significant.
func (cmd *Command) FindSubcommandIndex(args []string) (matchIdx, firstNonFlag int) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "" {
			continue
		}
		if a[0] != '-' {
			if cmd.FindSub(a) != nil {
				return i, i
			}
			return -1, i
		}
		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") {
			// --foo=bar inlines the value; the next arg is not consumed.
			continue
		}
		if cmd.Flags != nil {
			if f := cmd.Flags.Lookup(name); f != nil {
				if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
					continue
				}
			}
		}
		// Unknown flag or known non-bool flag: assume the next arg is its value.
		i++
	}
	return -1, -1
}

// PrintHelp prints the command's help to stdout in a style similar to gh CLI.
func (cmd *Command) PrintHelp() {
	if cmd.Long != "" {
		fmt.Println(cmd.Long)
		fmt.Println()
	}

	if len(cmd.SubCommands) > 0 {
		utils.PrintSectionHeader("COMMANDS")
		var entries []*utils.CommandHelp
		for _, sub := range cmd.SubCommands {
			if sub.Hidden {
				continue
			}
			name := sub.Name
			if len(sub.Aliases) > 0 {
				name += " (" + strings.Join(sub.Aliases, ", ") + ")"
			}
			entries = append(entries, &utils.CommandHelp{
				Command:     name,
				Description: sub.Short,
			})
		}
		// Alphabetical so users can scan; registration order is meaningless
		// to readers. Sort by primary name (everything before the first
		// space, i.e. before the "(alias)" suffix).
		slices.SortFunc(entries, func(a, b *utils.CommandHelp) int {
			return strings.Compare(primaryName(a.Command), primaryName(b.Command))
		})
		_ = utils.WriteCommandHelp(entries)
		fmt.Println()
	}

	if cmd.Flags != nil {
		printFlags(cmd.Flags)
	}

	if len(cmd.Args) > 0 {
		utils.PrintSectionHeader("ARGUMENTS")
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
		utils.PrintSectionHeader("EXAMPLES")
		for _, ex := range cmd.Examples {
			fmt.Printf("  $ %s\n", ex.Command)
			fmt.Printf("    %s\n", ex.Description)
		}
		fmt.Println()
	}

	utils.PrintSectionHeader("LEARN MORE")
	fmt.Println("  Use `hulak <command> --help` for more information about a command.")
}

// printFlags renders the FLAGS section, merging short/long aliases onto one
// line (e.g. "-fp, --file-path  string") and skipping hidden flags.
func printFlags(fs *flag.FlagSet) {
	// Collect which short names have a long alias
	longFor := make(map[string]string) // short → long
	for long, short := range FlagAliases {
		if fs.Lookup(long) != nil && fs.Lookup(short) != nil {
			longFor[short] = long
		}
	}

	hasVisible := false
	fs.VisitAll(func(f *flag.Flag) {
		if !HiddenFlags[f.Name] && FlagAliases[f.Name] == "" {
			hasVisible = true
		}
	})
	if !hasVisible {
		return
	}

	utils.PrintSectionHeader("FLAGS")
	fs.VisitAll(func(f *flag.Flag) {
		// Skip hidden and long-form aliases (shown with their short form)
		if HiddenFlags[f.Name] || FlagAliases[f.Name] != "" {
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
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" && f.DefValue != "0s" {
			usage += fmt.Sprintf(" (default %q)", f.DefValue)
		}
		fmt.Printf("    \t%s\n", usage)
	})
	fmt.Println()
}

// isHelpArg returns true if the argument is a help request.
func isHelpArg(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}

// primaryName strips the "(alias, alias)" suffix from a command label so
// sort comparisons see only the canonical name.
func primaryName(label string) string {
	if before, _, ok := strings.Cut(label, " "); ok {
		return before
	}
	return label
}
