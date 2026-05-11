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

// ── vault checks ───────────────────────────────────────────────────────────

// checkIdentityPresent verifies that at least one identity source resolves.
// Check #1.
func checkIdentityPresent() finding {
	_, err := vault.ResolveIdentity()
	if err != nil {
		return finding{
			check:    "identity-present",
			severity: sevError,
			message:  "no identity found (identity.txt, HULAK_MASTER_KEY, or SSH key)",
			fix:      "run 'hulak init' or set HULAK_MASTER_KEY for CI",
		}
	}
	return finding{
		check:    "identity-present",
		severity: sevOk,
		message:  "identity resolves",
	}
}

// checkIdentityMode verifies identity.txt is mode 0600.
// Check #2. Auto-fixable.
func checkIdentityMode() finding {
	path, err := vault.IdentityPath()
	if err != nil || !utils.FileExists(path) {
		return finding{
			check:    "identity-mode",
			severity: sevInfo,
			message:  "identity.txt not present (skipping mode check)",
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		return finding{
			check:    "identity-mode",
			severity: sevError,
			message:  fmt.Sprintf("cannot stat identity file: %v", err),
		}
	}

	perm := info.Mode().Perm()
	if perm == utils.SecretPer {
		return finding{
			check:    "identity-mode",
			severity: sevOk,
			message:  "identity file mode is 0600",
		}
	}

	return finding{
		check:    "identity-mode",
		severity: sevError,
		message:  fmt.Sprintf("identity file mode is %04o (should be 0600)", perm),
		fix:      fmt.Sprintf("chmod 600 %s", path),
		auto: func() error {
			return os.Chmod(path, utils.SecretPer)
		},
	}
}

// checkIdentityNotInGit walks parents of identity file looking for .git/.
// Follows symlinks. Check #3.
func checkIdentityNotInGit() finding {
	path, err := vault.IdentityPath()
	if err != nil || !utils.FileExists(path) {
		return finding{
			check:    "identity-in-git",
			severity: sevInfo,
			message:  "identity.txt not present (skipping git check)",
		}
	}

	// Resolve symlinks to catch the dotfiles repo case
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = path
	}

	// Walk up from both original and resolved paths
	for _, p := range uniquePaths(filepath.Dir(path), filepath.Dir(resolved)) {
		if isInsideGitRepo(p) {
			return finding{
				check:    "identity-in-git",
				severity: sevError,
				message:  fmt.Sprintf("identity file is inside a git repository (%s)", p),
				fix:      "move identity.txt outside any git-tracked directory",
			}
		}
	}

	return finding{
		check:    "identity-in-git",
		severity: sevOk,
		message:  "identity file is not git-tracked",
	}
}

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

// uniquePaths deduplicates two directory paths.
func uniquePaths(a, b string) []string {
	if a == b {
		return []string{a}
	}
	return []string{a, b}
}

// checkIdentityLeakedInProject scans tracked files for AGE-SECRET-KEY- prefix.
// Check #4.
func checkIdentityLeakedInProject() finding {
	projectRoot, ok := utils.FindProjectRoot()
	if !ok {
		return finding{
			check:    "identity-leaked-in-project",
			severity: sevInfo,
			message:  "no project root found (skipping leak scan)",
		}
	}

	leaked := scanForSecretKey(projectRoot)
	if len(leaked) > 0 {
		return finding{
			check:    "identity-leaked-in-project",
			severity: sevError,
			message:  fmt.Sprintf("AGE-SECRET-KEY- found in project file(s): %s", strings.Join(leaked, ", ")),
			fix:      "remove the secret key and run 'hulak secrets rotate-key'",
		}
	}

	return finding{
		check:    "identity-leaked-in-project",
		severity: sevOk,
		message:  "no secret keys found in project files",
	}
}

// scanForSecretKey walks the project looking for files containing AGE-SECRET-KEY-.
// Skips binary-looking files, files > 1 MiB, and the identity file itself
// (which is *supposed* to contain a secret key).
func scanForSecretKey(root string) []string {
	files := trackedFiles(root)
	if files == nil {
		files = walkTextFiles(root)
	}

	// Exclude the identity file — it legitimately contains AGE-SECRET-KEY-.
	identityPath, _ := vault.IdentityPath()
	if identityPath != "" {
		resolved, err := filepath.EvalSymlinks(identityPath)
		if err == nil {
			identityPath = resolved
		}
	}

	var leaked []string
	for _, path := range files {
		abs, _ := filepath.Abs(path)
		resolved, err := filepath.EvalSymlinks(abs)
		if err == nil {
			abs = resolved
		}
		if identityPath != "" && abs == identityPath {
			continue
		}
		if containsSecretKeyPrefix(path) {
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
		f = strings.TrimSpace(f)
		if f != "" {
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
		// Skip hidden dirs (except .hulak)
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

// containsSecretKeyPrefix checks if a file contains the AGE-SECRET-KEY- prefix.
func containsSecretKeyPrefix(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "AGE-SECRET-KEY-") {
			return true
		}
	}
	return false
}

// checkConfigDirMode verifies ~/.config/hulak/ is mode 0700.
// Check #5. Auto-fixable.
func checkConfigDirMode() finding {
	path, err := vault.IdentityPath()
	if err != nil {
		return finding{
			check:    "config-dir-mode",
			severity: sevInfo,
			message:  "config directory not resolved (skipping mode check)",
		}
	}

	configDir := filepath.Dir(path)
	info, err := os.Stat(configDir)
	if err != nil {
		return finding{
			check:    "config-dir-mode",
			severity: sevInfo,
			message:  "config directory does not exist (skipping mode check)",
		}
	}

	perm := info.Mode().Perm()
	if perm == utils.SecretDirPer {
		return finding{
			check:    "config-dir-mode",
			severity: sevOk,
			message:  "config directory mode is 0700",
		}
	}

	return finding{
		check:    "config-dir-mode",
		severity: sevWarn,
		message:  fmt.Sprintf("config directory mode is %04o (should be 0700)", perm),
		fix:      fmt.Sprintf("chmod 700 %s", configDir),
		auto: func() error {
			return os.Chmod(configDir, utils.SecretDirPer)
		},
	}
}

// checkStoreMode verifies store.age is mode 0600.
// Check #6. Auto-fixable.
func checkStoreMode() finding {
	path, err := vault.StorePath()
	if err != nil || !utils.FileExists(path) {
		return finding{
			check:    "store-mode",
			severity: sevInfo,
			message:  "store.age not found (skipping mode check)",
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		return finding{
			check:    "store-mode",
			severity: sevError,
			message:  fmt.Sprintf("cannot stat store.age: %v", err),
		}
	}

	perm := info.Mode().Perm()
	if perm == utils.SecretPer {
		return finding{
			check:    "store-mode",
			severity: sevOk,
			message:  "store.age mode is 0600",
		}
	}

	return finding{
		check:    "store-mode",
		severity: sevError,
		message:  fmt.Sprintf("store.age mode is %04o (should be 0600)", perm),
		fix:      fmt.Sprintf("chmod 600 %s", path),
		auto: func() error {
			return os.Chmod(path, utils.SecretPer)
		},
	}
}

// checkStoreEncrypted verifies store.age starts with the age header.
// Check #7.
func checkStoreEncrypted() finding {
	path, err := vault.StorePath()
	if err != nil || !utils.FileExists(path) {
		return finding{
			check:    "store-encrypted",
			severity: sevInfo,
			message:  "store.age not found (skipping encryption check)",
		}
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

	headerStr := string(header[:n])
	if strings.HasPrefix(headerStr, "age-encryption.org/v1") {
		return finding{
			check:    "store-encrypted",
			severity: sevOk,
			message:  "store.age has valid encryption header",
		}
	}

	return finding{
		check:    "store-encrypted",
		severity: sevError,
		message:  "store.age does not start with age encryption header",
		fix:      "the file may contain plaintext or be corrupt — re-encrypt or restore from backup",
	}
}

// checkStoreDecrypts tries to decrypt store.age with the current identity.
// Check #8.
func checkStoreDecrypts() finding {
	identity, err := vault.ResolveIdentity()
	if err != nil {
		return finding{
			check:    "store-decrypts",
			severity: sevInfo,
			message:  "no identity available (skipping decryption check)",
		}
	}

	_, err = vault.ReadStore(identity)
	if err != nil {
		return finding{
			check:    "store-decrypts",
			severity: sevError,
			message:  fmt.Sprintf("store.age cannot be decrypted: %v", err),
			fix:      "check that your identity matches the recipients list, or run 'hulak secrets rotate-key'",
		}
	}

	return finding{
		check:    "store-decrypts",
		severity: sevOk,
		message:  "store.age decrypts with current identity",
	}
}

// checkRecipientsExist verifies recipients.txt exists.
// Check #9.
func checkRecipientsExist() finding {
	path, err := vault.RecipientsFilePath()
	if err != nil {
		return finding{
			check:    "recipients-exist",
			severity: sevError,
			message:  "recipients.txt not found",
			fix:      "run 'hulak secrets add-recipient' to create it",
		}
	}

	if !utils.FileExists(path) {
		return finding{
			check:    "recipients-exist",
			severity: sevError,
			message:  "recipients.txt does not exist",
			fix:      "run 'hulak secrets add-recipient' to create it",
		}
	}

	return finding{
		check:    "recipients-exist",
		severity: sevOk,
		message:  "recipients.txt exists",
	}
}

// checkRecipientsValid verifies recipients.txt has ≥1 valid entry.
// Check #10.
func checkRecipientsValid() finding {
	path, err := vault.RecipientsFilePath()
	if err != nil || !utils.FileExists(path) {
		return finding{
			check:    "recipients-valid",
			severity: sevInfo,
			message:  "recipients.txt not found (skipping validation)",
		}
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
			fix:      "run 'hulak secrets add-recipient' to add one",
		}
	}

	return finding{
		check:    "recipients-valid",
		severity: sevOk,
		message:  fmt.Sprintf("recipients.txt has %d valid entry(ies)", len(entries)),
	}
}

// checkRecipientsMode verifies recipients.txt is mode 0644 (warn if stricter).
// Check #11. Auto-fixable.
func checkRecipientsMode() finding {
	path, err := vault.RecipientsFilePath()
	if err != nil || !utils.FileExists(path) {
		return finding{
			check:    "recipients-mode",
			severity: sevInfo,
			message:  "recipients.txt not found (skipping mode check)",
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		return finding{
			check:    "recipients-mode",
			severity: sevError,
			message:  fmt.Sprintf("cannot stat recipients.txt: %v", err),
		}
	}

	perm := info.Mode().Perm()
	if perm == utils.FilePer {
		return finding{
			check:    "recipients-mode",
			severity: sevOk,
			message:  "recipients.txt mode is 0644",
		}
	}

	return finding{
		check:    "recipients-mode",
		severity: sevWarn,
		message:  fmt.Sprintf("recipients.txt mode is %04o (expected 0644)", perm),
		fix:      fmt.Sprintf("chmod 644 %s", path),
		auto: func() error {
			return os.Chmod(path, utils.FilePer)
		},
	}
}

// checkRecipientsCommitted checks if recipients.txt is committed or staged in git.
// Check #12.
func checkRecipientsCommitted() finding {
	path, err := vault.RecipientsFilePath()
	if err != nil || !utils.FileExists(path) {
		return finding{
			check:    "recipients-committed",
			severity: sevInfo,
			message:  "recipients.txt not found (skipping commit check)",
		}
	}

	if _, err := exec.LookPath("git"); err != nil {
		return finding{
			check:    "recipients-committed",
			severity: sevInfo,
			message:  "git not available (skipping commit check)",
		}
	}

	// Check if the file is tracked by git (committed or staged)
	projectRoot, _ := utils.FindProjectRoot()
	rel, err := filepath.Rel(projectRoot, path)
	if err != nil {
		rel = path
	}

	cmd := exec.Command("git", "ls-files", "--error-unmatch", rel)
	cmd.Dir = projectRoot
	if err := cmd.Run(); err == nil {
		return finding{
			check:    "recipients-committed",
			severity: sevOk,
			message:  "recipients.txt is committed",
		}
	}

	// Check if staged but not yet committed
	cmd = exec.Command("git", "diff", "--cached", "--name-only", "--", rel)
	cmd.Dir = projectRoot
	output, err := cmd.Output()
	if err == nil && strings.TrimSpace(string(output)) != "" {
		return finding{
			check:    "recipients-committed",
			severity: sevOk,
			message:  "recipients.txt is staged",
		}
	}

	return finding{
		check:    "recipients-committed",
		severity: sevWarn,
		message:  "recipients.txt is not committed or staged",
		fix:      fmt.Sprintf("git add %s", rel),
	}
}

// ── drift + remaining checks ───────────────────────────────────────────────

// checkRecipientDrift counts recipient stanzas in store.age header and
// compares to the number of entries in recipients.txt.
// Check #13.
func checkRecipientDrift() finding {
	storePath, err := vault.StorePath()
	if err != nil || !utils.FileExists(storePath) {
		return finding{
			check:    "recipient-drift",
			severity: sevInfo,
			message:  "store.age not found (skipping drift check)",
		}
	}

	recipientsPath, err := vault.RecipientsFilePath()
	if err != nil || !utils.FileExists(recipientsPath) {
		return finding{
			check:    "recipient-drift",
			severity: sevInfo,
			message:  "recipients.txt not found (skipping drift check)",
		}
	}

	stanzaCount, err := countStanzas(storePath)
	if err != nil {
		return finding{
			check:    "recipient-drift",
			severity: sevInfo,
			message:  fmt.Sprintf("could not parse store.age header: %v", err),
		}
	}

	data, err := os.ReadFile(recipientsPath)
	if err != nil {
		return finding{
			check:    "recipient-drift",
			severity: sevInfo,
			message:  "could not read recipients.txt (skipping drift check)",
		}
	}

	entries, err := vault.ParseRecipientsFileContent(data)
	if err != nil {
		return finding{
			check:    "recipient-drift",
			severity: sevInfo,
			message:  "could not parse recipients.txt (skipping drift check)",
		}
	}

	if stanzaCount == len(entries) {
		return finding{
			check:    "recipient-drift",
			severity: sevOk,
			message:  fmt.Sprintf("recipient count matches store stanzas (%d)", stanzaCount),
		}
	}

	return finding{
		check:    "recipient-drift",
		severity: sevWarn,
		message: fmt.Sprintf(
			"recipients.txt has %d entries but store.age has %d stanzas — re-encryption needed",
			len(entries), stanzaCount,
		),
		fix: "run 'hulak secrets rotate-key' to re-encrypt for all current recipients",
	}
}

// countStanzas counts lines starting with "->" in the age binary header
// (before the "---" MAC line). Returns error if the file is armored or
// doesn't look like a valid age binary file.
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

// checkStoreNotGitignored warns when store.age is gitignored but there are
// multiple recipients (team project).
// Check #14.
func checkStoreNotGitignored() finding {
	recipientsPath, err := vault.RecipientsFilePath()
	if err != nil || !utils.FileExists(recipientsPath) {
		return finding{
			check:    "store-not-gitignored",
			severity: sevInfo,
			message:  "recipients.txt not found (skipping gitignore check)",
		}
	}

	data, err := os.ReadFile(recipientsPath)
	if err != nil {
		return finding{
			check:    "store-not-gitignored",
			severity: sevInfo,
			message:  "could not read recipients.txt (skipping gitignore check)",
		}
	}

	entries, _ := vault.ParseRecipientsFileContent(data)
	if len(entries) <= 1 {
		return finding{
			check:    "store-not-gitignored",
			severity: sevOk,
			message:  "single-recipient project — .gitignore check not applicable",
		}
	}

	// Check if store.age is gitignored
	if _, err := exec.LookPath("git"); err != nil {
		return finding{
			check:    "store-not-gitignored",
			severity: sevInfo,
			message:  "git not available (skipping gitignore check)",
		}
	}

	projectRoot, _ := utils.FindProjectRoot()
	storePath, err := vault.StorePath()
	if err != nil {
		return finding{
			check:    "store-not-gitignored",
			severity: sevInfo,
			message:  "store path not resolved (skipping gitignore check)",
		}
	}

	rel, err := filepath.Rel(projectRoot, storePath)
	if err != nil {
		rel = storePath
	}

	cmd := exec.Command("git", "check-ignore", "-q", rel)
	cmd.Dir = projectRoot
	if cmd.Run() == nil {
		// File IS gitignored
		return finding{
			check:    "store-not-gitignored",
			severity: sevWarn,
			message:  "store.age is in .gitignore — teammates won't be able to decrypt",
			fix:      "remove the store.age entry from .gitignore for team projects",
		}
	}

	return finding{
		check:    "store-not-gitignored",
		severity: sevOk,
		message:  "store.age is not gitignored",
	}
}

// checkLegacyKeyPub detects a lingering .hulak/key.pub file.
// Check #15.
func checkLegacyKeyPub() finding {
	markerPath, err := utils.GetProjectMarker()
	if err != nil {
		return finding{
			check:    "legacy-key-pub",
			severity: sevInfo,
			message:  "project marker not found (skipping legacy check)",
		}
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

	return finding{
		check:    "legacy-key-pub",
		severity: sevOk,
		message:  "no legacy key.pub found",
	}
}

// checkDualBackend detects both env/ and .hulak/store.age existing.
// Check #16.
func checkDualBackend() finding {
	projectRoot, ok := utils.FindProjectRoot()
	if !ok {
		return finding{
			check:    "dual-backend",
			severity: sevInfo,
			message:  "no project root (skipping dual backend check)",
		}
	}

	envDir := filepath.Join(projectRoot, utils.EnvironmentFolder)
	storeDir := filepath.Join(projectRoot, utils.HiddenProjectName, utils.StoreFile)

	hasEnv := utils.DirExists(envDir)
	hasStore := utils.FileExists(storeDir)

	if hasEnv && hasStore {
		return finding{
			check:    "dual-backend",
			severity: sevError,
			message:  "both env/ and .hulak/store.age exist",
			fix:      "verify the migration with 'hulak secrets migrate', then remove env/",
		}
	}

	return finding{
		check:    "dual-backend",
		severity: sevOk,
		message:  "single backend in use",
	}
}

// checkDualIdentity detects when both HULAK_MASTER_KEY and identity.txt exist.
// Check #17.
func checkDualIdentity() finding {
	hasMasterKey := strings.TrimSpace(os.Getenv(utils.MasterKey)) != ""
	hasIdentityFile := vault.IdentityExists()

	if hasMasterKey && hasIdentityFile {
		return finding{
			check:    "dual-identity",
			severity: sevInfo,
			message:  "both HULAK_MASTER_KEY and identity.txt are present — HULAK_MASTER_KEY takes precedence",
		}
	}

	return finding{
		check:    "dual-identity",
		severity: sevOk,
		message:  "single identity source",
	}
}

// checkStoreSize warns if store.age exceeds 1 MiB.
// Check #18.
func checkStoreSize() finding {
	path, err := vault.StorePath()
	if err != nil || !utils.FileExists(path) {
		return finding{
			check:    "store-size",
			severity: sevInfo,
			message:  "store.age not found (skipping size check)",
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		return finding{
			check:    "store-size",
			severity: sevInfo,
			message:  "cannot stat store.age (skipping size check)",
		}
	}

	if info.Size() > vault.MaxStoreSizeWarnBytes {
		sizeMB := float64(info.Size()) / (1 << 20)
		return finding{
			check:    "store-size",
			severity: sevWarn,
			message:  fmt.Sprintf("store.age is %.1f MiB — consider using getFile for large values", sizeMB),
		}
	}

	return finding{
		check:    "store-size",
		severity: sevOk,
		message:  "store.age size is within limits",
	}
}
