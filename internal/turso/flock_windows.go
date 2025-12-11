//go:build windows

package turso

import (
	"math"
	"os"

	"golang.org/x/sys/windows"
)

func lockFileExclusive(f *os.File) (unlock func(), err error) {
	handle := windows.Handle(f.Fd())
	// Lock the entire file (use max values for length)
	var overlapped windows.Overlapped
	err = windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, math.MaxUint32, math.MaxUint32, &overlapped)
	if err != nil {
		return nil, err
	}
	return func() {
		_ = windows.UnlockFileEx(handle, 0, math.MaxUint32, math.MaxUint32, &overlapped)
	}, nil
}
