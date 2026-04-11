package userflags

import (
	"bytes"
	"flag"
	"strings"
	"testing"
)

func TestExecuteDispatchesToSubcommand(t *testing.T) {
	called := ""
	root := &Command{
		Name: "root",
		SubCommands: []*Command{
			{
				Name:  "sub1",
				Short: "first sub",
				Run: func(_ []string) error {
					called = "sub1"
					return nil
				},
			},
			{
				Name:  "sub2",
				Short: "second sub",
				Run: func(_ []string) error {
					called = "sub2"
					return nil
				},
			},
		},
	}

	if err := root.Execute([]string{"sub2"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != "sub2" {
		t.Errorf("expected sub2 to be called, got %q", called)
	}
}

func TestExecuteAliasMatching(t *testing.T) {
	called := false
	root := &Command{
		Name: "root",
		SubCommands: []*Command{
			{
				Name:    "gql",
				Aliases: []string{"graphql", "GraphQL"},
				Short:   "GraphQL explorer",
				Run: func(_ []string) error {
					called = true
					return nil
				},
			},
		},
	}

	tests := []string{"gql", "graphql", "GraphQL"}
	for _, name := range tests {
		called = false
		if err := root.Execute([]string{name}); err != nil {
			t.Fatalf("unexpected error for %q: %v", name, err)
		}
		if !called {
			t.Errorf("expected gql handler to be called for alias %q", name)
		}
	}
}

func TestExecutePassesArgsToRun(t *testing.T) {
	var gotArgs []string
	root := &Command{
		Name: "root",
		SubCommands: []*Command{
			{
				Name: "migrate",
				Run: func(args []string) error {
					gotArgs = args
					return nil
				},
			},
		},
	}

	if err := root.Execute([]string{"migrate", "file1.json", "file2.json"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "file1.json" || gotArgs[1] != "file2.json" {
		t.Errorf("expected [file1.json file2.json], got %v", gotArgs)
	}
}

func TestExecuteWithFlags(t *testing.T) {
	var envVal string
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.StringVar(&envVal, "env", "", "environment")

	root := &Command{
		Name: "root",
		SubCommands: []*Command{
			{
				Name:  "init",
				Flags: fs,
				Run: func(_ []string) error {
					return nil
				},
			},
		},
	}

	if err := root.Execute([]string{"init", "-env", "staging"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envVal != "staging" {
		t.Errorf("expected env=staging, got %q", envVal)
	}
}

func TestExecuteHelpFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"--help flag", []string{"--help"}},
		{"-h flag", []string{"-h"}},
		{"help subcommand", []string{"help"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runCalled := false
			cmd := &Command{
				Name: "test",
				Long: "Test command help text",
				Run: func(_ []string) error {
					runCalled = true
					return nil
				},
			}

			if err := cmd.Execute(tc.args); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if runCalled {
				t.Error("Run should not be called when help is requested")
			}
		})
	}
}

func TestExecuteSubcommandHelp(t *testing.T) {
	runCalled := false
	root := &Command{
		Name: "root",
		SubCommands: []*Command{
			{
				Name:  "init",
				Short: "Initialize project",
				Long:  "Initialize a new hulak project with default settings",
				Run: func(_ []string) error {
					runCalled = true
					return nil
				},
			},
		},
	}

	// "init --help" should show init's help, not run init
	if err := root.Execute([]string{"init", "--help"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runCalled {
		t.Error("Run should not be called when help is requested for subcommand")
	}
}

func TestExecuteNoArgsSubcommandOnly(t *testing.T) {
	// Command with no Run and no args should print help without error
	cmd := &Command{
		Name: "root",
		Long: "Root command",
		SubCommands: []*Command{
			{Name: "sub1", Short: "first"},
		},
	}

	if err := cmd.Execute(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteNoArgsWithRun(t *testing.T) {
	called := false
	cmd := &Command{
		Name: "root",
		Run: func(_ []string) error {
			called = true
			return nil
		},
	}

	if err := cmd.Execute(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("Run should be called when no args and Run is set")
	}
}

func TestPrintHelp(t *testing.T) {
	fs := flag.NewFlagSet("gql", flag.ContinueOnError)
	fs.String("env", "", "Environment to use")

	cmd := &Command{
		Name:  "gql",
		Short: "GraphQL explorer",
		Long:  "Open the GraphQL explorer for files and directories",
		Flags: fs,
		Args: []ArgDef{
			{Name: "path", Required: true, Desc: "File or directory path"},
		},
		SubCommands: []*Command{
			{Name: "create", Short: "Create a new GraphQL file"},
		},
	}

	var buf bytes.Buffer
	cmd.printHelp(&buf)
	output := buf.String()

	// Check all sections are present
	checks := []string{
		"Open the GraphQL explorer",
		"Subcommands:",
		"create",
		"Create a new GraphQL file",
		"Flags:",
		"-env",
		"Arguments:",
		"<path>",
		"(required)",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("help output missing %q\nGot:\n%s", check, output)
		}
	}
}

func TestFindSub(t *testing.T) {
	sub := &Command{
		Name:    "gql",
		Aliases: []string{"graphql"},
	}
	root := &Command{
		Name:        "root",
		SubCommands: []*Command{sub},
	}

	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"by name", "gql", true},
		{"by alias", "graphql", true},
		{"not found", "unknown", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := root.findSub(tc.input)
			if tc.found && result == nil {
				t.Errorf("expected to find subcommand for %q", tc.input)
			}
			if !tc.found && result != nil {
				t.Errorf("expected nil for %q, got %v", tc.input, result)
			}
		})
	}
}
