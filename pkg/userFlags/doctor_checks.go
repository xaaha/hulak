package userflags

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

// --- permission check helper -------------------------------------------------

// modeCheck describes a file permission check.
type modeCheck struct {
	check    string
	label    string // human name: "identity file", "store.age"
	expected os.FileMode
	failSev  severity
}

// checkFileMode verifies a file's permission mode and returns an auto-fixable
// finding if the mode doesn't match.
func checkFileMode(path string, mc modeCheck) finding {
	if !utils.FileExists(path) {
		return skipFinding(mc.check, mc.label+" not found (skipping mode check)")
	}

	info, err := os.Stat(path)
	if err != nil {
		return finding{
			check:    mc.check,
			severity: sevError,
			message:  fmt.Sprintf("cannot stat %s: %v", mc.label, err),
		}
	}

	perm := info.Mode().Perm()
	if perm == mc.expected {
		return okFinding(mc.check, fmt.Sprintf("%s mode is %04o", mc.label, mc.expected))
	}

	return finding{
		check:    mc.check,
		severity: mc.failSev,
		message:  fmt.Sprintf("%s mode is %04o (should be %04o)", mc.label, perm, mc.expected),
		fix:      fmt.Sprintf("chmod %o %s", mc.expected, path),
		auto:     func() error { return os.Chmod(path, mc.expected) },
	}
}

// --- vault checks ------------------------------------------------------------

// checkIdentityPresent verifies that at least one identity source is available.
func checkIdentityPresent() finding {
	if !vault.HasAnyIdentity() {
		return finding{
			check:    "identity-present",
			severity: sevError,
			message:  "no identity found (identity.txt, HULAK_MASTER_KEY, or SSH key)",
			fix:      "run 'hulak init' or set HULAK_MASTER_KEY for CI",
		}
	}
	return okFinding("identity-present", "identity resolves")
}

// checkIdentityMode verifies identity.txt is mode 0600. Auto-fixable.
func checkIdentityMode() finding {
	path, err := vault.IdentityPath()
	if err != nil {
		return skipFinding("identity-mode", "identity.txt not present (skipping mode check)")
	}
	return checkFileMode(path, modeCheck{
		check: "identity-mode", label: "identity file",
		expected: utils.SecretPer, failSev: sevError,
	})
}

// checkIdentityNotInGit verifies the identity file is not tracked by git.
// Walks parents looking for .git/, then uses git ls-files to distinguish
// "tracked" (sevError) from "in repo but untracked" (sevWarn). The latter
// is common with dotfiles managers (yadm, stow) that place .git/ above
// ~/.config but typically don't track the identity.
func checkIdentityNotInGit() finding {
	path, err := vault.IdentityPath()
	if err != nil || !utils.FileExists(path) {
		return skipFinding("identity-in-git", "identity.txt not present (skipping git check)")
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = path
	}

	type candidate struct {
		dir  string
		file string
	}
	candidates := []candidate{{filepath.Dir(path), path}}
	if resolved != path {
		candidates = append(candidates, candidate{filepath.Dir(resolved), resolved})
	}

	for _, c := range candidates {
		if !isInsideGitRepo(c.dir) {
			continue
		}

		if isFileGitTracked(c.dir, c.file) {
			return finding{
				check:    "identity-in-git",
				severity: sevError,
				message:  fmt.Sprintf("identity file is tracked by git (%s)", c.dir),
				fix:      "remove from tracking with 'git rm --cached <path>' and add to .gitignore",
			}
		}

		return finding{
			check:    "identity-in-git",
			severity: sevWarn,
			message:  fmt.Sprintf("identity file is inside a git repository (%s) but not tracked", c.dir),
			fix:      "verify identity.txt stays in .gitignore, or move it outside the repo",
		}
	}

	return okFinding("identity-in-git", "identity file is not git-tracked")
}

// checkIdentityLeakedInProject scans tracked files for private key markers
// (AGE-SECRET-KEY- and -----BEGIN OPENSSH PRIVATE KEY-----).
func checkIdentityLeakedInProject() finding {
	projectRoot, ok := utils.FindProjectRoot()
	if !ok {
		return skipFinding("identity-leaked-in-project", "no project root found (skipping leak scan)")
	}

	leaked := scanForPrivateKey(projectRoot)
	if len(leaked) > 0 {
		return finding{
			check:    "identity-leaked-in-project",
			severity: sevError,
			message:  fmt.Sprintf("private key found in project file(s): %s", strings.Join(leaked, ", ")),
			fix:      "remove the private key and rotate credentials",
		}
	}

	return okFinding("identity-leaked-in-project", "no private keys found in project files")
}

// checkStoreMode verifies store.age is mode 0600. Auto-fixable.
func checkStoreMode() finding {
	path, err := vault.StorePath()
	if err != nil {
		return skipFinding("store-mode", "store.age not found (skipping mode check)")
	}
	return checkFileMode(path, modeCheck{
		check: "store-mode", label: "store.age",
		expected: utils.SecretPer, failSev: sevError,
	})
}

// checkStoreEncrypted verifies store.age starts with the age header.
func checkStoreEncrypted() finding {
	path, err := vault.StorePath()
	if err != nil || !utils.FileExists(path) {
		return skipFinding("store-encrypted", "store.age not found (skipping encryption check)")
	}

	f, err := os.Open(path)
	if err != nil {
		return finding{
			check:    "store-encrypted",
			severity: sevError,
			message:  fmt.Sprintf("cannot open store.age: %v", err),
		}
	}
	defer f.Close()

	header := make([]byte, 64)
	n, err := f.Read(header)
	if err != nil || n == 0 {
		return finding{
			check:    "store-encrypted",
			severity: sevError,
			message:  "store.age is empty or unreadable",
			fix:      "the store may be corrupt — check if you need to restore from backup",
		}
	}

	if strings.HasPrefix(string(header[:n]), "age-encryption.org/v1") {
		return okFinding("store-encrypted", "store.age has valid encryption header")
	}

	return finding{
		check:    "store-encrypted",
		severity: sevError,
		message:  "store.age does not start with age encryption header",
		fix:      "the file may contain plaintext or be corrupt — re-encrypt or restore from backup",
	}
}

// checkStoreDecrypts tries to decrypt store.age via the multi-source probe.
func checkStoreDecrypts() finding {
	if !vault.HasAnyIdentity() {
		return skipFinding("store-decrypts", "no identity available (skipping decryption check)")
	}

	if _, err := vault.ReadStore(); err != nil {
		return finding{
			check:    "store-decrypts",
			severity: sevError,
			message:  fmt.Sprintf("store.age cannot be decrypted: %v", err),
			fix:      "check that your identity matches the recipients list, or run 'hulak secrets identity rotate'",
		}
	}

	return okFinding("store-decrypts", "store.age decrypts with current identity")
}

// checkRecipientsExist verifies recipients.txt exists.
func checkRecipientsExist() finding {
	path, err := vault.RecipientsFilePath()
	if err != nil || !utils.FileExists(path) {
		return finding{
			check:    "recipients-exist",
			severity: sevError,
			message:  "recipients.txt not found",
			fix:      "run 'hulak secrets identity add-recipient' to create it",
		}
	}
	return okFinding("recipients-exist", "recipients.txt exists")
}

// checkRecipientsValid verifies recipients.txt has ≥1 valid entry.
func checkRecipientsValid() finding {
	path, err := vault.RecipientsFilePath()
	if err != nil || !utils.FileExists(path) {
		return skipFinding("recipients-valid", "recipients.txt not found (skipping validation)")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return finding{
			check:    "recipients-valid",
			severity: sevError,
			message:  fmt.Sprintf("cannot read recipients.txt: %v", err),
		}
	}

	entries, err := vault.ParseRecipientsFileContent(data)
	if err != nil || len(entries) == 0 {
		return finding{
			check:    "recipients-valid",
			severity: sevError,
			message:  "recipients.txt has no valid entries",
			fix:      "run 'hulak secrets identity add-recipient' to add one",
		}
	}

	return okFinding("recipients-valid", fmt.Sprintf("recipients.txt has %d valid entry(ies)", len(entries)))
}

// checkRecipientsCommitted checks if recipients.txt is committed or staged.
func checkRecipientsCommitted() finding {
	path, err := vault.RecipientsFilePath()
	if err != nil || !utils.FileExists(path) {
		return skipFinding("recipients-committed", "recipients.txt not found (skipping commit check)")
	}

	if _, err := exec.LookPath("git"); err != nil {
		return skipFinding("recipients-committed", "git not available (skipping commit check)")
	}

	projectRoot, _ := utils.FindProjectRoot()
	rel, err := filepath.Rel(projectRoot, path)
	if err != nil {
		rel = path
	}

	// Committed?
	cmd := exec.Command("git", "ls-files", "--error-unmatch", rel)
	cmd.Dir = projectRoot
	if cmd.Run() == nil {
		return okFinding("recipients-committed", "recipients.txt is committed")
	}

	// Staged?
	cmd = exec.Command("git", "diff", "--cached", "--name-only", "--", rel)
	cmd.Dir = projectRoot
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) != "" {
		return okFinding("recipients-committed", "recipients.txt is staged")
	}

	return finding{
		check:    "recipients-committed",
		severity: sevWarn,
		message:  "recipients.txt is not committed or staged",
		fix:      fmt.Sprintf("git add %s", rel),
	}
}

// --- drift + misc checks -----------------------------------------------------

// checkRecipientDrift compares recipient stanzas in store.age header to
// the number of entries in recipients.txt.
func checkRecipientDrift() finding {
	storePath, err := vault.StorePath()
	if err != nil || !utils.FileExists(storePath) {
		return skipFinding("recipient-drift", "store.age not found (skipping drift check)")
	}

	recipientsPath, err := vault.RecipientsFilePath()
	if err != nil || !utils.FileExists(recipientsPath) {
		return skipFinding("recipient-drift", "recipients.txt not found (skipping drift check)")
	}

	stanzaCount, err := countStanzas(storePath)
	if err != nil {
		return skipFinding("recipient-drift", fmt.Sprintf("could not parse store.age header: %v", err))
	}

	data, err := os.ReadFile(recipientsPath)
	if err != nil {
		return skipFinding("recipient-drift", "could not read recipients.txt (skipping drift check)")
	}

	entries, err := vault.ParseRecipientsFileContent(data)
	if err != nil {
		return skipFinding("recipient-drift", "could not parse recipients.txt (skipping drift check)")
	}

	if stanzaCount == len(entries) {
		return okFinding("recipient-drift", fmt.Sprintf("recipient count matches store stanzas (%d)", stanzaCount))
	}

	return finding{
		check:    "recipient-drift",
		severity: sevWarn,
		message: fmt.Sprintf(
			"recipients.txt has %d entries but store.age has %d stanzas — re-encryption needed",
			len(entries), stanzaCount,
		),
		fix: "run 'hulak secrets identity rotate' to re-encrypt for all current recipients",
	}
}

// checkStoreNotGitignored warns when store.age is gitignored in a team project.
func checkStoreNotGitignored() finding {
	recipientsPath, err := vault.RecipientsFilePath()
	if err != nil || !utils.FileExists(recipientsPath) {
		return skipFinding("store-not-gitignored", "recipients.txt not found (skipping gitignore check)")
	}

	data, err := os.ReadFile(recipientsPath)
	if err != nil {
		return skipFinding("store-not-gitignored", "could not read recipients.txt (skipping gitignore check)")
	}

	entries, _ := vault.ParseRecipientsFileContent(data)
	if len(entries) <= 1 {
		return okFinding("store-not-gitignored", "single-recipient project — .gitignore check not applicable")
	}

	if _, err := exec.LookPath("git"); err != nil {
		return skipFinding("store-not-gitignored", "git not available (skipping gitignore check)")
	}

	projectRoot, _ := utils.FindProjectRoot()
	storePath, err := vault.StorePath()
	if err != nil {
		return skipFinding("store-not-gitignored", "store path not resolved (skipping gitignore check)")
	}

	rel, err := filepath.Rel(projectRoot, storePath)
	if err != nil {
		rel = storePath
	}

	cmd := exec.Command("git", "check-ignore", "-q", rel)
	cmd.Dir = projectRoot
	if cmd.Run() == nil {
		return finding{
			check:    "store-not-gitignored",
			severity: sevWarn,
			message:  "store.age is in .gitignore — teammates won't be able to decrypt",
			fix:      "remove the store.age entry from .gitignore for team projects",
		}
	}

	return okFinding("store-not-gitignored", "store.age is not gitignored")
}

// checkLegacyKeyPub detects a lingering .hulak/key.pub file.
func checkLegacyKeyPub() finding {
	markerPath, err := utils.GetProjectMarker()
	if err != nil {
		return skipFinding("legacy-key-pub", "project marker not found (skipping legacy check)")
	}

	keyPubPath := filepath.Join(markerPath, "key.pub")
	if utils.FileExists(keyPubPath) {
		return finding{
			check:    "legacy-key-pub",
			severity: sevInfo,
			message:  "legacy .hulak/key.pub found — recipients.txt is now the authoritative recipient list",
			fix:      fmt.Sprintf("rm %s", keyPubPath),
		}
	}

	return okFinding("legacy-key-pub", "no legacy key.pub found")
}

// checkDualBackend detects both env/ and .hulak/store.age existing.
func checkDualBackend() finding {
	projectRoot, ok := utils.FindProjectRoot()
	if !ok {
		return skipFinding("dual-backend", "no project root (skipping dual backend check)")
	}

	envDir := filepath.Join(projectRoot, utils.EnvironmentFolder)
	storeDir := filepath.Join(projectRoot, utils.HiddenProjectName, utils.StoreFile)

	if utils.DirExists(envDir) && utils.FileExists(storeDir) {
		return finding{
			check:    "dual-backend",
			severity: sevError,
			message:  "both env/ and .hulak/store.age exist",
			fix:      "verify the migration with 'hulak secrets migrate', then remove env/",
		}
	}

	return okFinding("dual-backend", "single backend in use")
}

// checkDualIdentity detects when both HULAK_MASTER_KEY and identity.txt exist.
func checkDualIdentity() finding {
	hasMasterKey := strings.TrimSpace(os.Getenv(utils.MasterKey)) != ""
	if hasMasterKey && vault.IdentityExists() {
		return finding{
			check:    "dual-identity",
			severity: sevInfo,
			message:  "both HULAK_MASTER_KEY and identity.txt are present — HULAK_MASTER_KEY takes precedence",
		}
	}
	return okFinding("dual-identity", "single identity source")
}

// checkStoreSize warns if store.age exceeds 1 MiB.
func checkStoreSize() finding {
	path, err := vault.StorePath()
	if err != nil || !utils.FileExists(path) {
		return skipFinding("store-size", "store.age not found (skipping size check)")
	}

	info, err := os.Stat(path)
	if err != nil {
		return skipFinding("store-size", "cannot stat store.age (skipping size check)")
	}

	if info.Size() > vault.MaxStoreSizeWarnBytes {
		sizeMB := float64(info.Size()) / (1 << 20)
		return finding{
			check:    "store-size",
			severity: sevWarn,
			message:  fmt.Sprintf("store.age is %.1f MiB — consider using getFile for large values", sizeMB),
		}
	}

	return okFinding("store-size", "store.age size is within limits")
}

// --- helpers -----------------------------------------------------------------

// isInsideGitRepo walks from dir upward looking for a .git/ directory.
func isInsideGitRepo(dir string) bool {
	for {
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Lstat(gitDir); err == nil && info.IsDir() {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// isFileGitTracked checks if filePath is tracked (committed or staged) by git.
// Returns false if git is unavailable or the path is not inside a valid repo.
func isFileGitTracked(dir, filePath string) bool {
	if _, err := exec.LookPath("git"); err != nil {
		return false
	}
	cmd := exec.Command("git", "ls-files", "--error-unmatch", filePath)
	cmd.Dir = dir
	return cmd.Run() == nil
}

// scanForPrivateKey walks the project looking for files containing private key
// markers (AGE-SECRET-KEY- or -----BEGIN OPENSSH PRIVATE KEY-----).
// Skips files > 1 MiB and the identity file itself.
func scanForPrivateKey(root string) []string {
	files := trackedFiles(root)
	if files == nil {
		files = walkTextFiles(root)
	}

	identityPath, _ := vault.IdentityPath()
	if identityPath != "" {
		if resolved, err := filepath.EvalSymlinks(identityPath); err == nil {
			identityPath = resolved
		}
	}

	var leaked []string
	for _, path := range files {
		abs, _ := filepath.Abs(path)
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			abs = resolved
		}
		if identityPath != "" && abs == identityPath {
			continue
		}
		if containsPrivateKeyMarker(path) {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				rel = path
			}
			leaked = append(leaked, rel)
		}
	}
	return leaked
}

// trackedFiles returns git-tracked files. Returns nil if not in a git repo.
func trackedFiles(root string) []string {
	cmd := exec.Command("git", "ls-files", "-z")
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var files []string
	for _, f := range strings.Split(string(output), "\x00") {
		if f = strings.TrimSpace(f); f != "" {
			files = append(files, filepath.Join(root, f))
		}
	}
	return files
}

// walkTextFiles returns all regular files ≤ 1 MiB under root.
func walkTextFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && d.Name() != utils.HiddenProjectName {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > 1<<20 {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files
}

// containsPrivateKeyMarker checks if a file contains a private key marker
// (AGE-SECRET-KEY- or -----BEGIN OPENSSH PRIVATE KEY-----).
func containsPrivateKeyMarker(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "AGE-SECRET-KEY-") ||
			strings.Contains(line, "-----BEGIN OPENSSH PRIVATE KEY-----") {
			return true
		}
	}
	return false
}

// countStanzas counts "->" lines in the age binary header (before "---").
// hulak always writes binary (non-armored) format; armored files are rejected.
func countStanzas(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return 0, fmt.Errorf("empty file")
	}

	first := scanner.Text()
	if strings.HasPrefix(first, "-----BEGIN AGE ENCRYPTED FILE-----") {
		return 0, fmt.Errorf("armored format not supported for stanza counting")
	}
	if !strings.HasPrefix(first, "age-encryption.org/") {
		return 0, fmt.Errorf("not an age encrypted file")
	}

	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "---") {
			break
		}
		if strings.HasPrefix(line, "-> ") {
			count++
		}
	}
	return count, scanner.Err()
}
