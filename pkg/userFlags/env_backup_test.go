package userflags

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xaaha/hulak/pkg/utils"
	"github.com/xaaha/hulak/pkg/vault"
)

func TestRunBackup_DefaultPath(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"KEY", "val"}, "global", false); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	if err := runBackup("", false); err != nil {
		t.Fatalf("runBackup: %v", err)
	}

	backupsDir, _ := vault.BackupsDir()
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		t.Fatalf("read backups dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(entries))
	}

	name := entries[0].Name()
	if !strings.HasPrefix(name, backupPrefix) {
		t.Errorf("backup name %q missing prefix %q", name, backupPrefix)
	}

	// Check permissions
	info, _ := os.Stat(filepath.Join(backupsDir, name))
	if perm := info.Mode().Perm(); perm != utils.SecretPer {
		t.Errorf("backup permissions = %o, want %o", perm, utils.SecretPer)
	}
}

func TestRunBackup_OutPath(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"KEY", "val"}, "global", false); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "custom-backup.age")
	if err := runBackup(outPath, false); err != nil {
		t.Fatalf("runBackup --out: %v", err)
	}

	if !utils.FileExists(outPath) {
		t.Fatal("backup file not created at --out path")
	}
}

func TestRunBackup_OutExistingNoForce(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"KEY", "val"}, "global", false); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "existing.age")
	if err := os.WriteFile(outPath, []byte("old"), 0o600); err != nil {
		t.Fatalf("create existing file: %v", err)
	}

	err := runBackup(outPath, false)
	if err == nil {
		t.Fatal("expected error for existing --out without --force")
	}
	if !strings.Contains(err.Error(), "file already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunBackup_OutExistingForce(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"KEY", "val"}, "global", false); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "existing.age")
	if err := os.WriteFile(outPath, []byte("old"), 0o600); err != nil {
		t.Fatalf("create existing file: %v", err)
	}

	if err := runBackup(outPath, true); err != nil {
		t.Fatalf("runBackup --out --force: %v", err)
	}

	data, _ := os.ReadFile(outPath)
	if string(data) == "old" {
		t.Error("file was not overwritten")
	}
}

func TestRunBackup_SameSecondCollision(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"KEY", "val"}, "global", false); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	if err := runBackup("", false); err != nil {
		t.Fatalf("first backup: %v", err)
	}
	if err := runBackup("", false); err != nil {
		t.Fatalf("second backup: %v", err)
	}

	backupsDir, _ := vault.BackupsDir()
	entries, _ := os.ReadDir(backupsDir)
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 backups, got %d", len(entries))
	}

	found := false
	for _, e := range entries {
		if strings.Contains(e.Name(), ".1") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected collision suffix .1 on second backup")
	}
}

func TestRunBackup_NoStore(t *testing.T) {
	setupVaultProject(t)

	err := runBackup("", false)
	if err == nil {
		t.Fatal("expected error when no store exists")
	}
	if !strings.Contains(err.Error(), "nothing to back up") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunBackupList_Empty(t *testing.T) {
	setupVaultProject(t)

	if err := runBackupList(); err != nil {
		t.Fatalf("runBackupList: %v", err)
	}
}

func TestRunBackupList_WithBackups(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"KEY", "val"}, "global", false); err != nil {
		t.Fatalf("seed store: %v", err)
	}
	if err := runBackup("", false); err != nil {
		t.Fatalf("runBackup: %v", err)
	}

	if err := runBackupList(); err != nil {
		t.Fatalf("runBackupList: %v", err)
	}
}

func TestRunBackup_GitignoreUpdated(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"KEY", "val"}, "global", false); err != nil {
		t.Fatalf("seed store: %v", err)
	}
	if err := runBackup("", false); err != nil {
		t.Fatalf("runBackup: %v", err)
	}

	gitignorePath, _ := utils.CreatePath(".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}

	if !strings.Contains(string(data), ".hulak/backups/") {
		t.Error(".gitignore does not contain .hulak/backups/ entry")
	}
}

func TestRunRestore_LatestBackup(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"KEY", "original"}, "global", false); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	if err := runBackup("", false); err != nil {
		t.Fatalf("backup: %v", err)
	}

	if err := runEnvSet([]string{"KEY", "mutated"}, "global", false); err != nil {
		t.Fatalf("mutate store: %v", err)
	}
	if got := readStoredValue(t, "global", "KEY"); got != "mutated" {
		t.Fatalf("store not mutated: %v", got)
	}

	if err := runRestore("", true); err != nil {
		t.Fatalf("restore: %v", err)
	}

	if got := readStoredValue(t, "global", "KEY"); got != "original" {
		t.Errorf("after restore: KEY = %v, want %q", got, "original")
	}
}

func TestRunRestore_SpecificPath(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"KEY", "original"}, "global", false); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "specific.age")
	if err := runBackup(outPath, false); err != nil {
		t.Fatalf("backup: %v", err)
	}

	if err := runEnvSet([]string{"KEY", "changed"}, "global", false); err != nil {
		t.Fatalf("mutate: %v", err)
	}

	if err := runRestore(outPath, true); err != nil {
		t.Fatalf("restore specific: %v", err)
	}

	if got := readStoredValue(t, "global", "KEY"); got != "original" {
		t.Errorf("after restore: KEY = %v, want %q", got, "original")
	}
}

func TestRunRestore_NoBackups(t *testing.T) {
	setupVaultProject(t)

	err := runRestore("", true)
	if err == nil {
		t.Fatal("expected error when no backups exist")
	}
	if !strings.Contains(err.Error(), "no backups found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunRestore_MissingFile(t *testing.T) {
	setupVaultProject(t)

	err := runRestore("/nonexistent/backup.age", true)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunRestore_PlaintextFile(t *testing.T) {
	setupVaultProject(t)

	plainPath := filepath.Join(t.TempDir(), "plain.txt")
	if err := os.WriteFile(plainPath, []byte("KEY=VALUE\n"), 0o600); err != nil {
		t.Fatalf("create plaintext file: %v", err)
	}

	err := runRestore(plainPath, true)
	if err == nil {
		t.Fatal("expected error for plaintext file")
	}
	if !strings.Contains(err.Error(), "decrypt") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunRestore_WrongIdentity(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"KEY", "val"}, "global", false); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "backup.age")
	if err := runBackup(outPath, false); err != nil {
		t.Fatalf("backup: %v", err)
	}

	// Rotate to a new identity so the backup can't be decrypted
	newKey, err := vault.GenerateKeyPair()
	if err != nil {
		t.Fatalf("generate new key: %v", err)
	}
	if err := vault.SetIdentity(newKey.Identity.String()); err != nil {
		t.Fatalf("set new identity: %v", err)
	}

	err = runRestore(outPath, true)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong identity")
	}
}

func TestRunRestore_ReencryptsToCurrentRecipients(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"KEY", "val"}, "global", false); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "backup.age")
	if err := runBackup(outPath, false); err != nil {
		t.Fatalf("backup: %v", err)
	}

	// Add a second recipient
	newKey, _ := vault.GenerateKeyPair()
	if err := runAddRecipient([]string{newKey.Recipient.String()}, "teammate", false); err != nil {
		t.Fatalf("add recipient: %v", err)
	}

	// Restore the old backup (which was encrypted to 1 recipient)
	if err := runRestore(outPath, true); err != nil {
		t.Fatalf("restore: %v", err)
	}

	// The restored store should be decryptable by the new recipient
	store, err := vault.ReadStore(newKey.Identity)
	if err != nil {
		t.Fatalf("new recipient cannot decrypt restored store: %v", err)
	}
	if v := store.GetEnv("global")["KEY"]; v != "val" {
		t.Errorf("restored value = %v, want %q", v, "val")
	}
}

func TestRunRestore_CancelledPrompt(t *testing.T) {
	setupVaultProject(t)

	if err := runEnvSet([]string{"KEY", "original"}, "global", false); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	outPath := filepath.Join(t.TempDir(), "backup.age")
	if err := runBackup(outPath, false); err != nil {
		t.Fatalf("backup: %v", err)
	}

	if err := runEnvSet([]string{"KEY", "mutated"}, "global", false); err != nil {
		t.Fatalf("mutate: %v", err)
	}

	// Simulate "n" answer by piping to stdin
	r, w, _ := os.Pipe()
	if _, err := w.WriteString("n\n"); err != nil {
		t.Fatalf("write to pipe: %v", err)
	}
	w.Close()
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig })

	if err := runRestore(outPath, false); err != nil {
		t.Fatalf("restore should not error on cancel: %v", err)
	}

	// Store should NOT be restored — still mutated
	if got := readStoredValue(t, "global", "KEY"); got != "mutated" {
		t.Errorf("store was restored despite cancel: KEY = %v", got)
	}
}
