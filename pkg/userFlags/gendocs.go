// Generates man page and CLI markdown reference from the command tree.
// Run via: go generate ./pkg/userFlags  or  hulak gendocs
package userflags

//go:generate go run ../.. gendocs

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// newGenDocsCmd returns a hidden subcommand that regenerates man/hulak.1
// and docs/cli.md from the live command tree.
func newGenDocsCmd() *command {
	return &command{
		Name:   "gendocs",
		Hidden: true,
		Short:  "Regenerate man page and CLI markdown",
		Long:   "Regenerate man/hulak.1 and docs/cli.md from the live command tree.\n\nRun this before tagging a release to keep docs in sync with code.",
		Run: func(_ []string) error {
			root, err := findRepoRoot()
			if err != nil {
				return err
			}

			manPath := filepath.Join(root, "man", "hulak.1")
			mdPath := filepath.Join(root, "docs", "cli.md")

			if err := writeFile(manPath, generateManPage); err != nil {
				return fmt.Errorf("writing %s: %w", manPath, err)
			}
			fmt.Fprintf(os.Stderr, "wrote %s\n", manPath)

			if err := writeFile(mdPath, generateCLIMarkdown); err != nil {
				return fmt.Errorf("writing %s: %w", mdPath, err)
			}
			fmt.Fprintf(os.Stderr, "wrote %s\n", mdPath)

			return nil
		},
	}
}

func writeFile(path string, fn func(io.Writer)) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	fn(f)
	return f.Close()
}

// findRepoRoot walks up from cwd looking for go.mod.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod in any parent directory")
		}
		dir = parent
	}
}

// generateManPage writes the hulak(1) man page in roff format.
func generateManPage(w io.Writer) {
	root := subCommands()
	// Use a fixed epoch so CI regeneration doesn't produce a diff
	// just because the calendar month rolled over. Bump this when
	// cutting a release if you want the man page date to advance.
	date := "May 2026"

	p := func(format string, a ...any) { fmt.Fprintf(w, format+"\n", a...) }

	// Header
	p(`.TH HULAK 1 "%s" "hulak" "General Commands Manual"`, date)
	p("")

	// NAME
	p(".SH NAME")
	p("hulak \\- file-based API client for the terminal")
	p("")

	// SYNOPSIS
	p(".SH SYNOPSIS")
	p(".B hulak")
	p(".br")
	for _, sub := range root.SubCommands {
		if sub.Hidden {
			continue
		}
		synopsisLine := manSynopsis(sub, "hulak")
		p("%s", synopsisLine)
		p(".br")
	}
	// Shorthand forms
	p(".PP")
	p("Supported shorthand forms:")
	p(".br")
	p(".B hulak")
	p("[\\-env <environment>] \\-fp <file_path> [\\-debug]")
	p(".br")
	p(".B hulak")
	p("[\\-env <environment>] \\-f <file_name> [\\-debug]")
	p(".br")
	p(".B hulak")
	p("[\\-env <environment>] \\-dir <directory_path> [\\-debug]")
	p(".br")
	p(".B hulak")
	p("[\\-env <environment>] \\-dirseq <directory_path> [\\-debug]")
	p("")

	// DESCRIPTION
	p(".SH DESCRIPTION")
	p("Hulak runs API requests defined in YAML files. The recommended style is command\\-first usage with")
	p(".B hulak run <path>")
	p(", where <path> is either a single request file or a directory of request files.")
	p(".PP")
	p("Running hulak with no file or directory target opens the interactive picker. Hulak asks you to pick")
	p("a request file first, and only asks for an environment when the selected request uses template values")
	p("like {{.token}}.")
	p(".PP")
	p("Hulak also includes a GraphQL explorer through the gql subcommand for schema discovery, operation")
	p("search, query building, execution, and saving generated files.")
	p("")

	// COMMANDS
	p(".SH COMMANDS")
	for _, sub := range root.SubCommands {
		if sub.Hidden {
			continue
		}
		p(".TP")
		p(".B %s", sub.Name)
		p("%s", manEscape(sub.Short))
	}
	p("")

	// Per-command flag sections
	runCmd := root.findSub("run")
	if runCmd != nil && runCmd.Flags != nil {
		p(".SH RUN FLAGS")
		manWriteFlags(w, runCmd.Flags)
	}

	// Global shorthand flags
	p(".SH GLOBAL SHORTHAND FLAGS")
	p("These root flags remain supported, but")
	p(".B hulak run <path>")
	p("is the preferred form for examples and onboarding.")
	p("")
	manWriteFlags(w, flag.CommandLine)

	// INTERACTIVE MODE
	p(".SH INTERACTIVE MODE")
	p(".TP")
	p(".B hulak")
	p("Open the interactive request picker.")
	p(".TP")
	p(".B hulak \\-env staging")
	p("Open the interactive picker with a preselected environment.")
	p(".PP")
	p("In non\\-interactive shells, you should pass a file or directory target instead of relying on the picker.")
	p("")

	// ENVIRONMENT SETUP
	p(".SH ENVIRONMENT SETUP")
	p("Create a project directory and initialize it with:")
	p(".PP")
	p(".RS")
	p("hulak init")
	p(".RE")
	p(".PP")
	p("To create named environment files immediately:")
	p(".PP")
	p(".RS")
	p("hulak init \\-env staging prod")
	p(".RE")
	p(".PP")
	p("Hulak uses env/ files only when request files need template resolution such as {{.key}}.")
	p("")

	// EXAMPLES
	p(".SH EXAMPLES")
	for _, sub := range root.SubCommands {
		if sub.Hidden || len(sub.Examples) == 0 {
			continue
		}
		p("%s:", sub.Name)
		p(".PP")
		for _, ex := range sub.Examples {
			p(".RS")
			p("%s", manEscape(ex.Command))
			p(".RE")
		}
		p(".PP")
	}

	// INSTALLATION
	p(".SH INSTALLATION")
	p("Homebrew tap (cask):")
	p(".PP")
	p(".RS")
	p("brew install \\-\\-cask xaaha/tap/hulak")
	p(".RE")
	p(".PP")
	p("The Homebrew tap publishes Hulak as a cask generated by GoReleaser. That cask links the hulak binary")
	p("and the bundled manpage from the release archive.")
	p(".PP")
	p("go install:")
	p(".PP")
	p(".RS")
	p("go install github.com/xaaha/hulak@latest")
	p(".RE")
	p(".PP")
	p("Build from source:")
	p(".PP")
	p(".RS")
	p("git clone https://github.com/xaaha/hulak.git")
	p(".br")
	p("cd hulak")
	p(".br")
	p("go build \\-o hulak")
	p(".RE")
	p("")

	// SCHEMA
	p(".SH SCHEMA")
	p("Schema URL:")
	p(".PP")
	p(".RS")
	p("https://raw.githubusercontent.com/xaaha/hulak/refs/heads/main/assets/schema.json")
	p(".RE")
	p("")

	// DOCUMENTATION
	p(".SH DOCUMENTATION")
	p("CLI reference:")
	p(".RS")
	p("https://github.com/xaaha/hulak/blob/main/docs/cli.md")
	p(".RE")
	p(".PP")
	p("GraphQL explorer guide:")
	p(".RS")
	p("https://github.com/xaaha/hulak/blob/main/docs/graphql-explorer.md")
	p(".RE")
	p(".PP")
	p("Auth 2.0 guide:")
	p(".RS")
	p("https://github.com/xaaha/hulak/blob/main/docs/auth20.md")
	p(".RE")
	p(".PP")
	p("For the current CLI surface from the binary itself, use:")
	p(".PP")
	p(".RS")
	p("hulak help")
	p(".br")
	p("hulak <command> \\-\\-help")
	p(".RE")
	p("")

	// COPYRIGHT
	p(".SH COPYRIGHT")
	p("Copyright (c) 2025 Pratik Thapa")
	p(".PP")
	p("This software is released under the MIT License. For details, see the LICENSE file in the project repository.")
	p("")

	// SOURCE CODE
	p(".SH SOURCE CODE")
	p("https://github.com/xaaha/hulak")
	p("")

	// AUTHOR
	p(".SH AUTHOR")
	p("xaaha")
}

// generateCLIMarkdown writes the CLI reference in GitHub-flavored markdown.
func generateCLIMarkdown(w io.Writer) {
	root := subCommands()

	p := func(format string, a ...any) { fmt.Fprintf(w, format+"\n", a...) }

	p("# Hulak CLI Reference")
	p("")
	p("Hulak supports two ways of running requests:")
	p("")
	p("- **Recommended:** command-first usage such as `hulak run path/to/file.yaml`")
	p("- **Supported shorthand:** root flags such as `hulak -fp path/to/file.yaml` or `hulak -dir path/to/dir/`")
	p("")
	p("If you are documenting or teaching Hulak, prefer the command-first form.")
	p("")

	// Quick Start
	p("## Quick Start")
	p("")
	p("```bash")
	p("# run one request file")
	p("hulak run path/to/file.yaml")
	p("")
	p("# run one request file with a specific environment")
	p("hulak run path/to/file.yaml --env staging")
	p("")
	p("# run a directory concurrently")
	p("hulak run path/to/dir/")
	p("")
	p("# run a directory sequentially")
	p("hulak run path/to/dir/ --sequential")
	p("")
	p("# open the interactive picker")
	p("hulak")
	p("```")
	p("")

	// Discovering Commands
	p("## Discovering Commands")
	p("")
	p("Use these help entry points when you want the current CLI surface from the binary itself:")
	p("")
	p("```bash")
	p("hulak help")
	p("hulak run --help")
	p("hulak gql --help")
	p("hulak secrets --help")
	p("```")
	p("")
	p("For command-specific help, prefer `hulak <command> --help`.")
	p("")

	// Command Index
	p("## Command Index")
	p("")
	p("| Command   | Purpose                                      | Example                               |")
	p("| --------- | -------------------------------------------- | ------------------------------------- |")
	for _, sub := range root.SubCommands {
		if sub.Hidden {
			continue
		}
		example := ""
		if len(sub.Examples) > 0 {
			example = sub.Examples[0].Command
		}
		p("| `%s` | %s | `%s` |", sub.Name, sub.Short, example)
	}
	p("")

	// Core Behaviors
	p("## Core Behaviors")
	p("")
	p("### Interactive mode")
	p("")
	p("Running `hulak` with no file or directory target opens the interactive picker.")
	p("")
	p("- Hulak asks you to choose a request file first.")
	p("- It only asks for an environment if the selected request uses template values like `{{.key}}`.")
	p("- In non-interactive shells, you should pass a file or directory target instead.")
	p("")
	p("### Environment selection")
	p("")
	p("When `--env` is omitted, behavior depends on the command:")
	p("")
	p("- **`run` and `gql`**: open the interactive picker if the request files reference environment variables (`{{.key}}`). If a request needs no env, no picker.")
	p("- **`hulak secrets edit`**: always opens the picker — no default. Pass `--env <name>` (including for new envs you want to create).")
	p("- **`hulak secrets set`, `get`, `delete`, `keys`**: default to `global`. These are non-interactive operations on a known env; the default keeps scripts terse.")
	p("- **`hulak secrets list`**: takes no `--env` (it lists envs themselves).")
	p("")
	p("All commands above accept `--env` / `--environment` to bypass any picker or default explicitly.")
	p("")

	// Commands
	p("## Commands")
	p("")
	for _, sub := range root.SubCommands {
		if sub.Hidden {
			continue
		}
		mdWriteCommand(w, sub)
	}

	// Root Flags
	p("## Supported Root Flags (Shorthand)")
	p("")
	p("These are still supported. They are useful when you want the older root-flag style or need file-name search behavior.")
	p("")
	p("| Flag                    | Meaning                                                    | Example                                           |")
	p("| ----------------------- | ---------------------------------------------------------- | ------------------------------------------------- |")
	p("| `-env`, `--environment` | Select an environment for root-flag execution              | `hulak -env prod -fp requests/get-user.hk.yaml`   |")
	p("| `-fp`, `--file-path`    | Run one exact file path                                    | `hulak -fp requests/get-user.hk.yaml`             |")
	p("| `-f`, `--file`          | Search for matching file names recursively and run matches | `hulak -f getUser`                                |")
	p("| `-dir`                  | Run a directory concurrently                               | `hulak -dir ./requests/`                          |")
	p("| `-dirseq`               | Run a directory sequentially                               | `hulak -dirseq ./requests/`                       |")
	p("| `-debug`                | Enable debug output                                        | `hulak -fp requests/get-user.hk.yaml -debug`      |")
	p("| `-timeout`              | Per-request timeout, e.g. `5m` or `90s`                    | `hulak -fp requests/get-user.hk.yaml -timeout 2m` |")
	p("| `-v`, `--version`       | Print version                                              | `hulak --version`                                 |")
	p("| `-h`, `--help`          | Print help                                                 | `hulak --help`                                    |")
	p("")
	p("Use the shorthand form when it fits your workflow, but prefer `hulak run ...` in examples and onboarding material.")
}

// mdWriteCommand writes a markdown section for a single command.
func mdWriteCommand(w io.Writer, cmd *command) {
	p := func(format string, a ...any) { fmt.Fprintf(w, format+"\n", a...) }

	p("### `%s`", cmd.Name)
	p("")

	if cmd.Long != "" {
		desc := cmd.Long
		if cmd.Name == "secrets" {
			desc += " See [docs/store.md](./store.md) for the full encryption model and team-sharing flows."
		}
		p("%s", desc)
		p("")
	} else if cmd.Short != "" {
		p("%s", cmd.Short)
		p("")
	}

	if len(cmd.Aliases) > 0 {
		p("Aliases:")
		p("")
		for _, a := range cmd.Aliases {
			p("- `%s`", a)
		}
		p("")
	}

	if len(cmd.Examples) > 0 {
		p("```bash")
		for _, ex := range cmd.Examples {
			p("%s", ex.Command)
		}
		p("```")
		p("")
	}

	// Flags table
	if cmd.Flags != nil {
		flags := mdCollectFlags(cmd.Flags)
		if len(flags) > 0 {
			p("Supported flags:")
			p("")
			p("| Flag | Meaning |")
			p("| ---- | ------- |")
			for _, f := range flags {
				p("| %s | %s |", f.label, f.usage)
			}
			p("")
		}
	}

	switch cmd.Name {
	case "run":
		p("Notes:")
		p("")
		p("- `hulak run` accepts either a file path or a directory path.")
		p("- Directories run concurrently by default.")
		p("- `hulak run path/to/file.yaml --debug --env staging` is supported; trailing flags after the path are parsed correctly.")
		p("")
	case "init":
		p("Notes:")
		p("")
		p("- `hulak init` creates the default setup, including `env/global.env` and the example API options file.")
		p("- On `init`, `-env` is a **boolean setup flag**, not an environment selector. It tells Hulak to create the named env files you pass after it.")
		p("")
	case "gql":
		p("Read the full guide in [graphql-explorer.md](./graphql-explorer.md).")
		p("")
	case "secrets":
		mdWriteSecretsSubcommands(w, cmd)
	case "help":
		p("For command-specific help, use:")
		p("")
		p("```bash")
		p("hulak <command> --help")
		p("```")
		p("")
	}
}

// mdWriteSecretsSubcommands writes the subcommand table for secrets.
func mdWriteSecretsSubcommands(w io.Writer, cmd *command) {
	p := func(format string, a ...any) { fmt.Fprintf(w, format+"\n", a...) }

	p("| Subcommand | Notes |")
	p("| ---------- | ----- |")
	for _, sub := range cmd.SubCommands {
		name := "`" + sub.Name + "`"
		if len(sub.Aliases) > 0 {
			aliases := make([]string, len(sub.Aliases))
			for i, a := range sub.Aliases {
				aliases[i] = "`" + a + "`"
			}
			name += " (" + strings.Join(aliases, ", ") + ")"
		}
		p("| %s | %s |", name, sub.Short)
	}
	p("")
	p("**GUI editors** for `edit`: pass the wait flag in `$EDITOR` so hulak blocks until you save. e.g. `EDITOR=\"zed --wait\"` or `EDITOR=\"code -w\"`. Without it, the editor returns immediately and the file is read back unchanged.")
	p("")
}

// --- man page helpers ---

func manSynopsis(cmd *command, parent string) string {
	var b strings.Builder
	fmt.Fprintf(&b, ".B %s %s", parent, cmd.Name)

	if cmd.Flags != nil {
		cmd.Flags.VisitAll(func(f *flag.Flag) {
			if hiddenFlags[f.Name] || flagAliases[f.Name] != "" {
				return
			}
			fmt.Fprintf(&b, " [\\-\\-%s]", f.Name)
		})
	}

	for _, a := range cmd.Args {
		if a.Required {
			fmt.Fprintf(&b, " <%s>", a.Name)
		} else {
			fmt.Fprintf(&b, " [%s]", a.Name)
		}
	}

	if len(cmd.SubCommands) > 0 {
		fmt.Fprintf(&b, " <subcommand> [options]")
	}

	return b.String()
}

func manWriteFlags(w io.Writer, fs *flag.FlagSet) {
	p := func(format string, a ...any) { fmt.Fprintf(w, format+"\n", a...) }

	longFor := make(map[string]string)
	for long, short := range flagAliases {
		if fs.Lookup(long) != nil && fs.Lookup(short) != nil {
			longFor[short] = long
		}
	}

	fs.VisitAll(func(f *flag.Flag) {
		if hiddenFlags[f.Name] || flagAliases[f.Name] != "" {
			return
		}

		label := "\\-\\-" + f.Name
		if long, ok := longFor[f.Name]; ok {
			label += ", \\-\\-" + long
		}

		if f.DefValue != "false" && f.DefValue != "true" {
			label += " <" + flagTypeName(f) + ">"
		}

		p(".TP")
		p(".B %s", label)
		p("%s", manEscape(f.Usage))
	})
	p("")
}

func manEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "-", "\\-")
	s = strings.ReplaceAll(s, ".", "\\&.")
	return s
}

// --- markdown helpers ---

type mdFlag struct {
	label string
	usage string
}

func mdCollectFlags(fs *flag.FlagSet) []mdFlag {
	longFor := make(map[string]string)
	for long, short := range flagAliases {
		if fs.Lookup(long) != nil && fs.Lookup(short) != nil {
			longFor[short] = long
		}
	}

	var flags []mdFlag
	fs.VisitAll(func(f *flag.Flag) {
		if hiddenFlags[f.Name] || flagAliases[f.Name] != "" {
			return
		}

		label := "`--" + f.Name + "`"
		if long, ok := longFor[f.Name]; ok {
			label += ", `--" + long + "`"
		}

		flags = append(flags, mdFlag{label: label, usage: f.Usage})
	})
	return flags
}

func flagTypeName(f *flag.Flag) string {
	typeName := fmt.Sprintf("%T", f.Value)
	switch {
	case strings.Contains(typeName, "int"):
		return "int"
	case strings.Contains(typeName, "float"):
		return "float"
	case strings.Contains(typeName, "duration"):
		return "duration"
	case strings.Contains(typeName, "bool"):
		return "bool"
	default:
		return "string"
	}
}
