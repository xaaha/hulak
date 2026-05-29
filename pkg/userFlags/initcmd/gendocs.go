// Generates the hulak(1) man page from the live command tree.
// Run via: go generate ./pkg/userFlags  or  hulak gendocs
package initcmd

//go:generate go run ../../.. gendocs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/userFlags/cli"
)

// NewGenDocs builds the hidden `hulak gendocs` command. The rootFn closure
// is the top-level dispatcher's tree builder. gendocs walks it to render the
// man page. Passed as a closure (rather than a baked-in *cli.Command) so the
// page reflects the live tree at run time.
func NewGenDocs(rootFn func() *cli.Command) *cli.Command {
	return &cli.Command{
		Name:   "gendocs",
		Hidden: true,
		Short:  "Regenerate the hulak man page",
		Long:   "Regenerate man/hulak.1 from the live command tree.\n\nRun this before tagging a release to keep the man page in sync with code.",
		Run: func(_ []string) error {
			repoRoot, err := findRepoRoot()
			if err != nil {
				return err
			}

			cmdRoot := rootFn()
			manPath := filepath.Join(repoRoot, "man", "hulak.1")

			if err := writeFile(manPath, func(w io.Writer) { generateManPage(w, cmdRoot) }); err != nil {
				return fmt.Errorf("writing %s: %w", manPath, err)
			}
			fmt.Fprintf(os.Stderr, "wrote %s\n", manPath)

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

// generateManPage writes a slim hulak(1) man page in roff format. The page
// orients the reader to the command set and the picker behavior, then sends
// them to `hulak <command> --help` for flag detail.
func generateManPage(w io.Writer, root *cli.Command) {
	// Fixed date so CI regeneration is stable across the calendar.
	// Bump when cutting a release if you want the date to advance.
	date := "May 2026"

	p := func(format string, a ...any) { fmt.Fprintf(w, format+"\n", a...) }

	p(`.TH HULAK 1 "%s" "hulak" "General Commands Manual"`, date)
	p("")

	p(".SH NAME")
	p("hulak \\- file\\-based API client with encrypted secrets")
	p("")

	p(".SH SYNOPSIS")
	p(".B hulak")
	p("[<command>] [options]")
	p(".br")
	p(".B hulak")
	p("[\\-env <environment>] [\\-fp|\\-f|\\-dir|\\-dirseq <target>] [\\-debug]")
	p("")

	p(".SH DESCRIPTION")
	p("Hulak runs API requests defined in YAML files. Requests live in")
	p(".B .hk.yaml")
	p("files. Secrets live in")
	p(".B .hulak/store.age")
	p("(encrypted, default) or")
	p(".B env/*.env")
	p("(classic mode).")
	p(".PP")
	p("Run")
	p(".B hulak <command> \\-\\-help")
	p("for flags and per\\-command examples.")
	p("")

	p(".SH COMMANDS")
	for _, sub := range root.SubCommands {
		if sub.Hidden {
			continue
		}
		p(".TP")
		name := sub.Name
		if len(sub.Aliases) > 0 {
			name += " (alias: " + strings.Join(sub.Aliases, ", ") + ")"
		}
		p(".B %s", name)
		p("%s", manEscape(sub.Short))
	}
	p("")

	p(".SH PICKER BEHAVIOR")
	p("Omitting")
	p(".B \\-\\-env")
	p("opens an interactive picker.")
	p(".PP")
	p(".B hulak run")
	p("and")
	p(".B hulak gql")
	p("only prompt when files reference")
	p(".B {{.key}}")
	p("variables.")
	p(".PP")
	p(".B hulak secrets")
	p("subcommands prompt every time, except")
	p(".B secrets list")
	p("which lists env names and takes no")
	p(".B \\-\\-env")
	p(".")
	p(".PP")
	p("Non\\-interactive shells require")
	p(".B \\-\\-env <name>")
	p(".")
	p("")

	p(".SH ENVIRONMENT SETUP")
	p("Create a project and initialize:")
	p(".PP")
	p(".RS")
	p("hulak init")
	p(".RE")
	p(".PP")
	p("Scaffold named env files at init time:")
	p(".PP")
	p(".RS")
	p("hulak init \\-env staging prod")
	p(".RE")
	p(".PP")
	p("Use")
	p(".B hulak init classic")
	p("for the plaintext")
	p(".B env/")
	p("layout instead of the encrypted vault.")
	p("")

	p(".SH INSTALLATION")
	p(".PP")
	p(".RS")
	p("brew install xaaha/tap/hulak")
	p(".RE")
	p(".PP")
	p("Or:")
	p(".PP")
	p(".RS")
	p("go install github.com/xaaha/hulak@latest")
	p(".RE")
	p("")

	p(".SH SEE ALSO")
	p("Project README and docs:")
	p(".RS")
	p("https://github.com/xaaha/hulak")
	p(".RE")
	p(".PP")
	p("Schema:")
	p(".RS")
	p("https://raw.githubusercontent.com/xaaha/hulak/refs/heads/main/assets/schema.json")
	p(".RE")
	p(".PP")
	p("For the live CLI surface, run")
	p(".B hulak help")
	p("or")
	p(".B hulak <command> \\-\\-help")
	p(".")
	p("")

	p(".SH COPYRIGHT")
	p("Copyright (c) 2025 Pratik Thapa. Released under the MIT License.")
	p("")

	p(".SH SOURCE CODE")
	p("https://github.com/xaaha/hulak")
	p("")

	p(".SH AUTHOR")
	p("xaaha")
}

func manEscape(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "-", "\\-")
	s = strings.ReplaceAll(s, ".", "\\&.")
	return s
}
