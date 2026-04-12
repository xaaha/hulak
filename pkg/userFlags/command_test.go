package userflags

import (
	"bytes"
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

func TestExecuteDispatchesToSubcommand(t *testing.T) {
	called := ""
	root := &command{
		Name: "root",
		SubCommands: []*command{
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
	root := &command{
		Name: "root",
		SubCommands: []*command{
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
	root := &command{
		Name: "root",
		SubCommands: []*command{
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

	root := &command{
		Name: "root",
		SubCommands: []*command{
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
			cmd := &command{
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
	root := &command{
		Name: "root",
		SubCommands: []*command{
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
	// command with no Run and no args should print help without error
	cmd := &command{
		Name: "root",
		Long: "Root command",
		SubCommands: []*command{
			{Name: "sub1", Short: "first"},
		},
	}

	if err := cmd.Execute(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteNoArgsWithRun(t *testing.T) {
	called := false
	cmd := &command{
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

	cmd := &command{
		Name:  "gql",
		Short: "GraphQL explorer",
		Long:  "Open the GraphQL explorer for files and directories",
		Flags: fs,
		Args: []argDef{
			{Name: "path", Required: true, Desc: "File or directory path"},
		},
		SubCommands: []*command{
			{Name: "create", Short: "Create a new GraphQL file"},
		},
	}

	output := captureStdout(t, func() {
		cmd.printHelp()
	})

	checks := []string{
		"Open the GraphQL explorer",
		"COMMANDS",
		"create",
		"Create a new GraphQL file",
		"FLAGS",
		"-env",
		"ARGUMENTS",
		"<path>",
		"(required)",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("help output missing %q\nGot:\n%s", check, output)
		}
	}
}

func TestPrintHelpShowsAliases(t *testing.T) {
	cmd := &command{
		Name: "root",
		Long: "Root command",
		SubCommands: []*command{
			{Name: "list", Aliases: []string{"ls"}, Short: "List items"},
			{Name: "delete", Aliases: []string{"rm", "remove"}, Short: "Delete an item"},
			{Name: "get", Short: "Get an item"},
		},
	}

	output := captureStdout(t, func() {
		cmd.printHelp()
	})

	checks := []string{
		"list (ls)",
		"delete (rm, remove)",
		"get",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("help output missing %q\nGot:\n%s", check, output)
		}
	}

	// "get" should NOT have parentheses since it has no aliases
	if strings.Contains(output, "get (") {
		t.Errorf("get should not show alias parentheses\nGot:\n%s", output)
	}
}

// captureStdout redirects os.Stdout to a pipe, runs fn, and returns
// everything that was written to stdout as a string
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("could not read from pipe: %v", err)
	}
	return buf.String()
}

func TestExecuteNestedSubcommands(t *testing.T) {
	var gotArgs []string
	root := &command{
		Name: "root",
		SubCommands: []*command{
			{
				Name: "env",
				SubCommands: []*command{
					{
						Name: "set",
						Run: func(args []string) error {
							gotArgs = args
							return nil
						},
					},
				},
			},
		},
	}

	// root → env → set with remaining args
	if err := root.Execute([]string{"env", "set", "API_KEY", "--env", "prod"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotArgs) != 3 || gotArgs[0] != "API_KEY" {
		t.Errorf("expected [API_KEY --env prod], got %v", gotArgs)
	}

	// help at the middle level should not reach the leaf
	gotArgs = nil
	if err := root.Execute([]string{"env", "--help"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotArgs != nil {
		t.Error("set.Run should not be called when help is requested on parent")
	}
}

func TestPrintHelpIncludesExamples(t *testing.T) {
	cmd := &command{
		Name: "gql",
		Long: "GraphQL explorer",
		Examples: []*utils.CommandHelp{
			{Command: "hulak gql .", Description: "All files in current dir"},
		},
	}

	output := captureStdout(t, func() {
		cmd.printHelp()
	})

	checks := []string{"EXAMPLES", "hulak gql .", "All files in current dir"}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("help output missing %q\nGot:\n%s", check, output)
		}
	}
}

func TestFindSub(t *testing.T) {
	sub := &command{
		Name:    "gql",
		Aliases: []string{"graphql"},
	}
	root := &command{
		Name:        "root",
		SubCommands: []*command{sub},
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
