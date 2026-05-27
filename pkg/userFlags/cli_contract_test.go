package userflags

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSubCommandsExist verifies every expected subcommand is registered.
// If a subcommand is removed or renamed, this test fails.
func TestSubCommandsExist(t *testing.T) {
	root := subCommands()

	expected := []string{"run", "version", "init", "migrate", "doctor", "gql", "secrets", "help"}
	for _, name := range expected {
		if root.findSub(name) == nil {
			t.Errorf("expected subcommand %q to exist", name)
		}
	}
}

// TestGQLAliases verifies gql responds to all documented aliases.
func TestGQLAliases(t *testing.T) {
	root := subCommands()

	for _, alias := range []string{"gql", "graphql"} {
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
		"dry-run", "show",
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

	for _, name := range []string{"env", "environment", "sequential", "seq", "debug", "dry-run", "show"} {
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
		return
	}
	if runCmd.Run == nil {
		t.Error("run subcommand is missing a Run handler")
	}
}

// TestParseRunArgsSetsFilePath verifies that passing a file path sets FilePath.
func TestParseRunArgsSetsFilePath(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{Args: []string{tmpFile}})
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
	tmpDir := t.TempDir()

	f, err := parseRunArgs(runCmdArgs{Args: []string{tmpDir}})
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

// TestParseRunArgsRoutesFlags verifies that parsed flag values map to the
// right runner.Flags fields. Flag parsing itself is the framework's job —
// this test only checks the path → Flags translation.
func TestParseRunArgsRoutesFlags(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{Env: "staging", Debug: true, Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !f.Debug {
		t.Error("Debug should mirror the debug arg")
	}
	if f.Env != "staging" {
		t.Errorf("Env = %q, want %q", f.Env, "staging")
	}
	if !f.EnvSet {
		t.Error("EnvSet should be true when an env is provided")
	}
}

// TestParseRunArgsQuietPlumbed verifies quiet=true lands on runner.Flags.
func TestParseRunArgsQuietPlumbed(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{Quiet: true, Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Quiet {
		t.Error("Quiet should be true when quiet arg is true")
	}
}

// TestParseRunArgsDryRunPlumbed verifies DryRun=true lands on runner.Flags.
func TestParseRunArgsDryRunPlumbed(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{DryRun: true, Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.DryRun {
		t.Error("DryRun should be true when DryRun arg is true")
	}
}

// TestParseRunArgsShowPlumbed verifies Show=true lands on runner.Flags.
func TestParseRunArgsShowPlumbed(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{Show: true, Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Show {
		t.Error("Show should be true when Show arg is true")
	}
}

// TestParseRunArgsDryRunAndShowPlumbed verifies both flags set together
// land on runner.Flags so --dry-run --show reveals headers as intended.
func TestParseRunArgsDryRunAndShowPlumbed(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{DryRun: true, Show: true, Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.DryRun {
		t.Error("DryRun should be true")
	}
	if !f.Show {
		t.Error("Show should be true")
	}
}

// TestParseRunArgsSequentialDir verifies sequential=true on a directory
// routes to Dirseq instead of Dir.
func TestParseRunArgsSequentialDir(t *testing.T) {
	tmpDir := t.TempDir()

	f, err := parseRunArgs(runCmdArgs{Sequential: true, Args: []string{tmpDir}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.Dirseq != tmpDir {
		t.Errorf("Dirseq = %q, want %q", f.Dirseq, tmpDir)
	}
	if f.Dir != "" {
		t.Error("Dir should be empty when sequential is true")
	}
}

// TestParseRunArgsTimeoutPlumbed verifies the --timeout flag value lands on
// runner.Flags so the runner can resolve it. Precedence/fallback is asserted
// in runner.TestResolveBaseTimeout.
func TestParseRunArgsTimeoutPlumbed(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	want := 5 * time.Minute
	f, err := parseRunArgs(runCmdArgs{Timeout: want, Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Timeout != want {
		t.Errorf("Timeout = %v, want %v", f.Timeout, want)
	}
}

// TestParseRunArgsBadPath verifies that a nonexistent path returns an error.
func TestParseRunArgsBadPath(t *testing.T) {
	_, err := parseRunArgs(runCmdArgs{Args: []string{"/nonexistent/path.yaml"}})
	if err == nil {
		t.Fatal("expected an error for nonexistent path")
	}
}

// TestParseRunArgsNoEnv verifies that omitting --env leaves Env empty and
// EnvSet false, so the runner knows to invoke the picker instead of silently
// defaulting to "global".
func TestParseRunArgsNoEnv(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.hk.yaml")
	if err := os.WriteFile(tmpFile, []byte("kind: API"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := parseRunArgs(runCmdArgs{Args: []string{tmpFile}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.Env != "" {
		t.Errorf("Env = %q, want empty so the runner invokes the picker", f.Env)
	}
	if f.EnvSet {
		t.Error("EnvSet should be false when no env is provided")
	}
}

// TestEnvSubCommandsExist verifies every expected env subcommand is registered.
func TestEnvSubCommandsExist(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("secrets")
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
	envCmd := root.findSub("secrets")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	tests := []struct {
		name    string
		aliases []string
	}{
		// When this list changes, also update the snapshot in TestSecretsSurfaceSnapshot.
		{"list", []string{"ls"}},
		{"set", []string{"add"}},
		{"get", nil}, // no alias: --show flag (unmask values) would conflict
		{"keys", []string{"key"}},
		{"delete", []string{"rm"}},
		{"sync", []string{"rotate"}},
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

// TestEnvSubcommandAliasesUnique guards against accidentally registering the
// same name or alias under two env subcommands. findSub does first-match wins,
// so a duplicate would silently shadow whichever command is later in the slice.
func TestEnvSubcommandAliasesUnique(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("secrets")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
		return
	}

	seen := map[string]string{} // identifier → owner name
	for _, sub := range envCmd.SubCommands {
		ids := append([]string{sub.Name}, sub.Aliases...)
		for _, id := range ids {
			if owner, dup := seen[id]; dup {
				t.Errorf("identifier %q is claimed by both %q and %q", id, owner, sub.Name)
				continue
			}
			seen[id] = sub.Name
		}
	}
}

// TestEnvSubCommandsHaveFlags verifies env subcommands that operate on a
// specific environment have an --env flag in their FlagSet.
func TestEnvSubCommandsHaveFlags(t *testing.T) {
	root := subCommands()
	envCmd := root.findSub("secrets")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	// Subcommands that target a specific environment
	// list does not take --env (it lists environment names themselves)
	needsEnvFlag := []string{"set", "get", "keys", "delete", "edit"}
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
	envCmd := root.findSub("secrets")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	// list does not take --env (it lists environment names themselves)
	needsEnvFlag := []string{"set", "get", "keys", "delete", "edit"}
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
	envCmd := root.findSub("secrets")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	tests := []struct {
		subcommand string
		flag       string
	}{
		{"set", "stdin"},
		{"keys", "show"},
		{"keys", "search"},
		{"import-key", "stdin"},
		{"import-key", "force"},
		{"export-key", "out"},
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
	envCmd := root.findSub("secrets")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
		return
	}

	for _, sub := range envCmd.SubCommands {
		if sub.Run == nil {
			t.Errorf("env subcommand %q is missing a Run handler", sub.Name)
		}
	}
}

// TestEnvFlagDoesNotConflictWithSecretsSubcommand verifies that the -env global
// flag and the secrets subcommand coexist. The flag is prefixed with "-" so the
// router dispatches them to different paths. The "env" alias on the secrets
// command is preserved per #201.
func TestEnvFlagDoesNotConflictWithSecretsSubcommand(t *testing.T) {
	root := subCommands()

	// "secrets" should resolve as a subcommand
	if root.findSub("secrets") == nil {
		t.Error("expected 'secrets' to resolve as a subcommand")
	}

	// "env" should resolve as an alias of "secrets" (#201)
	if root.findSub("env") == nil {
		t.Error("expected 'env' to resolve as an alias of 'secrets'")
	}

	// "-env" should NOT resolve as a subcommand (it's a flag)
	if root.findSub("-env") != nil {
		t.Error("'-env' should not resolve as a subcommand")
	}
}

// TestSecretsHasEnvAlias verifies that legacy `hulak env ...` invocations
// resolve to the renamed `secrets` command. The alias was promised in #201
// to keep older docs and muscle memory working.
func TestSecretsHasEnvAlias(t *testing.T) {
	root := subCommands()
	envResolved := root.findSub("env")
	if envResolved == nil {
		t.Fatal("expected 'env' to resolve to 'secrets' command")
	}
	if envResolved != root.findSub("secrets") {
		t.Fatal("'env' should resolve to the same command as 'secrets'")
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
