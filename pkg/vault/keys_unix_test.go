//go:build !windows

package vault

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
)

// TestSetIdentityCreatesDirOwnerOnly verifies that when SetIdentity has to
// MkdirAll the parent directory itself (first-use bootstrap), it creates the
// dir owner-only (0700). Group/other access on a shared host would let other
// users `ls` the dir and learn an identity file exists, even if they can't
// read it — defense in depth around the most sensitive file in the system.
//
// We zero the umask so the assertion catches a regression where SetIdentity
// passes a wider mode (e.g. 0755) and the user's umask is what actually
// trims it down to 0700 in practice.
func TestSetIdentityCreatesDirOwnerOnly(t *testing.T) {
	xdg := t.TempDir()
	xdg, err := filepath.EvalSymlinks(xdg)
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", xdg)

	oldMask := syscall.Umask(0)
	t.Cleanup(func() { syscall.Umask(oldMask) })

	id, _ := age.GenerateX25519Identity()
	if err := SetIdentity(id.String()); err != nil {
		t.Fatalf("SetIdentity: %v", err)
	}

	configDir, err := utils.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("stat config dir: %v", err)
	}
	if got := info.Mode().Perm(); got != utils.SecretDirPer {
		t.Errorf(
			"config dir permissions = %o, want %o (identity dir must be owner-only)",
			got, utils.SecretDirPer,
		)
	}
}
