//go:build windows

package vault

import (
	"os"

	"golang.org/x/sys/windows"
)

// acquireFileLock takes an exclusive blocking lock on f via LockFileEx.
// Locks the first byte of the file (which need not exist) — the byte range
// is just a synchronization rendezvous, no data is read or written.
func acquireFileLock(f *os.File) error {
	var ol windows.Overlapped
	return windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK,
		0, // reserved
		1, // bytes to lock (low)
		0, // bytes to lock (high)
		&ol,
	)
}

// releaseFileLock releases the LockFileEx lock on f.
func releaseFileLock(f *os.File) error {
	var ol windows.Overlapped
	return windows.UnlockFileEx(
		windows.Handle(f.Fd()),
		0, // reserved
		1, // bytes to unlock (low)
		0, // bytes to unlock (high)
		&ol,
	)
}
