package mcpcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	cmd := New("v1.2.3")
	if cmd.Name != "mcp" {
		t.Errorf("Name = %q, want mcp", cmd.Name)
	}
	for _, f := range []string{"project", "default-project"} {
		if cmd.Flags.Lookup(f) == nil {
			t.Errorf("expected a --%s flag", f)
		}
	}
	if cmd.Run == nil {
		t.Error("Run must be set")
	}
}

func TestProjectMap_Set(t *testing.T) {
	t.Run("parses name=path", func(t *testing.T) {
		p := projectMap{}
		if err := p.Set("api=~/work/api"); err != nil {
			t.Fatal(err)
		}
		if p["api"] != "~/work/api" {
			t.Errorf("got %q", p["api"])
		}
	})
	t.Run("rejects bad format", func(t *testing.T) {
		p := projectMap{}
		for _, bad := range []string{"noequals", "=nopath", "noname="} {
			if err := p.Set(bad); err == nil {
				t.Errorf("Set(%q) should error", bad)
			}
		}
	})
	t.Run("rejects duplicate", func(t *testing.T) {
		p := projectMap{}
		_ = p.Set("api=/one")
		if err := p.Set("api=/two"); err == nil {
			t.Error("duplicate project should error")
		}
	})
}

func TestRequireIdentity_NonVaultOK(t *testing.T) {
	// Plain dirs, no .hulak/store.age => no identity needed.
	err := requireIdentity(map[string]string{"api": t.TempDir(), "mob": t.TempDir()})
	if err != nil {
		t.Errorf("non-vault projects should not require identity, got: %v", err)
	}
}

func TestRequireIdentity_AgeStoreNoIdentityFails(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, ".hulak"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".hulak", "store.age"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Neutralize every identity source: no env keys; HOME and XDG_CONFIG_HOME
	// point at empty dirs so identity.txt and ~/.ssh/id_ed25519 are absent.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HULAK_MASTER_KEY", "")
	t.Setenv("HULAK_SSH_IDENTITY", "")

	err := requireIdentity(map[string]string{"vaulted": tmp})
	if err == nil {
		t.Fatal("age store with no identity should fail")
	}
	if !strings.Contains(err.Error(), "no identity is configured") {
		t.Errorf("error should explain missing identity, got: %v", err)
	}
}
