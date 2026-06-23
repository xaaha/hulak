package vault

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xaaha/hulak/pkg/utils"
)

// BootstrapResult holds the output of BootstrapVault for both age and SSH flows.
// Exported so callers in initcmd/ and secrets/ can read summary fields without
// re-deriving them from the underlying identity / store.
type BootstrapResult struct {
	RecipientKey string // public key written to recipients.txt
	IdentityDesc string // human-readable identity location
	Store        *Store // current store (empty if first run)
	IsSSH        bool   // true when vault was bootstrapped with SSH
}

// BootstrapVault ensures .hulak/, identity, and recipients exist,
// then returns the bootstrap result with the current store.
//
// When sshIdentityPath is empty, the age keypair flow runs (EnsureKeypair).
// When sshIdentityPath is set, the SSH flow runs: no identity.txt is created,
// and the SSH public key is written to recipients.txt instead.
func BootstrapVault(projectRoot, sshIdentityPath string) (*BootstrapResult, error) {
	hulakDir := filepath.Join(projectRoot, utils.HiddenProjectName)
	if err := os.MkdirAll(hulakDir, utils.DirPer); err != nil {
		return nil, fmt.Errorf("could not create %s/: %w", utils.HiddenProjectName, err)
	}

	if sshIdentityPath != "" {
		return bootstrapSSH(sshIdentityPath)
	}
	return bootstrapAge()
}

// bootstrapAge is the default flow: generate or load an age keypair.
func bootstrapAge() (*BootstrapResult, error) {
	ageKey, err := EnsureKeypair()
	if err != nil {
		return nil, err
	}

	if err := EnsureRecipientsFile(ageKey.Recipient.String(), utils.Username()); err != nil {
		return nil, err
	}

	store, err := DecryptStore(ageKey.Identity)
	if err != nil {
		return nil, err
	}

	identityPath, _ := IdentityPath()
	return &BootstrapResult{
		RecipientKey: ageKey.Recipient.String(),
		IdentityDesc: identityPath,
		Store:        store,
	}, nil
}

// bootstrapSSH uses an existing SSH private key instead of generating an age keypair.
func bootstrapSSH(sshIdentityPath string) (*BootstrapResult, error) {
	// Reject if an age identity already exists — ambiguous ownership.
	if IdentityExists() {
		idPath, _ := IdentityPath()
		return nil, fmt.Errorf(
			"an age identity already exists at %s\n\n"+
				"Remove it first to use SSH, or init without --ssh-identity",
			idPath,
		)
	}

	// Load identity and derive public key in one read.
	identity, pubKey, err := LoadSSHIdentityWithPubKey(sshIdentityPath)
	if err != nil {
		return nil, err
	}

	if err := EnsureRecipientsFile(pubKey, utils.Username()); err != nil {
		return nil, err
	}

	store, err := DecryptStore(identity)
	if err != nil {
		return nil, err
	}

	return &BootstrapResult{
		RecipientKey: pubKey,
		IdentityDesc: sshIdentityPath,
		Store:        store,
		IsSSH:        true,
	}, nil
}

// EnsureRecipientsFile creates .hulak/recipients.txt with the given public
// key if the file doesn't already exist. Accepts both age (age1...) and SSH
// (ssh-ed25519 ...) public keys. Idempotent — re-running init is a no-op.
func EnsureRecipientsFile(pubKey, name string) error {
	path, err := RecipientsFilePath()
	if err != nil {
		return err
	}
	if utils.FileExists(path) {
		return nil
	}
	return SaveRecipients([]RecipientEntry{
		{Key: pubKey, Name: FormatRecipientName(name)},
	})
}
