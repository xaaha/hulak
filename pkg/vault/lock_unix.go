//go:build !windows

package vault

import (
	"os"

	"golang.org/x/sys/unix"
)

// acquireFileLock takes an exclusive blocking flock on f. Blocks until the
// lock is available; cancellation is via process signal.
func acquireFileLock(f *os.File) error {
	// File descriptors are always small non-negative ints in practice; the
	// uintptr->int conversion is safe.
	return unix.Flock(int(f.Fd()), unix.LOCK_EX) //nolint:gosec // G115
}

// releaseFileLock releases the flock on f. Closing f also releases it,
// so this is best-effort during normal shutdown.
func releaseFileLock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN) //nolint:gosec // G115
}
