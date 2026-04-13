package userflags

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
)

// newTestRunFlagSet creates a fresh FlagSet with the same flags as the run
// subcommand, returning the pointers tests need to pass to parseRunArgs.
func newTestRunFlagSet() (fs *flag.FlagSet, envVal *string, seq *bool, debug *bool) {
	fs = flag.NewFlagSet("run", flag.ContinueOnError)
	fs.Usage = func() {}
	fs.SetOutput(io.Discard)
	envVal = registerEnvFlag(fs, "", "Environment to use")
	var s, d bool
	fs.BoolVar(&s, "sequential", false, "")
	fs.BoolVar(&s, "seq", false, "")
	fs.BoolVar(&d, "debug", false, "")
	return fs, envVal, &s, &d
}

// TestSubCommandsExist verifies every expected subcommand is registered.
// If a subcommand is removed or renamed, this test fails.
func TestSubCommandsExist(t *testing.T) {
	root := subCommands()

	expected := []string{"run", "version", "init", "migrate", "doctor", "gql", "env", "help"}
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

// TestRunSubcommandFlags verifies the run subcommand has all expected flags.
func TestRunSubcommandFlags(t *testing.T) {
	root := subCommands()
	runCmd := root.findSub("run")
	if runCmd == nil {
		t.Fatal("expected run subcommand to exist")
	}

	for _, name := range []string{"env", "environment", "sequential", "seq", "debug"} {
		if runCmd.Flags.Lookup(name) == nil {
			t.Errorf("run subcommand should have --%s flag", name)
		}
	}
}

// TestRunFlagAliasesShareVariable verifies that run's flag aliases
// point to the same underlying variable.
func TestRunFlagAliasesShareVariable(t *testing.T) {
	root := subCommands()
	runCmd := root.findSub("run")
	if runCmd == nil {
		t.Fatal("expected run subcommand to exist")
	}

	tests := []struct {
		short string
		long  string
		val   string // test value to set (must be valid for the flag type)
	}{
		{"env", "environment", "alias-test"},
		{"seq", "sequential", "true"},
	}

	for _, tc := range tests {
		t.Run(tc.short+"/"+tc.long, func(t *testing.T) {
			short := runCmd.Flags.Lookup(tc.short)
			long := runCmd.Flags.Lookup(tc.long)
			if short == nil || long == nil {
				t.Fatalf("missing flag: short=%v long=%v", short, long)
			}

			if err := runCmd.Flags.Set(tc.long, tc.val); err != nil {
				t.Fatalf("failed to set --%s: %v", tc.long, err)
			}
			if short.Value.String() != tc.val {
				t.Errorf("--%s = %q, want %q (should share variable with --%s)",
					tc.short, short.Value.String(), tc.val, tc.long)
			}
		})
	}
}

// TestRunSubcommandHasRunHandler verifies the run subcommand has a Run handler.
func TestRunSubcommandHasRunHandler(t *testing.T) {
	root := subCommands()
	runCmd := root.findSub("run")
	if runCmd == nil {
		t.Fatal("expected run subcommand to exist")
	}
	if runCmd.Run == nil {
		t.Error("run subcommand is missing a Run handler")
	}
}

// TestParseRunArgsSetsFilePath verifies that passing a file path sets FilePath.
func TestParseRunArgsSetsFilePath(t *testing.T) {
	fs, envVal, seq, debug := newTestRunFlagSet()

	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(fs, envVal, seq, debug, []string{tmpFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.FilePath != tmpFile {
		t.Errorf("FilePath = %q, want %q", f.FilePath, tmpFile)
	}
	if f.Dir != "" || f.Dirseq != "" {
		t.Error("Dir/Dirseq should be empty for a file path")
	}
}

// TestParseRunArgsSetsDir verifies that passing a directory sets Dir (concurrent).
func TestParseRunArgsSetsDir(t *testing.T) {
	fs, envVal, seq, debug := newTestRunFlagSet()
	tmpDir := t.TempDir()

	f, err := parseRunArgs(fs, envVal, seq, debug, []string{tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.Dir != tmpDir {
		t.Errorf("Dir = %q, want %q", f.Dir, tmpDir)
	}
	if f.Dirseq != "" {
		t.Error("Dirseq should be empty without --sequential")
	}
	if f.FilePath != "" {
		t.Error("FilePath should be empty for a directory")
	}
}

// TestParseRunArgsTrailingFlags verifies that flags after the positional
// argument are still parsed correctly.
func TestParseRunArgsTrailingFlags(t *testing.T) {
	fs, envVal, seq, debug := newTestRunFlagSet()

	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(fs, envVal, seq, debug, []string{tmpFile, "--debug", "--env", "staging"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !f.Debug {
		t.Error("Debug should be true when --debug is passed after the path")
	}
	if f.Env != "staging" {
		t.Errorf("Env = %q, want %q", f.Env, "staging")
	}
	if !f.EnvSet {
		t.Error("EnvSet should be true when --env is provided")
	}
}

// TestParseRunArgsSequentialDir verifies --sequential after a directory
// sets Dirseq instead of Dir.
func TestParseRunArgsSequentialDir(t *testing.T) {
	fs, envVal, seq, debug := newTestRunFlagSet()
	tmpDir := t.TempDir()

	f, err := parseRunArgs(fs, envVal, seq, debug, []string{tmpDir, "--sequential"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.Dirseq != tmpDir {
		t.Errorf("Dirseq = %q, want %q", f.Dirseq, tmpDir)
	}
	if f.Dir != "" {
		t.Error("Dir should be empty when --sequential is set")
	}
}

// TestParseRunArgsBadPath verifies that a nonexistent path returns an error.
func TestParseRunArgsBadPath(t *testing.T) {
	fs, envVal, seq, debug := newTestRunFlagSet()

	_, err := parseRunArgs(fs, envVal, seq, debug, []string{"/nonexistent/path.yaml"})
	if err == nil {
		t.Fatal("expected an error for nonexistent path")
	}
}

// TestParseRunArgsUnknownTrailingFlag verifies that an unknown flag after
// the path produces a parse error, not a silent pass.
func TestParseRunArgsUnknownTrailingFlag(t *testing.T) {
	fs, envVal, seq, debug := newTestRunFlagSet()

	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := parseRunArgs(fs, envVal, seq, debug, []string{tmpFile, "--bogus"})
	if err == nil {
		t.Fatal("expected an error for unknown trailing flag")
	}
}

// TestParseRunArgsDefaultEnv verifies that when no --env is passed,
// the default environment is used.
func TestParseRunArgsDefaultEnv(t *testing.T) {
	fs, envVal, seq, debug := newTestRunFlagSet()

	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(fs, envVal, seq, debug, []string{tmpFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.Env != utils.DefaultEnvVal {
		t.Errorf("Env = %q, want default %q", f.Env, utils.DefaultEnvVal)
	}
	if f.EnvSet {
		t.Error("EnvSet should be false when no --env is provided")
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

// TestEnvAliases verifies that all env subcommand aliases resolve correctly.
func TestEnvAliases(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("env")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	tests := []struct {
		name    string
		aliases []string
	}{
		{"list", []string{"ls"}},
		{"set", []string{"add"}},
		{"keys", []string{"key"}},
		{"delete", []string{"rm", "remove"}},
	}

	for _, tc := range tests {
		for _, alias := range append([]string{tc.name}, tc.aliases...) {
			t.Run(alias, func(t *testing.T) {
				if envCmd.findSub(alias) == nil {
					t.Errorf("expected env subcommand %q to resolve (alias of %q)", alias, tc.name)
				}
			})
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
