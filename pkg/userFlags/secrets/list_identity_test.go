package secrets

import (
	"testing"

	"filippo.io/age"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

func TestRunListIdentity(t *testing.T) {
	t.Run("requires a vault project", func(t *testing.T) {
		// No vault setup → requireVaultProject should fail
		t.Setenv("HULAK_MASTER_KEY", "")
		t.Setenv("HULAK_SSH_IDENTITY", "")
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		err := runListIdentity(nil)
		if err == nil {
			t.Fatal("expected error outside a vault project")
		}
	})

	t.Run("rejects positional arguments", func(t *testing.T) {
		err := runListIdentity([]string{"unexpected"})
		if err == nil {
			t.Fatal("expected error for unexpected argument")
		}
	})

	t.Run("lists decrypting identity inside a vault", func(t *testing.T) {
		// Build a vault encrypted to a known age identity, then verify
		// list-identity finds it.
		recipientID, _ := age.GenerateX25519Identity()
		setupImportKeyTest(t, true, recipientID)

		// Make identity.txt match the recipient so it decrypts
		if err := vault.SetIdentity(recipientID.String()); err != nil {
			t.Fatal(err)
		}

		if err := runListIdentity(nil); err != nil {
			t.Fatalf("runListIdentity: %v", err)
		}
	})

	t.Run("errors when no identity decrypts", func(t *testing.T) {
		recipientID, _ := age.GenerateX25519Identity()
		setupImportKeyTest(t, true, recipientID)

		// Plant a stranger key — won't decrypt
		stranger, _ := age.GenerateX25519Identity()
		if err := vault.SetIdentity(stranger.String()); err != nil {
			t.Fatal(err)
		}

		err := runListIdentity(nil)
		if err == nil {
			t.Fatal("expected error when no identity decrypts")
		}
	})
}

// sanity-check that the constant we use for the default marker is non-empty
// so the table cell isn't accidentally blank when an identity is the default.
func TestAsteriskMarkerNonEmpty(t *testing.T) {
	if utils.Asterisk == "" {
		t.Fatal("utils.Asterisk must be non-empty for the default-identity marker")
	}
}
