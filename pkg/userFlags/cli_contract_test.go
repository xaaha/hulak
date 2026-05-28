package userflags

import (
	"flag"
	"testing"
)

// TestSubCommandsExist verifies every expected subcommand is registered.
// If a subcommand is removed or renamed, this test fails.
func TestSubCommandsExist(t *testing.T) {
	root := subCommands()

	expected := []string{"run", "version", "init", "example", "migrate", "doctor", "gql", "secrets", "help"}
	for _, name := range expected {
		if root.FindSub(name) == nil {
			t.Errorf("expected subcommand %q to exist", name)
		}
	}
}

// TestGQLAliases verifies gql responds to all documented aliases.
func TestGQLAliases(t *testing.T) {
	root := subCommands()

	for _, alias := range []string{"gql", "graphql"} {
		if root.FindSub(alias) == nil {
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
	runCmd := root.FindSub("run")
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
	runCmd := root.FindSub("run")
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
	runCmd := root.FindSub("run")
	if runCmd == nil {
		t.Fatal("expected run subcommand to exist")
		return
	}
	if runCmd.Run == nil {
		t.Error("run subcommand is missing a Run handler")
	}
}

// TestParseRunArgsSetsFilePath verifies that passing a file path sets FilePath.

// TestEnvSubCommandsExist verifies every expected env subcommand is registered.
func TestEnvSubCommandsExist(t *testing.T) {
	root := subCommands()
	envCmd := root.FindSub("secrets")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	// Identity-related cmds moved under `secrets identity` — see
	// TestSecretsIdentitySurfaceSnapshot.
	expected := []string{"list", "keys", "edit", "identity"}
	for _, name := range expected {
		if envCmd.FindSub(name) == nil {
			t.Errorf("expected env subcommand %q to exist", name)
		}
	}
}

// TestEnvAliases verifies that all env subcommand aliases resolve correctly.
func TestEnvAliases(t *testing.T) {
	root := subCommands()
	envCmd := root.FindSub("secrets")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	tests := []struct {
		name    string
		aliases []string
	}{
		// When this list changes, also update the snapshot in TestSecretsSurfaceSnapshot.
		// `set`/`get`/`delete` no longer live at this level — they moved
		// under `secrets keys`. See TestSecretsKeysAliases for the nested
		// versions.
		{"list", []string{"ls"}},
		{"keys", []string{"key"}},
		{"sync", nil}, // intentionally no `rotate` alias — that name belongs to `identity rotate`
	}

	for _, tc := range tests {
		for _, alias := range append([]string{tc.name}, tc.aliases...) {
			t.Run(alias, func(t *testing.T) {
				if envCmd.FindSub(alias) == nil {
					t.Errorf("expected env subcommand %q to resolve (alias of %q)", alias, tc.name)
				}
			})
		}
	}
}

// TestSecretsKeysAliases verifies the nested `secrets keys` subgroup
// registers each leaf command with its documented alias.
func TestSecretsKeysAliases(t *testing.T) {
	root := subCommands()
	envCmd := root.FindSub("secrets")
	if envCmd == nil {
		t.Fatal("expected secrets subcommand to exist")
	}
	keys := envCmd.FindSub("keys")
	if keys == nil {
		t.Fatal("expected secrets keys subcommand to exist")
	}

	tests := []struct {
		name    string
		aliases []string
	}{
		{"list", []string{"ls"}},
		{"set", []string{"add"}},
		{"get", nil}, // no alias: --show flag (unmask values) would conflict
		{"delete", []string{"rm"}},
	}

	for _, tc := range tests {
		for _, alias := range append([]string{tc.name}, tc.aliases...) {
			t.Run(alias, func(t *testing.T) {
				if keys.FindSub(alias) == nil {
					t.Errorf("expected secrets keys subcommand %q to resolve (alias of %q)", alias, tc.name)
				}
			})
		}
	}
}

// TestSecretsKeysSubCommandsHaveEnvFlag verifies every leaf of `secrets keys`
// registers --env / --environment so users can target a specific environment
// or fall through to the picker.
func TestSecretsKeysSubCommandsHaveEnvFlag(t *testing.T) {
	root := subCommands()
	envCmd := root.FindSub("secrets")
	if envCmd == nil {
		t.Fatal("expected secrets subcommand to exist")
	}
	keys := envCmd.FindSub("keys")
	if keys == nil {
		t.Fatal("expected secrets keys subcommand to exist")
	}

	for _, leaf := range []string{"list", "set", "get", "delete"} {
		sub := keys.FindSub(leaf)
		if sub == nil {
			t.Errorf("expected secrets keys %q to exist", leaf)
			continue
		}
		if sub.Flags == nil {
			t.Errorf("secrets keys %q should have its own FlagSet", leaf)
			continue
		}
		if sub.Flags.Lookup("env") == nil {
			t.Errorf("secrets keys %q should have an --env flag", leaf)
		}
		if sub.Flags.Lookup("environment") == nil {
			t.Errorf("secrets keys %q should have an --environment alias", leaf)
		}
	}
}

// TestEnvSubcommandAliasesUnique guards against accidentally registering the
// same name or alias under two env subcommands. findSub does first-match wins,
// so a duplicate would silently shadow whichever command is later in the slice.
func TestEnvSubcommandAliasesUnique(t *testing.T) {
	root := subCommands()
	envCmd := root.FindSub("secrets")
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
	envCmd := root.FindSub("secrets")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	// Subcommands that target a specific environment
	// list does not take --env (it lists environment names themselves)
	needsEnvFlag := []string{"keys", "edit"}
	for _, name := range needsEnvFlag {
		sub := envCmd.FindSub(name)
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
	envCmd := root.FindSub("secrets")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	// list does not take --env (it lists environment names themselves)
	needsEnvFlag := []string{"keys", "edit"}
	for _, name := range needsEnvFlag {
		sub := envCmd.FindSub(name)
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
	envCmd := root.FindSub("secrets")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
	}

	tests := []struct {
		subcommand string
		flag       string
	}{
		{"keys", "show"},
		{"keys", "search"},
	}

	for _, tc := range tests {
		sub := envCmd.FindSub(tc.subcommand)
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

	// Identity subgroup leaves with their own contract-critical flags.
	identity := envCmd.FindSub("identity")
	if identity == nil {
		t.Fatal("expected secrets identity subgroup to exist")
	}
	identityTests := []struct {
		leaf string
		flag string
	}{
		{"import", "stdin"},
		{"import", "force"},
		{"export", "out"},
	}
	for _, tc := range identityTests {
		sub := identity.FindSub(tc.leaf)
		if sub == nil {
			t.Errorf("expected secrets identity %q to exist", tc.leaf)
			continue
		}
		if sub.Flags == nil {
			t.Errorf("secrets identity %q should have a FlagSet", tc.leaf)
			continue
		}
		if sub.Flags.Lookup(tc.flag) == nil {
			t.Errorf("secrets identity %q should have a --%s flag", tc.leaf, tc.flag)
		}
	}
}

// TestEnvSubCommandsHaveRunHandlers verifies every leaf env subcommand has a
// non-nil Run handler so dispatch doesn't silently fall through to help.
// Subgroups (commands with SubCommands of their own, like `keys` and
// `identity`) intentionally have no Run handler — they delegate to their
// children, or to a fallback Run when the leaf is omitted.
func TestEnvSubCommandsHaveRunHandlers(t *testing.T) {
	root := subCommands()
	envCmd := root.FindSub("secrets")
	if envCmd == nil {
		t.Fatal("expected env subcommand to exist")
		return
	}

	for _, sub := range envCmd.SubCommands {
		// Subgroups may legitimately have a nil Run.
		if len(sub.SubCommands) > 0 {
			continue
		}
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
	if root.FindSub("secrets") == nil {
		t.Error("expected 'secrets' to resolve as a subcommand")
	}

	// "env" should resolve as an alias of "secrets" (#201)
	if root.FindSub("env") == nil {
		t.Error("expected 'env' to resolve as an alias of 'secrets'")
	}

	// "-env" should NOT resolve as a subcommand (it's a flag)
	if root.FindSub("-env") != nil {
		t.Error("'-env' should not resolve as a subcommand")
	}
}

// TestSecretsHasEnvAlias verifies that legacy `hulak env ...` invocations
// resolve to the renamed `secrets` command. The alias was promised in #201
// to keep older docs and muscle memory working.
func TestSecretsHasEnvAlias(t *testing.T) {
	root := subCommands()
	envResolved := root.FindSub("env")
	if envResolved == nil {
		t.Fatal("expected 'env' to resolve to 'secrets' command")
	}
	if envResolved != root.FindSub("secrets") {
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

// TestSecretsRotateBelongsToIdentity verifies that `hulak secrets rotate`
// resolves only as `secrets identity rotate` — there is no env-level `rotate`
// alias of `sync`. The two operations are not interchangeable: `sync` re-encrypts
// without changing keys, `identity rotate` issues a new keypair. Aliasing them
// would let `hulak secrets rotate` silently do the safe-but-wrong thing for a
// user trying to respond to a key compromise.
func TestSecretsRotateBelongsToIdentity(t *testing.T) {
	root := subCommands()
	secrets := root.FindSub("secrets")
	if secrets == nil {
		t.Fatal("expected secrets subcommand to exist")
	}

	// At the env level, `rotate` must not resolve.
	if got := secrets.FindSub("rotate"); got != nil {
		t.Errorf("`secrets rotate` should not resolve at the env level (resolved to %q); only `secrets identity rotate` should exist", got.Name)
	}

	// At the identity level, `rotate` must resolve.
	identity := secrets.FindSub("identity")
	if identity == nil {
		t.Fatal("expected secrets identity subgroup to exist")
	}
	if identity.FindSub("rotate") == nil {
		t.Error("`secrets identity rotate` should resolve")
	}
}

// TestFindSubcommandIndex_FlagBeforeVerb verifies that dispatch skips past
// leading flags when locating a subcommand. Without this, `secrets keys
// --env prod list` would land in the parent's Run handler with `list` as
// a stray positional. The leaf-level flag-anywhere parser in Execute makes
// flag ordering insignificant; this test holds dispatch to the same contract.
func TestFindSubcommandIndex_FlagBeforeVerb(t *testing.T) {
	root := subCommands()
	keys := root.FindSub("secrets").FindSub("keys")
	if keys == nil {
		t.Fatal("secrets keys subcommand missing")
	}

	tests := []struct {
		name         string
		args         []string
		wantMatchIdx int
		wantFirstPos int
	}{
		{"verb first", []string{"list"}, 0, 0},
		{"verb after string flag (space form)", []string{"--env", "prod", "list"}, 2, 2},
		{"verb after string flag (inline form)", []string{"--env=prod", "list"}, 1, 1},
		{"verb after bool flag", []string{"--show", "list"}, 1, 1},
		{"verb after multiple flags", []string{"--env", "prod", "--show", "list"}, 3, 3},
		{"all flags, no verb", []string{"--env", "prod"}, -1, -1},
		{"empty args", nil, -1, -1},
		{"typo after flag", []string{"--env", "prod", "ghost"}, -1, 2},
		{"alias resolves", []string{"--env", "prod", "ls"}, 2, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotMatch, gotFirst := keys.FindSubcommandIndex(tc.args)
			if gotMatch != tc.wantMatchIdx {
				t.Errorf("matchIdx = %d, want %d", gotMatch, tc.wantMatchIdx)
			}
			if gotFirst != tc.wantFirstPos {
				t.Errorf("firstNonFlag = %d, want %d", gotFirst, tc.wantFirstPos)
			}
		})
	}
}
