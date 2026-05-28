package userflags

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xaaha/hulak/pkg/userFlags/cli"
	"github.com/xaaha/hulak/pkg/userFlags/cliflags"
	"github.com/xaaha/hulak/pkg/utils"
)

//go:embed examples/*
var embeddedExamples embed.FS

// exampleType maps a subcommand argument to the embedded filename it scaffolds.
// Aliases are listed in exampleAliases below.
var exampleType = map[string]string{
	"api":        "example-api.hk.yaml",
	"formdata":   "example-formdata.hk.yaml",
	"urlencoded": "example-urlencoded.hk.yaml",
	"graphql":    "example-graphql.hk.yaml",
	"auth":       "example-auth.hk.yaml",
	"options":    utils.OptionsReference,
}

// exampleAliases maps alias → canonical type name. Kept narrow on purpose:
// adding aliases is a one-way ratchet (we can't remove one without breaking
// scripts), so we only ship them where the short form (or schema-literal
// form) is already in heavy use.
var exampleAliases = map[string]string{
	"gql":                "graphql",
	"urlencodedformdata": "urlencoded",
}

func newExampleCmd() *cli.Command {
	fs := flag.NewFlagSet("example", flag.ContinueOnError)
	out := cliflags.RegisterOutput(
		fs,
		"Output path. Directory (ends with '/' or no .yaml/.yml extension) → file lands inside with the canonical name; otherwise treated as a full file path. Parent directories are created.",
	)

	return &cli.Command{
		Name:  "example",
		Short: "Scaffold an example request file",
		Long: "Scaffold a starter request file into the current directory.\n\n" +
			"Each type writes a self-contained, schema-valid file that runs against a\n" +
			"public test API (jsonplaceholder, httpbin, trevorblades countries). The\n" +
			"'options' type writes a reference card listing every available request\n" +
			"field — it's not runnable on its own.\n\n" +
			"Use -o/--out to write somewhere other than the current directory. Pass a\n" +
			"directory to keep the canonical filename, or a full path to rename. Parent\n" +
			"directories are created on demand.\n\n" +
			"Idempotent: re-running for a path that already exists keeps the existing\n" +
			"file untouched.",
		Examples: []*utils.CommandHelp{
			{Command: "hulak example api", Description: "Scaffold a REST POST request"},
			{Command: "hulak example formdata", Description: "Scaffold a multipart/form-data POST"},
			{Command: "hulak example urlencoded", Description: "Scaffold an application/x-www-form-urlencoded POST"},
			{Command: "hulak example graphql", Description: "Scaffold a GraphQL query (alias: gql)"},
			{Command: "hulak example auth", Description: "Scaffold an OAuth 2.0 flow template"},
			{Command: "hulak example options", Description: "Scaffold the reference card of every request field"},
			{Command: "hulak example api -o requests/", Description: "Write into a subdirectory (canonical filename)"},
			{Command: "hulak example api -o requests/health.hk.yaml", Description: "Rename on write"},
			{Command: "hulak example", Description: "List available example types"},
		},
		Flags: fs,
		Args: []cli.ArgDef{
			{Name: "type", Desc: "Example type to scaffold (api, formdata, urlencoded, graphql, auth, options)"},
		},
		Run: func(args []string) error {
			if len(args) == 0 {
				return printExampleTypes()
			}
			return scaffoldExample(args[0], *out)
		},
	}
}

// scaffoldExample writes the embedded example for typeArg. outPath controls
// where it lands — see cliflags.ResolveOutputPath for the dir-vs-file rules.
// Resolves aliases (e.g. gql → graphql) before lookup. Returns an error if
// the type is unknown or the file cannot be written; a no-clobber on an
// existing file is reported as a warning and returns nil.
func scaffoldExample(typeArg, outPath string) error {
	name := strings.ToLower(typeArg)
	if canonical, ok := exampleAliases[name]; ok {
		name = canonical
	}

	filename, ok := exampleType[name]
	if !ok {
		return fmt.Errorf(
			"unknown example type %q — available: %s",
			typeArg, strings.Join(availableTypes(), ", "),
		)
	}

	dest, err := cliflags.ResolveOutputPath(outPath, filename, ".yaml", ".yml")
	if err != nil {
		return err
	}
	if utils.FileExists(dest) {
		utils.PrintWarningStderr(
			fmt.Sprintf("Kept existing '%s' (delete it to regenerate)", dest),
		)
		return nil
	}

	if parent := filepath.Dir(dest); parent != "." && parent != "" {
		if err := os.MkdirAll(parent, utils.DirPer); err != nil {
			return fmt.Errorf("creating parent dir for %q: %w", dest, err)
		}
	}

	content, err := embeddedExamples.ReadFile("examples/" + filename)
	if err != nil {
		return fmt.Errorf("reading embedded example %q: %w", filename, err)
	}
	if err := os.WriteFile(dest, content, utils.FilePer); err != nil {
		return fmt.Errorf("writing %q: %w", dest, err)
	}

	utils.PrintSuccessStderr(fmt.Sprintf("Created '%s'", dest))
	return nil
}

func availableTypes() []string {
	types := make([]string, 0, len(exampleType))
	for k := range exampleType {
		types = append(types, k)
	}
	sort.Strings(types)
	return types
}

// printExampleTypes renders the type → scaffolded-filename mapping as a
// borderless table to stdout, with aliases noted on stderr as a footnote.
// Stdout/stderr split: the table is data (pipeable), the alias note is
// metadata (hidden from `hulak example | grep ...`).
func printExampleTypes() error {
	rows := make([][]string, 0, len(exampleType))
	for _, name := range availableTypes() {
		rows = append(rows, []string{name, exampleType[name]})
	}
	if err := utils.PrintTable(
		os.Stdout,
		utils.StdoutHeaders([]string{"TYPE", "FILE"}),
		rows,
		0,
	); err != nil {
		return err
	}

	aliases := make([]string, 0, len(exampleAliases))
	for alias, canonical := range exampleAliases {
		aliases = append(aliases, fmt.Sprintf("%s → %s", alias, canonical))
	}
	sort.Strings(aliases)
	if len(aliases) > 0 {
		utils.PrintInfoStderr("Aliases: " + strings.Join(aliases, ", "))
	}
	return nil
}
