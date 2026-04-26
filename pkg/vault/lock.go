package vault

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xaaha/hulak/pkg/utils"
)

// LockFile is the basename of the cross-process write lock inside .hulak/.
// Each project has exactly one. The file is created on demand and never
// deleted — its presence is harmless, and removing it could race with another
// process holding the lock.
const LockFile = ".lock"

// WithStoreLock acquires an exclusive OS-level file lock on .hulak/.lock,
// invokes fn, and releases the lock when fn returns. Use this around any
// read-modify-write of the encrypted store so concurrent `hulak env set`
// calls cannot lose each other's edits.
//
// Reads do not need this lock: WriteStore renames atomically, so readers
// always observe a single consistent snapshot.
//
// The lock is released automatically if the holding process crashes — the OS
// drops it when the file descriptor closes.
func WithStoreLock(fn func() error) error {
	path, err := lockFilePath()
	if err != nil {
		return err
	}

	// Ensure .hulak/ exists so the lock file (and downstream store ops)
	// can be created on first use, before init has formalized the project.
	if err := os.MkdirAll(filepath.Dir(path), utils.DirPer); err != nil {
		return fmt.Errorf("failed to create %s/: %w", utils.HiddenProjectName, err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, utils.SecretPer)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	defer f.Close()

	if err := acquireFileLock(f); err != nil {
		return fmt.Errorf("failed to acquire store lock: %w", err)
	}
	defer func() {
		// Best-effort release. Closing f also releases the lock, so even
		// if this errors the OS will clean up.
		_ = releaseFileLock(f)
	}()

	return fn()
}

// lockFilePath returns the absolute path to .hulak/.lock in the project root.
func lockFilePath() (string, error) {
	markerPath, err := utils.GetProjectMarker()
	if err != nil {
		return "", err
	}
	return filepath.Join(markerPath, LockFile), nil
}
