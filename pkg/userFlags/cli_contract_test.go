package userflags

import (
	"flag"
	"testing"
)

// TestSubCommandsExist verifies every expected subcommand is registered.
// If a subcommand is removed or renamed, this test fails.
func TestSubCommandsExist(t *testing.T) {
	root := subCommands()

	expected := []string{"version", "init", "migrate", "doctor", "gql", "help"}
	for _, name := range expected {
		if root.findSub(name) == nil {
			t.Errorf("expected subcommand %q to exist", name)
		}
	}
}

// TestGQLAliases verifies gql responds to all documented aliases.
func TestGQLAliases(t *testing.T) {
	root := subCommands()

	for _, alias := range []string{"gql", "graphql", "GraphQL"} {
		if root.findSub(alias) == nil {
			t.Errorf("expected gql alias %q to resolve", alias)
		}
	}
}

// TestGlobalFlagsRegistered verifies every expected global flag is
// registered on flag.CommandLine. Removing a flag breaks the CLI contract.
func TestGlobalFlagsRegistered(t *testing.T) {
	expected := []string{
		"env", "fp", "file-path", "f", "file",
		"debug", "dir", "dirseq",
		"v", "version", "h", "help",
	}

	for _, name := range expected {
		if flag.Lookup(name) == nil {
			t.Errorf("expected global flag %q to be registered", name)
		}
	}
}

// TestFlagAliasesShareVariable verifies that short and long flag forms
// write to the same variable. Without this, -fp and --file-path would
// silently diverge.
func TestFlagAliasesShareVariable(t *testing.T) {
	aliases := []struct {
		short string
		long  string
	}{
		{"fp", "file-path"},
		{"f", "file"},
		{"v", "version"},
		{"h", "help"},
	}

	for _, a := range aliases {
		short := flag.Lookup(a.short)
		long := flag.Lookup(a.long)
		if short == nil || long == nil {
			t.Errorf("flag %q or %q not registered", a.short, a.long)
			continue
		}

		// Set via the short form and verify the long form sees it
		if err := short.Value.Set("test-value"); err != nil {
			// bool flags reject "test-value", so use "true" for bools
			if err := short.Value.Set("true"); err != nil {
				t.Errorf("could not set flag %q: %v", a.short, err)
				continue
			}
		}

		if short.Value.String() != long.Value.String() {
			t.Errorf(
				"flags -%s and --%s are not aliased: short=%q, long=%q",
				a.short, a.long, short.Value.String(), long.Value.String(),
			)
		}
	}
}

// TestSubCommandsHaveHelp verifies every subcommand has at minimum
// a Short description (shows in parent help) and a Long description
// (shows with --help).
func TestSubCommandsHaveHelp(t *testing.T) {
	root := subCommands()

	for _, sub := range root.SubCommands {
		if sub.Short == "" {
			t.Errorf("subcommand %q is missing Short description", sub.Name)
		}
		// help subcommand delegates to root.printHelp, so Long is optional
		if sub.Name != "help" && sub.Long == "" {
			t.Errorf("subcommand %q is missing Long description", sub.Name)
		}
	}
}
