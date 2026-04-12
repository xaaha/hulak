package userflags

import (
	"flag"
	"testing"
)

// TestSubCommandsExist verifies every expected subcommand is registered.
// If a subcommand is removed or renamed, this test fails.
func TestSubCommandsExist(t *testing.T) {
	root := subCommands()

	expected := []string{"version", "init", "migrate", "doctor", "gql", "env", "help"}
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
		"env", "environment", "fp", "file-path", "f", "file",
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
		{"env", "environment"},
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

		// Save original values to restore after test
		origShort := short.Value.String()
		origLong := long.Value.String()

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

		// Restore original values to avoid leaking state to other tests
		_ = short.Value.Set(origShort)
		_ = long.Value.Set(origLong)
	}
}

// TestEnvSubCommandsExist verifies every expected env subcommand is registered.
func TestEnvSubCommandsExist(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("env")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	expected := []string{
		"set", "get", "list", "keys", "delete", "edit",
		"import-key", "export-key",
		"add-recipient", "remove-recipient", "list-recipients",
	}
	for _, name := range expected {
		if envCmd.findSub(name) == nil {
			t.Errorf("expected env subcommand %q to exist", name)
		}
	}
}

// TestEnvListAlias verifies "ls" resolves to the "list" subcommand.
func TestEnvListAlias(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("env")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	for _, name := range []string{"list", "ls"} {
		if envCmd.findSub(name) == nil {
			t.Errorf("expected env subcommand %q to resolve", name)
		}
	}
}

// TestEnvSetAlias verifies "add" resolves to the "set" subcommand.
func TestEnvSetAlias(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("env")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	for _, name := range []string{"set", "add"} {
		if envCmd.findSub(name) == nil {
			t.Errorf("expected env subcommand %q to resolve", name)
		}
	}
}

// TestEnvKeysAlias verifies "key" resolves to the "keys" subcommand.
func TestEnvKeysAlias(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("env")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	for _, name := range []string{"keys", "key"} {
		if envCmd.findSub(name) == nil {
			t.Errorf("expected env subcommand %q to resolve", name)
		}
	}
}

// TestEnvDeleteAlias verifies "rm" and "remove" resolve to the "delete" subcommand.
func TestEnvDeleteAlias(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("env")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	for _, name := range []string{"delete", "rm", "remove"} {
		if envCmd.findSub(name) == nil {
			t.Errorf("expected env subcommand %q to resolve", name)
		}
	}
}

// TestEnvSubCommandsHaveFlags verifies env subcommands that operate on a
// specific environment have an --env flag in their FlagSet.
func TestEnvSubCommandsHaveFlags(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("env")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	// Subcommands that target a specific environment
	needsEnvFlag := []string{"set", "get", "list", "keys", "delete", "edit"}
	for _, name := range needsEnvFlag {
		sub := envCmd.findSub(name)
		if sub == nil {
			t.Errorf("expected env subcommand %q to exist", name)
			continue
		}
		if sub.Flags == nil {
			t.Errorf("env subcommand %q should have its own FlagSet", name)
			continue
		}
		if sub.Flags.Lookup("env") == nil {
			t.Errorf("env subcommand %q should have an --env flag", name)
		}
	}
}

// TestEnvSubCommandsHaveEnvironmentAlias verifies that env subcommands
// accepting --env also accept --environment.
func TestEnvSubCommandsHaveEnvironmentAlias(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("env")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	needsEnvFlag := []string{"set", "get", "list", "keys", "delete", "edit"}
	for _, name := range needsEnvFlag {
		sub := envCmd.findSub(name)
		if sub == nil {
			t.Errorf("expected env subcommand %q to exist", name)
			continue
		}
		if sub.Flags == nil {
			t.Errorf("env subcommand %q should have its own FlagSet", name)
			continue
		}
		if sub.Flags.Lookup("environment") == nil {
			t.Errorf("env subcommand %q should have an --environment alias", name)
		}
	}
}

// TestEnvSubCommandSpecificFlags verifies subcommand-specific flags
// that are part of the CLI contract don't get accidentally removed.
func TestEnvSubCommandSpecificFlags(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("env")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	tests := []struct {
		subcommand string
		flag       string
	}{
		{"set", "stdin"},
		{"keys", "show"},
		{"import-key", "stdin"},
		{"export-key", "armor"},
	}

	for _, tc := range tests {
		sub := envCmd.findSub(tc.subcommand)
		if sub == nil {
			t.Errorf("expected env subcommand %q to exist", tc.subcommand)
			continue
		}
		if sub.Flags == nil {
			t.Errorf("env subcommand %q should have a FlagSet", tc.subcommand)
			continue
		}
		if sub.Flags.Lookup(tc.flag) == nil {
			t.Errorf("env subcommand %q should have a --%s flag", tc.subcommand, tc.flag)
		}
	}
}

// TestEnvSubCommandsHaveRunHandlers verifies every env subcommand has a
// non-nil Run handler so dispatch doesn't silently fall through to help.
func TestEnvSubCommandsHaveRunHandlers(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("env")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	for _, sub := range envCmd.SubCommands {
		if sub.Run == nil {
			t.Errorf("env subcommand %q is missing a Run handler", sub.Name)
		}
	}
}

// TestEnvFlagDoesNotConflictWithEnvSubcommand verifies that the -env global
// flag and the env subcommand coexist. The flag is prefixed with "-" so the
// router dispatches them to different paths.
func TestEnvFlagDoesNotConflictWithEnvSubcommand(t *testing.T) {
	root := subCommands()

	// "env" (no dash) should resolve as a subcommand
	if root.findSub("env") == nil {
		t.Error("expected 'env' to resolve as a subcommand")
	}

	// "-env" should NOT resolve as a subcommand (it's a flag)
	if root.findSub("-env") != nil {
		t.Error("'-env' should not resolve as a subcommand")
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
