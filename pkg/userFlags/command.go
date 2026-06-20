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
	"quiet":       "q",
	"sequential":  "seq",
	"type":        "t",
	"out":         "o",
}

// hiddenFlags are omitted from help output entirely (utility flags
// that don't need to clutter command-specific help)
var hiddenFlags = map[string]bool{
	"h": true, "v": true,
}

// pathFlagNames lists flag names whose value is a filesystem path.
// Used by shell completion to pick the right value completer.
// When adding a new path-taking flag, register the name here so its
// value autocompletes to files.
var pathFlagNames = map[string]bool{
	"out": true, "o": true, // registerOutputFlag
	"file-path": true, "fp": true, // root --file-path/--fp
	"file": true, "f": true, // root --file/-f
	"ssh-identity": true, // run/init
	"dir":          true, // root --dir
	"dirseq":       true, // root --dirseq
}

// command represents a CLI command with optional subcommands and flags
type command struct {
	Name        string                    // primary name (e.g. "gql")
	Aliases     []string                  // alternative names (e.g. "graphql")
	Short       string                    // one-line description for parent's help listing
	Long        string                    // detailed help shown with --help
	Hidden      bool                      // omit from help listings (still callable)
	Examples    []*utils.CommandHelp      // usage examples (printed via WriteCommandHelp)
	Flags       *flag.FlagSet             // scoped flags for this command
	Args        []argDef                  // positional arg descriptions (for help only)
	SubCommands []*command                // nested subcommands
	Run         func(args []string) error // handler; nil means subcommand-only
}

// argDef describes a positional argument for help output and completion.
// Kind selects the value completer: "yaml" (run/gql), "file" (any path), or
// "" (no completion — for opaque values like secret keys or env names).
type argDef struct {
	Name     string
	Required bool
	Desc     string
	Kind     string
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

	// Unknown subcommand — show help, then return an error so the top-level
	// caller (main) can decide on the exit code. Help goes to stderr because
	// it's diagnostic output, not the user's intended program output.
	if len(cmd.SubCommands) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		fmt.Fprintln(os.Stderr)
		cmd.printHelp()
		return fmt.Errorf("unknown command %q for %s", args[0], cmd.Name)
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
				cmd.printHelp()
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

// visibleSubs returns subcommands that aren't hidden. Help, man, markdown,
// and completion generators all skip hidden commands the same way; go
// through this helper so the rule lives in one place.
func (cmd *command) visibleSubs() []*command {
	out := make([]*command, 0, len(cmd.SubCommands))
	for _, sub := range cmd.SubCommands {
		if !sub.Hidden {
			out = append(out, sub)
		}
	}
	return out
}

// flagPairings returns a short→long map for flags where both forms are
// registered on fs. Renderers (help, man, markdown, completion) use this
// to merge `-fp, --file-path` onto one line.
func flagPairings(fs *flag.FlagSet) map[string]string {
	out := make(map[string]string)
	for long, short := range flagAliases {
		if fs.Lookup(long) != nil && fs.Lookup(short) != nil {
			out[short] = long
		}
	}
	return out
}

// visitVisibleFlags calls fn for each visible flag on fs — that is, every
// flag except hidden ones and long-form aliases whose short partner is also
// registered (the short form pulls the long form in via flagPairings).
// Same iteration order as flag.FlagSet.VisitAll (alphabetical by name).
func visitVisibleFlags(fs *flag.FlagSet, fn func(*flag.Flag)) {
	fs.VisitAll(func(f *flag.Flag) {
		if hiddenFlags[f.Name] || flagAliases[f.Name] != "" {
			return
		}
		fn(f)
	})
}

// printHelp prints the command's help to stdout in a style similar to gh CLI
func (cmd *command) printHelp() {
	if cmd.Long != "" {
		fmt.Println(cmd.Long)
		fmt.Println()
	}

	if subs := cmd.visibleSubs(); len(subs) > 0 {
		utils.PrintSectionHeader("COMMANDS")
		var entries []*utils.CommandHelp
		for _, sub := range subs {
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
// line (e.g. "-fp, --file-path  string") and skipping hidden flags
func printFlags(fs *flag.FlagSet) {
	longFor := flagPairings(fs)

	hasVisible := false
	visitVisibleFlags(fs, func(*flag.Flag) { hasVisible = true })
	if !hasVisible {
		return
	}

	utils.PrintSectionHeader("FLAGS")
	visitVisibleFlags(fs, func(f *flag.Flag) {
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

// isHelpArg returns true if the argument is a help request
func isHelpArg(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}

// primaryName strips the "(alias, alias)" suffix from a command label so
// sort comparisons see only the canonical name.
func primaryName(label string) string {
	if i := strings.IndexByte(label, ' '); i >= 0 {
		return label[:i]
	}
	return label
}
