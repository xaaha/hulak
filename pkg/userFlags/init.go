// Package userflags have everything related to user's flags & subcommands
package userflags

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

//go:embed apiOptions.hk.yaml
var embeddedFiles embed.FS

// InitClassicProject sets up the plaintext env/ layout: env/ directory,
// global.env, the apiOptions.hk.yaml example, and an env/ entry in .gitignore.
//
// Refuses to run if .hulak/store.age is present — that's an initialized
// encrypted vault and bolting a parallel plaintext layout next to it would
// create two sources of truth. The store file (not just the .hulak/ dir) is
// the right signal: a partially-failed vault init can leave an empty .hulak/
// behind, and that shouldn't lock the user out of the classic path.
func InitClassicProject() error {
	storePath, _ := vault.StorePath()
	if storePath != "" && utils.FileExists(storePath) {
		return fmt.Errorf(
			"refusing to create plaintext env/ layout: %s exists "+
				"(this project is using the encrypted vault) — "+
				"remove %s/ first if you really want to switch",
			storePath, utils.HiddenProjectName,
		)
	}

	if err := envparser.CreateDefaultEnvs(nil); err != nil {
		return err
	}

	// gitignored path suffix is "/"
	if err := ensureGitignoreEntry(utils.EnvironmentFolder + "/"); err != nil {
		utils.PrintWarningStderr(fmt.Sprintf("could not update .gitignore: %v", err))
	}

	if _, err := writeAPIOptionsExample(); err != nil {
		return err
	}

	utils.PrintSuccessStderr("Done")
	return nil
}

// InitVaultProject sets up the encrypted vault layout: .hulak/store.age,
// .hulak/recipients.txt, and the user's age identity at ~/.config/hulak/identity.txt.
// Also writes the apiOptions.hk.yaml example.
//
// Behaviour notes:
//   - If env/ exists but .hulak/ does not, returns nil after printing a one-line
//     migration nudge — the user should choose between `hulak secrets migrate`
//     (vault) and `hulak init classic` (stay plaintext) rather than have hulak
//     silently bolt a vault next to existing plaintext.
//   - Idempotent: re-running on a project that already has .hulak/ does not
//     regenerate the identity, does not overwrite the store, and does not
//     clobber a customized apiOptions.hk.yaml.
//   - envNames seeds extra empty environment sections in the store (in
//     addition to the always-present "global"). Each name is validated up
//     front; an invalid name aborts before any I/O.
//   - On first-run identity creation, prints the public key and identity path
//     to stderr so the user knows what to back up. Subsequent runs do not
//     repeat this — the identity file already exists at a known location.
func InitVaultProject(envNames []string, sshIdentityPath string) error {
	// Validate input BEFORE touching the filesystem so a typo'd env name
	// can't leave a half-initialised .hulak/ behind.
	for _, name := range envNames {
		if err := utils.ValidateEnvName(name); err != nil {
			return err
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not determine current directory: %w", err)
	}
	hulakDir := filepath.Join(cwd, utils.HiddenProjectName)
	envDir := filepath.Join(cwd, utils.EnvironmentFolder)

	// Pre-existing classic layout, no vault yet → don't surprise the user
	// by creating .hulak/ next to their env/ files. Point at migrate, exit.
	if !utils.DirExists(hulakDir) && utils.DirExists(envDir) {
		utils.PrintInfoStderr(
			"This project uses the legacy env/ layout. " +
				"Run 'hulak secrets migrate' to upgrade, or 'hulak init classic' to stay plaintext.",
		)
		return nil
	}

	storePath, _ := vault.StorePath()
	vaultExists := storePath != "" && utils.FileExists(storePath)

	// Existing vault → try to add the requested identity type as a recipient.
	if vaultExists {
		added, err := maybeAddIdentity(sshIdentityPath)
		if err != nil {
			return err
		}
		if !added {
			utils.PrintSuccessStderr(
				fmt.Sprintf("Vault ready at %s/", utils.HiddenProjectName),
			)
		}
		return ensureStoreSectionsAndExample(envNames)
	}

	// Fresh vault — bootstrap from scratch.
	result, err := bootstrapVault(cwd, sshIdentityPath)
	if err != nil {
		return err
	}

	added := ensureStoreSections(result.store, envNames)

	if err := vault.WriteStoreToRecipients(result.store); err != nil {
		return err
	}

	if _, err := writeAPIOptionsExample(); err != nil {
		return err
	}

	printInitSummary(result)
	printAddedEnvs(added)
	return nil
}

// maybeAddIdentity checks if the requested identity type is missing from the
// vault and adds it as a recipient. Returns true if a new identity was added.
//
//   - sshIdentityPath set → add SSH pub key if not already in recipients
//   - sshIdentityPath empty + no identity.txt → generate age key, add as recipient
//   - otherwise → nothing to add
func maybeAddIdentity(sshIdentityPath string) (bool, error) {
	if sshIdentityPath != "" {
		return maybeAddSSHIdentity(sshIdentityPath)
	}
	if !vault.IdentityExists() {
		return maybeAddAgeIdentity()
	}
	return false, nil
}

// maybeAddSSHIdentity adds an SSH public key as a recipient if not already present.
// Uses ResolveIdentity for decryption (the existing age or SSH key), not the
// new SSH key being added — the store may not be encrypted to it yet.
func maybeAddSSHIdentity(sshIdentityPath string) (bool, error) {
	pubKey, err := vault.DeriveSSHPublicKey(sshIdentityPath)
	if err != nil {
		return false, err
	}

	currentIdentity, err := vault.ResolveIdentity()
	if err != nil {
		return false, fmt.Errorf("failed to load identity: %w", err)
	}

	added, err := vault.AddRecipientAndReencrypt(pubKey, utils.Username(), currentIdentity)
	if err != nil {
		return false, err
	}
	if added {
		utils.PrintSuccessStderr("Added SSH identity as recipient")
		utils.PrintInfoStderr(fmt.Sprintf("  SSH identity:  %s", sshIdentityPath))
		utils.PrintInfoStderr(fmt.Sprintf("  Public key:    %s", pubKey))
	}
	return added, nil
}

// maybeAddAgeIdentity generates a fresh age keypair, writes it to identity.txt,
// and adds the public key as a recipient. Resolves identity BEFORE writing the
// new key — the existing SSH identity is needed to decrypt the store for
// re-encryption. Uses GenerateKeyPair + SetIdentity directly (not EnsureKeypair)
// because EnsureKeypair refuses when store.age exists.
func maybeAddAgeIdentity() (bool, error) {
	// Resolve current identity (SSH) BEFORE creating identity.txt.
	// Once identity.txt exists, ResolveIdentity returns the new age key
	// which can't decrypt the SSH-encrypted store.
	currentIdentity, err := vault.ResolveIdentity()
	if err != nil {
		return false, fmt.Errorf("failed to load identity: %w", err)
	}

	ageKey, err := vault.GenerateKeyPair()
	if err != nil {
		return false, fmt.Errorf("failed to generate age keypair: %w", err)
	}

	if err := vault.SetIdentity(ageKey.Identity.String()); err != nil {
		return false, fmt.Errorf("failed to write identity: %w", err)
	}

	pubKey := ageKey.Recipient.String()
	identityPath, _ := vault.IdentityPath()

	added, err := vault.AddRecipientAndReencrypt(pubKey, utils.Username(), currentIdentity)
	if err != nil {
		return false, err
	}
	if added {
		utils.PrintSuccessStderr("Added age identity as recipient")
		utils.PrintInfoStderr(fmt.Sprintf("  Identity file: %s", identityPath))
		utils.PrintInfoStderr(fmt.Sprintf("  Public key:    %s", pubKey))
		utils.PrintWarningStderr(
			"Back up the identity file — losing it means losing access to the vault.",
		)
	}
	return added, nil
}

// ensureStoreSectionsAndExample handles env section creation and API options
// for the "vault already exists" path.
func ensureStoreSectionsAndExample(envNames []string) error {
	if _, err := writeAPIOptionsExample(); err != nil {
		return err
	}

	if len(envNames) == 0 {
		return nil
	}

	identity, err := vault.ResolveIdentity()
	if err != nil {
		return fmt.Errorf("failed to load identity: %w", err)
	}
	store, err := vault.ReadStore(identity)
	if err != nil {
		return err
	}

	added := ensureStoreSections(store, envNames)
	if len(added) > 0 {
		if err := vault.WriteStoreToRecipients(store); err != nil {
			return err
		}
	}

	printAddedEnvs(added)
	return nil
}

// printInitSummary prints the post-bootstrap summary for fresh vault creation.
func printInitSummary(result *bootstrapResult) {
	utils.PrintSuccessStderr(
		fmt.Sprintf("Initialized vault at %s/", utils.HiddenProjectName),
	)
	utils.PrintInfoStderr(fmt.Sprintf("  Public key:    %s", result.recipientKey))
	utils.PrintInfoStderr(fmt.Sprintf("  Recipients:    %s/%s", utils.HiddenProjectName, utils.RecipientsFile))
	if result.isSSH {
		utils.PrintInfoStderr(fmt.Sprintf("  SSH identity:  %s", result.identityDesc))
		utils.PrintWarningStderr(
			"Your SSH private key is your vault identity — protect it.",
		)
	} else {
		utils.PrintInfoStderr(fmt.Sprintf("  Identity file: %s", result.identityDesc))
		utils.PrintWarningStderr(
			"Back up the identity file — losing it means losing access to the vault.",
		)
	}
}

// printAddedEnvs reports explicitly-requested env sections (excludes "global").
func printAddedEnvs(added []string) {
	var extras []string
	for _, name := range added {
		if name != utils.DefaultEnvVal {
			extras = append(extras, name)
		}
	}
	if len(extras) > 0 {
		utils.PrintInfoStderr("Added envs: " + strings.Join(extras, ", "))
	}
}

// ensureStoreSections makes sure every env in `names` (plus the default
// "global") exists as an empty section in store. Returns the names that were
// newly created so the caller can report them. Names that match the default
// case-insensitively are folded into "global" rather than creating duplicates.
func ensureStoreSections(store *vault.Store, names []string) []string {
	var added []string
	if store.EnsureSection(utils.DefaultEnvVal) {
		added = append(added, utils.DefaultEnvVal)
	}
	for _, name := range names {
		if strings.EqualFold(name, utils.DefaultEnvVal) {
			continue
		}
		if store.EnsureSection(name) {
			added = append(added, name)
		}
	}
	return added
}

// ensureRecipientsFile creates .hulak/recipients.txt with the given public
// key if the file doesn't already exist. Accepts both age (age1...) and SSH
// (ssh-ed25519 ...) public keys. Idempotent — re-running init is a no-op.
func ensureRecipientsFile(pubKey, name string) error {
	path, err := vault.RecipientsFilePath()
	if err != nil {
		return err
	}
	if utils.FileExists(path) {
		return nil
	}
	return vault.SaveRecipients([]vault.RecipientEntry{
		{Key: pubKey, Name: vault.FormatRecipientName(name)},
	})
}

// writeAPIOptionsExample writes the embedded apiOptions.hk.yaml to the project
// root if absent. Returns whether the file was newly written. Skips with a
// "kept existing" warning if the user has customized it — re-running init
// must never clobber edited content.
func writeAPIOptionsExample() (bool, error) {
	root, err := utils.CreatePath(utils.APIOptions)
	if err != nil {
		return false, err
	}
	if utils.FileExists(root) {
		utils.PrintWarningStderr(
			fmt.Sprintf("Kept existing '%s' (delete it to regenerate)", utils.APIOptions),
		)
		return false, nil
	}

	content, err := embeddedFiles.ReadFile(utils.APIOptions)
	if err != nil {
		return false, err
	}

	if err := os.WriteFile(root, content, utils.FilePer); err != nil {
		return false, fmt.Errorf("error on writing '%s' file: %w", utils.APIOptions, err)
	}

	utils.PrintSuccessStderr(fmt.Sprintf("Created '%s'", utils.APIOptions))
	return true, nil
}

// ensureGitignoreEntry adds entry to .gitignore if not already present.
func ensureGitignoreEntry(entry string) error {
	gitignorePath, err := utils.CreatePath(".gitignore")
	if err != nil {
		return fmt.Errorf("could not resolve .gitignore path: %w", err)
	}

	if utils.FileExists(gitignorePath) {
		file, err := os.Open(gitignorePath)
		if err != nil {
			return fmt.Errorf("could not read .gitignore: %w", err)
		}
		defer file.Close()

		bare := strings.TrimRight(entry, "/")
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == entry || line == bare {
				return nil
			}
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading .gitignore: %w", err)
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, utils.FilePer)
	if err != nil {
		return fmt.Errorf("could not open .gitignore for writing: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("could not stat .gitignore: %w", err)
	}

	prefix := ""
	if info.Size() > 0 {
		content, err := os.ReadFile(gitignorePath)
		if err != nil {
			return fmt.Errorf("could not read .gitignore: %w", err)
		}
		if len(content) > 0 && content[len(content)-1] != '\n' {
			prefix = "\n"
		}
	}

	if _, err := fmt.Fprintf(f, "%s%s\n", prefix, entry); err != nil {
		return fmt.Errorf("could not write to .gitignore: %w", err)
	}

	return nil
}
